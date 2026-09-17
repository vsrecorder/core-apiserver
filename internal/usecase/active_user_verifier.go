package usecase

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	// activeUserCacheTTL は「登録済みで退会していない」と確認した結果を保持する時間。
	// 認証が必要なリクエストのたびに users を引かないためのもので、退会がこの時間だけ
	// 遅れて効くことは許容する(webapp 側の退会チェックは30分間隔)。
	activeUserCacheTTL = time.Minute

	// activeUserCacheMaxEntries は保持するユーザー数の上限。超えたら期限切れを掃除し、
	// それでも超えていれば全て捨てる(プロセス内メモリを無制限に増やさないための単純な方式)。
	activeUserCacheMaxEntries = 10000
)

/*
 * ActiveUserVerifier は、認証トークンの uid が登録済みで退会していないユーザーかを答える
 * (authentication.UserVerifier の実装)。
 *
 * トークンは署名と有効期限しか保証しない。退会(論理削除)後も Firebase 側のアカウントが
 * 残ることがあり、そのトークンで書き込めると退会後にデータが作られ、
 * 「退会したユーザのデータを残さない」が崩れる。未登録(POST /users 前)の uid も同様に通さない。
 *
 * 否定の結果(未登録・退会済み)は保持しない。登録直後のリクエストが古い否定結果で
 * 弾かれないようにするため。
 */
type ActiveUserVerifier struct {
	userRepository repository.UserInterface

	mu sync.Mutex
	// expiresAt は uid ごとの確認結果の有効期限(有効なユーザーだけを入れる)。
	expiresAt map[string]time.Time
}

func NewActiveUserVerifier(userRepository repository.UserInterface) *ActiveUserVerifier {
	return &ActiveUserVerifier{
		userRepository: userRepository,
		expiresAt:      map[string]time.Time{},
	}
}

func (v *ActiveUserVerifier) IsActiveUser(ctx context.Context, uid string) (bool, error) {
	now := timeNow()

	if v.isCached(uid, now) {
		return true, nil
	}

	// FindById は論理削除済みのユーザーを返さないため、「見つからない」が未登録と退会済みの両方を表す。
	if _, err := v.userRepository.FindById(ctx, uid); err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			return false, nil
		}

		logError(ctx, err)
		return false, err
	}

	v.remember(uid, now)

	return true, nil
}

func (v *ActiveUserVerifier) isCached(uid string, now time.Time) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	expiresAt, ok := v.expiresAt[uid]

	return ok && now.Before(expiresAt)
}

func (v *ActiveUserVerifier) remember(uid string, now time.Time) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if len(v.expiresAt) >= activeUserCacheMaxEntries {
		for k, expiresAt := range v.expiresAt {
			if !now.Before(expiresAt) {
				delete(v.expiresAt, k)
			}
		}
		if len(v.expiresAt) >= activeUserCacheMaxEntries {
			v.expiresAt = map[string]time.Time{}
		}
	}

	v.expiresAt[uid] = now.Add(activeUserCacheTTL)
}
