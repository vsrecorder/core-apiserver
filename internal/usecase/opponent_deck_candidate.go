package usecase

import (
	"context"
	"sync"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	// OpponentDeckCandidateWindow は候補の集計対象にする期間(対戦結果の作成日時から遡る)。
	// 環境は数か月で入れ替わるため、全期間で数えると過去の有力デッキが上位に残り続ける。
	// 以前の webapp が「直近100件の対戦」から候補を作っていたのと同じく、最近のものに寄せる。
	OpponentDeckCandidateWindow = 90 * 24 * time.Hour

	// opponentDeckCandidateCacheTTL は集計結果を保持する時間。候補は全ユーザーの対戦結果を
	// GROUP BY する集計で、対戦結果の作成フォームを開くたびに走らせる必要はない。
	opponentDeckCandidateCacheTTL = 10 * time.Minute

	// opponentDeckCandidateCacheSize は保持する候補の数。一覧の limit の上限(helper.MaxLimit)と
	// 同じ 100 にし、リクエストの limit はこの中から切り出す。
	opponentDeckCandidateCacheSize = 100
)

type OpponentDeckCandidateInterface interface {
	// FindOpponentDeckCandidates は相手デッキの入力候補を出現回数の多い順に limit 件返す。
	// 全ユーザーの対戦結果から作る(自分の対戦がまだ無いユーザー向け)。
	FindOpponentDeckCandidates(
		ctx context.Context,
		limit int,
	) ([]*entity.OpponentDeckCandidate, error)
}

type OpponentDeckCandidate struct {
	repository repository.OpponentDeckCandidateInterface

	mu sync.Mutex
	// cached は直近の集計結果(出現回数順・最大 opponentDeckCandidateCacheSize 件)。
	cached    []*entity.OpponentDeckCandidate
	expiresAt time.Time
}

func NewOpponentDeckCandidate(
	repository repository.OpponentDeckCandidateInterface,
) OpponentDeckCandidateInterface {
	return &OpponentDeckCandidate{repository: repository}
}

func (u *OpponentDeckCandidate) FindOpponentDeckCandidates(
	ctx context.Context,
	limit int,
) ([]*entity.OpponentDeckCandidate, error) {
	// 集計中に同じ問い合わせが重ならないよう、更新はロックの中で行う(数ミリ秒の集計で、
	// 10分に1回しか走らない)。
	u.mu.Lock()
	defer u.mu.Unlock()

	now := timeNow()
	if u.cached == nil || !now.Before(u.expiresAt) {
		candidates, err := u.repository.FindOpponentDeckCandidates(ctx, now.Add(-OpponentDeckCandidateWindow), opponentDeckCandidateCacheSize)
		if err != nil {
			logError(ctx, err)
			return nil, err
		}

		u.cached = candidates
		u.expiresAt = now.Add(opponentDeckCandidateCacheTTL)
	}

	if limit > len(u.cached) {
		limit = len(u.cached)
	}

	// 呼び出し側が並び替えても保持している結果に影響しないよう、切り出したコピーを返す。
	ret := make([]*entity.OpponentDeckCandidate, limit)
	copy(ret, u.cached[:limit])

	return ret, nil
}
