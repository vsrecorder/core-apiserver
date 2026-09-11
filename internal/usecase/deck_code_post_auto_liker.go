package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// ErrAutoLikerUserIdEmpty はいいねを押す公式アカウントが指定されていない場合に返す。
// 誰のいいねか決まらないまま書き込むことを防ぐ(バッチの設定漏れの検知を兼ねる)。
var ErrAutoLikerUserIdEmpty = errors.New("auto liker user id is empty")

// DeckCodePostAutoLikerInterface は、みんなの公開デッキへ投稿されたデッキに運営の公式
// アカウントで自動的にいいねを付ける。
//
// 公開しても反応が無いと投稿は続かないため、運営が最初の1つを必ず付ける。押したいいねは
// 投稿者への日次まとめ通知には出さない(自動で全投稿に付くため、毎日「公式がいいねしました」が
// 届いて人が押したいいねの知らせが埋もれる)。除外は FindLikeDigests の excludeLikerUserId で行う。
type DeckCodePostAutoLikerInterface interface {
	// LikeUnliked は公式アカウントがまだいいねしていない公開中の投稿へいいねを付け、その件数を返す。
	// ownerUserId を指定するとその投稿者の投稿だけを対象にする(空なら全員)。
	// publishedFrom がゼロ値でなければ、その日時以降に公開された投稿だけを対象にする。
	// limit は1回の実行で押す上限(0以下なら上限なし)。
	// dryRun のときはいいねを付けず、対象の件数だけ返す。
	//
	// 既にいいね済みの投稿は対象に入らず、押す側も ON CONFLICT DO NOTHING なので、
	// 何度実行しても同じ投稿へ二重にいいねは付かない。
	LikeUnliked(
		ctx context.Context,
		ownerUserId string,
		publishedFrom time.Time,
		limit int,
		dryRun bool,
	) (int, error)
}

type DeckCodePostAutoLiker struct {
	postRepo repository.DeckCodePostInterface
	// likerUserId はいいねを押す運営の公式アカウント。
	likerUserId string
}

func NewDeckCodePostAutoLiker(
	postRepo repository.DeckCodePostInterface,
	likerUserId string,
) DeckCodePostAutoLikerInterface {
	return &DeckCodePostAutoLiker{postRepo, likerUserId}
}

func (u *DeckCodePostAutoLiker) LikeUnliked(
	ctx context.Context,
	ownerUserId string,
	publishedFrom time.Time,
	limit int,
	dryRun bool,
) (int, error) {
	// 主体が決まらないまま書き込まないよう、対象を引く前に止める。
	if u.likerUserId == "" {
		logError(ctx, ErrAutoLikerUserIdEmpty)
		return 0, ErrAutoLikerUserIdEmpty
	}

	posts, err := u.postRepo.FindActiveNotLikedBy(ctx, u.likerUserId, ownerUserId, publishedFrom, limit)
	if err != nil {
		logError(ctx, err)
		return 0, err
	}

	count := 0
	for _, post := range posts {
		// どの投稿へ押したかは cron のログから追えるようにする(いいねは通知に出さないため、
		// 取り消し依頼などの問い合わせで経緯を確かめる手段がログしかない)。
		slog.InfoContext(ctx, "liking deck code post",
			slog.String("post_id", post.ID),
			slog.String("owner_user_id", post.UserId),
			slog.Bool("dry_run", dryRun),
		)

		if dryRun {
			count++
			continue
		}

		if err := u.postRepo.Like(ctx, post.ID, u.likerUserId, timeNow()); err != nil {
			logError(ctx, err)
			// 途中で止めても、次の実行が残りを拾う(未いいねの投稿だけを対象にするため)。
			return count, err
		}
		count++
	}

	return count, nil
}
