package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	// NotificationCategoryWeeklyReport は週次バトルレポートの通知カテゴリ。
	// webapp 側の NotificationCategory / ベルのアイコン対応表と一致させる。
	NotificationCategoryWeeklyReport = "weekly_report"

	// weeklyReportTitle は週次レポート通知の見出し。固定文言にしておく。
	weeklyReportTitle = "先週のバトルレポートができました"

	// weeklyReportLinkFormat は通知のリンク先。%s には対象週の月曜(YYYY-MM-DD)が入り、
	// webapp の /users/report/weeks/[week] に対応する。
	weeklyReportLinkFormat = "/users/report/weeks/%s"

)

type WeeklyReportNotifierInterface interface {
	// NotifyUser は week(週内の任意日 YYYY-MM-DD)が属する週のユーザーの戦績を集計し、
	// 1戦以上あればその週のバトルレポートへ誘導するアプリ内通知を1件作成する。
	// 同じ週の通知が既にあれば作らない(冪等)。0戦の週は「配当」が無いので送らない。
	// dryRun=true の場合は作成対象かどうかだけを返し、通知は作らない。
	// 戻り値の bool は「作成した(dryRunなら作成対象だった)」かどうか。
	NotifyUser(ctx context.Context, userId string, week string, dryRun bool) (bool, error)
}

type WeeklyReportNotifier struct {
	userStatRepo      repository.UserStatInterface
	deckUsageStatRepo repository.DeckUsageStatInterface
	notificationRepo  repository.NotificationInterface
	pushNotifier      PushNotifierInterface
}

func NewWeeklyReportNotifier(
	userStatRepo repository.UserStatInterface,
	deckUsageStatRepo repository.DeckUsageStatInterface,
	notificationRepo repository.NotificationInterface,
	pushNotifier PushNotifierInterface,
) WeeklyReportNotifierInterface {
	return &WeeklyReportNotifier{userStatRepo, deckUsageStatRepo, notificationRepo, pushNotifier}
}

func (u *WeeklyReportNotifier) NotifyUser(ctx context.Context, userId string, week string, dryRun bool) (bool, error) {
	now := timeNow()

	fromDate, toDate, err := weekRange(week, now.Local())
	if err != nil {
		logError(ctx, err)
		return false, err
	}

	linkUrl := fmt.Sprintf(weeklyReportLinkFormat, fromDate.Format(weekDateLayout))

	// 全レギュレーション合算(regulationId=0)。webapp のバトルレポートも絞らずに出す
	stat, err := u.userStatRepo.FindUserStat(ctx, userId, fromDate, toDate, 0)
	if err != nil {
		logError(ctx, err)
		if errors.Is(err, apperror.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if stat == nil || stat.TotalMatches == 0 {
		// その週に対戦が無ければレポートも空になる。配当の無い通知は送らない
		return false, nil
	}

	already, err := u.alreadyNotified(ctx, userId, linkUrl)
	if err != nil {
		logError(ctx, err)
		return false, err
	}
	if already {
		return false, nil
	}

	if dryRun {
		return true, nil
	}

	id, err := generateId()
	if err != nil {
		logError(ctx, err)
		return false, err
	}

	notification := entity.NewNotification(
		id,
		now,
		userId,
		NotificationCategoryWeeklyReport,
		weeklyReportTitle,
		weeklyReportBody(stat, u.topDeckName(ctx, userId, fromDate, toDate)),
		linkUrl,
	)

	if err := u.notificationRepo.Save(ctx, notification); err != nil {
		logError(ctx, err)
		return false, err
	}

	// B-1: アプリ内通知を作った上で push を撃つ(D2)。「記録の配当」を来ていない人にも押し出す。
	// push の失敗で通知作成は巻き戻さない
	if _, err := u.pushNotifier.Deliver(ctx, notification, PushCampaignWeeklyReport); err != nil {
		logWarn(ctx, err)
	}

	return true, nil
}

// alreadyNotified は、同じ週(=同じリンク先)のレポート通知を既に作っているかを返す。
// 週をキーにしているため、cron の多重起動でも -week 指定の再実行でも二重には作らない。
// 直近N件ではなく全期間で見るのは、-week で古い週をバックフィルしたときに、その週の通知が
// 直近から外れていて二重に作られるのを防ぐため。
func (u *WeeklyReportNotifier) alreadyNotified(ctx context.Context, userId string, linkUrl string) (bool, error) {
	exists, err := u.notificationRepo.ExistsByUserIdAndCategoryAndLinkUrl(ctx, userId, NotificationCategoryWeeklyReport, linkUrl)
	if err != nil {
		logError(ctx, err)
		return false, err
	}

	return exists, nil
}

// topDeckName はその週に最も多く使ったデッキ(相棒デッキ)の名前を返す。
// 集計に失敗しても通知本体は出したいので、失敗時は警告だけ残して空文字を返す。
func (u *WeeklyReportNotifier) topDeckName(ctx context.Context, userId string, fromDate, toDate time.Time) string {
	stat, err := u.deckUsageStatRepo.FindDeckUsageStat(ctx, userId, fromDate, toDate, 0)
	if err != nil {
		logWarn(ctx, err)
		return ""
	}
	if stat == nil {
		return ""
	}

	var top *entity.DeckUsage
	for _, deck := range stat.Decks {
		if top == nil || deck.Count > top.Count {
			top = deck
		}
	}
	if top == nil {
		return ""
	}

	return strings.TrimSpace(top.Name)
}

// weeklyReportBody は通知本文を組み立てる。
// 例: 先週は12戦 8勝4敗（勝率 66.7%）。相棒デッキは『リザードンex』でした。
// 引き分けは試合数と勝敗の合計が食い違って見えないよう、ある週だけ内訳に添える。
// デッキ名が取れない(デッキ未登録など)場合は相棒デッキの文を省く。
func weeklyReportBody(stat *entity.UserStat, deckName string) string {
	record := fmt.Sprintf("%d勝%d敗", stat.Wins, stat.Losses)
	if draws := stat.TotalMatches - stat.Wins - stat.Losses; draws > 0 {
		record += fmt.Sprintf("%d分", draws)
	}

	body := fmt.Sprintf("先週は%d戦 %s（勝率 %.1f%%）。", stat.TotalMatches, record, stat.WinRate*100)
	if deckName != "" {
		body += fmt.Sprintf("相棒デッキは『%s』でした。", deckName)
	}

	return body
}
