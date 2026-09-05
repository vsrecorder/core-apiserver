package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	// NotificationCategoryLike はみんなの公開デッキの投稿へのいいね(日次のまとめ)。
	// webapp の NotificationCategory と一致させる。
	NotificationCategoryLike = "like"

	// PushCampaignDeckCodePostLike は push 配達ログ(push_deliveries)に残す施策名。
	PushCampaignDeckCodePostLike = "deck_code_post_like"

	deckCodePostLikeNotificationTitle = "いいねが届きました"
)

// DeckCodePostLikeNotifierInterface は、みんなの公開デッキの投稿へ付いたいいねを
// 1日1回・投稿ごとに1通にまとめて投稿者へ通知する。
//
// いいねのたびに通知すると、人気の投稿ほど通知が連打になるため日次でまとめる。
// 文面は定型で、いいねした人が書いた文章は含まれない(ユーザ間のメッセージにならないようにする)。
type DeckCodePostLikeNotifierInterface interface {
	// NotifyDay は day(暦日、ローカル時刻)に付いたいいねをまとめて通知し、作成した通知数を返す。
	// userId を指定すると、その投稿者宛ての通知だけを対象にする(空なら全員)。
	// dryRun のときは通知を作らず、対象の件数だけ返す。
	NotifyDay(ctx context.Context, day time.Time, userId string, dryRun bool) (int, error)
}

type DeckCodePostLikeNotifier struct {
	postRepo         repository.DeckCodePostInterface
	notificationRepo repository.NotificationInterface
	pushNotifier     PushNotifierInterface
}

func NewDeckCodePostLikeNotifier(
	postRepo repository.DeckCodePostInterface,
	notificationRepo repository.NotificationInterface,
	pushNotifier PushNotifierInterface,
) DeckCodePostLikeNotifierInterface {
	return &DeckCodePostLikeNotifier{postRepo, notificationRepo, pushNotifier}
}

// deckCodePostLikeNotificationLinkUrl は通知のリンク先。
// 同じ投稿・同じ日の通知を二重に作らないための重複判定キー
// (ExistsByUserIdAndCategoryAndLinkUrl)も兼ねるため、対象日をクエリに含める。
// 個別ページはこのクエリを無視する。
func deckCodePostLikeNotificationLinkUrl(postId string, day time.Time) string {
	return fmt.Sprintf("/shared_decks/%s?d=%s", postId, day.Format(time.DateOnly))
}

// deckCodePostLikeNotificationBody は「◯◯さんほかN人が「デッキ名」にいいねしました」の定型文。
func deckCodePostLikeNotificationBody(digest *entity.DeckCodePostLikeDigest) string {
	if digest.LikeCount <= 1 {
		return fmt.Sprintf("%sさんが「%s」にいいねしました", digest.LatestLikerName, digest.DeckName)
	}

	return fmt.Sprintf("%sさんほか%d人が「%s」にいいねしました", digest.LatestLikerName, digest.LikeCount-1, digest.DeckName)
}

// dayRange は day の暦日を [0時, 翌日0時) の半開区間(ローカル時刻)で返す。
func dayRange(day time.Time) (time.Time, time.Time) {
	y, m, d := day.Date()
	from := time.Date(y, m, d, 0, 0, 0, 0, time.Local)

	return from, from.AddDate(0, 0, 1)
}

func (u *DeckCodePostLikeNotifier) NotifyDay(ctx context.Context, day time.Time, userId string, dryRun bool) (int, error) {
	from, to := dayRange(day)

	digests, err := u.postRepo.FindLikeDigests(ctx, from, to)
	if err != nil {
		logError(ctx, err)
		return 0, err
	}

	count := 0
	for _, digest := range digests {
		if userId != "" && digest.OwnerUserId != userId {
			continue
		}

		linkUrl := deckCodePostLikeNotificationLinkUrl(digest.PostId, from)

		// 多重起動や再実行で同じ通知を二度作らない
		exists, err := u.notificationRepo.ExistsByUserIdAndCategoryAndLinkUrl(ctx, digest.OwnerUserId, NotificationCategoryLike, linkUrl)
		if err != nil {
			logError(ctx, err)
			return count, err
		}
		if exists {
			continue
		}

		if dryRun {
			count++
			continue
		}

		id, err := generateId()
		if err != nil {
			logError(ctx, err)
			return count, err
		}

		notification := entity.NewNotification(
			id,
			timeNow(),
			digest.OwnerUserId,
			NotificationCategoryLike,
			deckCodePostLikeNotificationTitle,
			deckCodePostLikeNotificationBody(digest),
			linkUrl,
		)

		if err := u.notificationRepo.Save(ctx, notification); err != nil {
			logError(ctx, err)
			return count, err
		}
		count++

		// push は「アプリ内通知の配達手段」なので、送れなくても通知は残す
		if u.pushNotifier != nil {
			if _, err := u.pushNotifier.Deliver(ctx, notification, PushCampaignDeckCodePostLike); err != nil {
				logWarn(ctx, err)
			}
		}
	}

	return count, nil
}
