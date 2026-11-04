package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	// NotificationCategoryMonthlyReport は月次バトルレポートの通知カテゴリ。
	// webapp 側の NotificationCategory / ベルのアイコン対応表と一致させる。
	NotificationCategoryMonthlyReport = "monthly_report"

	// NotificationCategoryEnvironmentReport は環境別バトルレポートの通知カテゴリ。
	NotificationCategoryEnvironmentReport = "environment_report"

	// monthlyReportLinkFormat は通知のリンク先。%s には対象月(YYYY-MM)が入り、
	// webapp の /users/report/[yearMonth] に対応する。
	monthlyReportLinkFormat = "/users/report/%s"

	// environmentReportLinkFormat は通知のリンク先。%s には環境ID(m6a など)が入り、
	// webapp の /users/report/environments/[id] に対応する。
	environmentReportLinkFormat = "/users/report/environments/%s"

	yearMonthLayout = "2006-01"
)

type PeriodReportNotifierInterface interface {
	// NotifyMonthly は yearMonth(YYYY-MM)の月について、その月に1戦以上記録したユーザーへ
	// 「10月のバトルレポートができました」の通知を1件作成し、push 購読者には push も送る。
	// 0戦なら何もしない(週次と違い、環境ニュースの代替は無い)。同じ月の通知が既にあれば作らない(冪等)。
	// dryRun=true の場合は作成対象かどうかだけを返し、通知は作らない。
	NotifyMonthly(ctx context.Context, userId string, yearMonth string, dryRun bool) (bool, error)

	// NotifyEnvironment は environmentId の環境について、その環境で1戦以上記録したユーザーへ
	// 「『○○』環境のバトルレポートができました」の通知を1件作成し、push 購読者には push も送る。
	// 集計は webapp の環境別レポートと同じく、開催日と環境がズレる公式イベントの例外を含める。
	// 0戦なら何もしない。同じ環境の通知が既にあれば作らない(冪等)。
	NotifyEnvironment(ctx context.Context, userId string, environmentId string, dryRun bool) (bool, error)
}

// periodReportTarget は通知1種類ぶんの材料。対象者ごとに変わらないので、
// 1回のバッチの中では月・環境ごとに1度だけ組み立てる。
type periodReportTarget struct {
	category string
	campaign string
	title    string
	// prefix は本文の書き出し(「10月は」「この環境では」)
	prefix  string
	linkUrl string
	period  repository.StatPeriod
}

type PeriodReportNotifier struct {
	userStatRepo                 repository.UserStatInterface
	deckUsageStatRepo            repository.DeckUsageStatInterface
	notificationRepo             repository.NotificationInterface
	pushNotifier                 PushNotifierInterface
	environmentRepo              repository.EnvironmentInterface
	officialEventEnvironmentRepo repository.OfficialEventEnvironmentInterface

	// environmentTargets は環境IDごとの材料。環境の期間は例外テーブルの全件読みを伴うため、
	// 対象者の数だけ繰り返さないよう1回のバッチ内で持ち回る。
	environmentTargets map[string]*periodReportTarget
}

func NewPeriodReportNotifier(
	userStatRepo repository.UserStatInterface,
	deckUsageStatRepo repository.DeckUsageStatInterface,
	notificationRepo repository.NotificationInterface,
	pushNotifier PushNotifierInterface,
	environmentRepo repository.EnvironmentInterface,
	officialEventEnvironmentRepo repository.OfficialEventEnvironmentInterface,
) PeriodReportNotifierInterface {
	return &PeriodReportNotifier{
		userStatRepo:                 userStatRepo,
		deckUsageStatRepo:            deckUsageStatRepo,
		notificationRepo:             notificationRepo,
		pushNotifier:                 pushNotifier,
		environmentRepo:              environmentRepo,
		officialEventEnvironmentRepo: officialEventEnvironmentRepo,
		environmentTargets:           map[string]*periodReportTarget{},
	}
}

func (u *PeriodReportNotifier) NotifyMonthly(ctx context.Context, userId string, yearMonth string, dryRun bool) (bool, error) {
	t, err := time.Parse(yearMonthLayout, yearMonth)
	if err != nil {
		logError(ctx, err)
		return false, err
	}

	// 期間の取り方は API の year_month(usecase.UserStat.GetUserStat)と揃える。
	// ずれると、通知の数字と開いたレポートの数字が食い違う。
	from := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
	to := from.AddDate(0, 1, 0)

	return u.notify(ctx, userId, &periodReportTarget{
		category: NotificationCategoryMonthlyReport,
		campaign: PushCampaignMonthlyReport,
		title:    fmt.Sprintf("%d月のバトルレポートができました", int(t.Month())),
		prefix:   fmt.Sprintf("%d月は", int(t.Month())),
		linkUrl:  fmt.Sprintf(monthlyReportLinkFormat, from.Format(yearMonthLayout)),
		period:   statPeriodOf(from, to),
	}, dryRun)
}

func (u *PeriodReportNotifier) NotifyEnvironment(ctx context.Context, userId string, environmentId string, dryRun bool) (bool, error) {
	target, err := u.environmentTarget(ctx, environmentId)
	if err != nil {
		return false, err
	}

	return u.notify(ctx, userId, target, dryRun)
}

func (u *PeriodReportNotifier) environmentTarget(ctx context.Context, environmentId string) (*periodReportTarget, error) {
	if target, ok := u.environmentTargets[environmentId]; ok {
		return target, nil
	}

	env, err := u.environmentRepo.FindById(ctx, environmentId)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	// 期間の取り方は API の environment_id と同じ buildStatPeriod を使う(例外イベントを含む)
	period, err := buildStatPeriod(ctx, u.environmentRepo, u.officialEventEnvironmentRepo, environmentId, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}

	target := &periodReportTarget{
		category: NotificationCategoryEnvironmentReport,
		campaign: PushCampaignEnvironmentReport,
		title:    fmt.Sprintf("『%s』環境のバトルレポートができました", env.Title),
		prefix:   "この環境では",
		linkUrl:  fmt.Sprintf(environmentReportLinkFormat, env.ID),
		period:   period,
	}
	u.environmentTargets[environmentId] = target

	return target, nil
}

// notify は期間の戦績を集計し、1戦以上あれば通知を作って push を送る。
func (u *PeriodReportNotifier) notify(ctx context.Context, userId string, target *periodReportTarget, dryRun bool) (bool, error) {
	// 全レギュレーション合算(regulationId=0)・不戦勝/不戦敗を除外(excludeDefaultMatches=true)。
	// リンク先のレポートと同じ条件にして、通知を開いた瞬間に数字が変わらないようにする。
	stat, err := u.userStatRepo.FindUserStat(ctx, userId, target.period, 0, true)
	if err != nil && !errors.Is(err, apperror.ErrRecordNotFound) {
		logError(ctx, err)
		return false, err
	}
	if stat == nil || stat.TotalMatches == 0 {
		return false, nil
	}

	// 月・環境(=リンク先)をキーにしているため、cron の多重起動や再実行でも二重には作らない
	already, err := u.notificationRepo.ExistsByUserIdAndCategoryAndLinkUrl(ctx, userId, target.category, target.linkUrl)
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
		timeNow(),
		userId,
		target.category,
		target.title,
		reportBody(target.prefix, stat, topDeckName(ctx, u.deckUsageStatRepo, userId, target.period)),
		target.linkUrl,
	)

	if err := u.notificationRepo.Save(ctx, notification); err != nil {
		logError(ctx, err)
		return false, err
	}

	// 週次レポートと同じく、アプリ内通知を作った上で push を撃つ(D2)。
	// push の失敗で通知作成は巻き戻さない
	if _, err := u.pushNotifier.Deliver(ctx, notification, target.campaign); err != nil {
		logWarn(ctx, err)
	}

	return true, nil
}
