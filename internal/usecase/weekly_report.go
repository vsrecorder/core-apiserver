package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

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

	// NotificationCategoryEnvNews は「記録ゼロの週」に代わりに届ける環境ニュースの通知カテゴリ。
	// 配当(レポート)が無い週でも毎週何かが届く状態を作る(pmf-plan-2026-08-27.md P-2)。
	NotificationCategoryEnvNews = "env_news"

	// envNewsTitle は環境ニュース通知の見出し。
	envNewsTitle = "先週のポケカ環境ニュース"

	// envNewsLinkFormat は環境ニュースのリンク先。%s には対象週の月曜が入り、
	// webapp の /deck_meta が week パラメータで初期表示週として引き継ぐ。
	envNewsLinkFormat = "/deck_meta?week=%s"

	// notificationBodyMaxLength は notifications.body の VARCHAR(256)。
	// デッキ名は自由入力ではなくスプライトの正式名だが、念のため超えないように畳む。
	notificationBodyMaxLength = 256
)

type WeeklyReportNotifierInterface interface {
	// NotifyUser は week(週内の任意日 YYYY-MM-DD)が属する週について、ユーザーへ月曜の通知を1件作成する。
	//   - その週に1戦以上あれば「先週のバトルレポート」(配当)。購読の有無によらず全員に作り、購読者には push も送る
	//   - 0戦なら「先週の環境ニュース」。push 購読者にだけ作る(届かない人のベルに未読を溜めないため)
	// 同じ週の通知が既にあれば作らない(冪等)。環境データが無い週の環境ニュースは送らない。
	// dryRun=true の場合は作成対象かどうかだけを返し、通知は作らない。
	// 戻り値の bool は「作成した(dryRunなら作成対象だった)」かどうか。
	NotifyUser(ctx context.Context, userId string, week string, dryRun bool) (bool, error)
}

// envNewsHeadline は環境ニュース本文の材料。1回のバッチでは週ごとに1度だけ集計する。
type envNewsHeadline struct {
	TopName string
	TopRate float64
	// RiserName が空なら「伸びたデッキ」の文は付けない(前週データが無い週など)
	RiserName  string
	RiserDelta float64
}

type WeeklyReportNotifier struct {
	userStatRepo            repository.UserStatInterface
	deckUsageStatRepo       repository.DeckUsageStatInterface
	notificationRepo        repository.NotificationInterface
	pushNotifier            PushNotifierInterface
	pushSubscriptionRepo    repository.PushSubscriptionInterface
	pushDeliveryRepo        repository.PushDeliveryInterface
	userStreakRepo          repository.UserStreakInterface
	weeklyDeckUsageStatRepo repository.WeeklyDeckUsageStatInterface
	pokemonSpriteRepo       repository.PokemonSpriteNameInterface

	// envNewsCache は週(月曜)ごとの環境ニュースの材料。プラットフォーム全体の週次集計は
	// 重い(前週比較で2週ぶん)ため、購読者の数だけ繰り返さないよう1回のバッチ内で持ち回る。
	// 値が nil の週は「データ無し」を意味し、それもキャッシュする。
	envNewsCache map[string]*envNewsHeadline
}

func NewWeeklyReportNotifier(
	userStatRepo repository.UserStatInterface,
	deckUsageStatRepo repository.DeckUsageStatInterface,
	notificationRepo repository.NotificationInterface,
	pushNotifier PushNotifierInterface,
	pushSubscriptionRepo repository.PushSubscriptionInterface,
	pushDeliveryRepo repository.PushDeliveryInterface,
	userStreakRepo repository.UserStreakInterface,
	weeklyDeckUsageStatRepo repository.WeeklyDeckUsageStatInterface,
	pokemonSpriteRepo repository.PokemonSpriteNameInterface,
) WeeklyReportNotifierInterface {
	return &WeeklyReportNotifier{
		userStatRepo:            userStatRepo,
		deckUsageStatRepo:       deckUsageStatRepo,
		notificationRepo:        notificationRepo,
		pushNotifier:            pushNotifier,
		pushSubscriptionRepo:    pushSubscriptionRepo,
		pushDeliveryRepo:        pushDeliveryRepo,
		userStreakRepo:          userStreakRepo,
		weeklyDeckUsageStatRepo: weeklyDeckUsageStatRepo,
		pokemonSpriteRepo:       pokemonSpriteRepo,
		envNewsCache:            map[string]*envNewsHeadline{},
	}
}

func (u *WeeklyReportNotifier) NotifyUser(ctx context.Context, userId string, week string, dryRun bool) (bool, error) {
	now := timeNow()

	fromDate, toDate, err := weekRange(week, now.Local())
	if err != nil {
		logError(ctx, err)
		return false, err
	}
	weekKey := fromDate.Format(weekDateLayout)

	// 全レギュレーション合算(regulationId=0)。webapp のバトルレポートも絞らずに出す
	stat, err := u.userStatRepo.FindUserStat(ctx, userId, fromDate, toDate, 0)
	if err != nil && !errors.Is(err, apperror.ErrRecordNotFound) {
		logError(ctx, err)
		return false, err
	}

	if stat != nil && stat.TotalMatches > 0 {
		return u.notifyReport(ctx, userId, weekKey, stat, fromDate, toDate, now, dryRun)
	}

	// その週に対戦が無ければレポートは空になる。代わりに環境ニュースを届ける
	return u.notifyEnvNews(ctx, userId, weekKey, fromDate, toDate, now, dryRun)
}

// notifyReport は「先週のバトルレポート」(記録の配当)を作る。
func (u *WeeklyReportNotifier) notifyReport(
	ctx context.Context,
	userId string,
	weekKey string,
	stat *entity.UserStat,
	fromDate, toDate time.Time,
	now time.Time,
	dryRun bool,
) (bool, error) {
	linkUrl := fmt.Sprintf(weeklyReportLinkFormat, weekKey)

	already, err := u.alreadyNotified(ctx, userId, NotificationCategoryWeeklyReport, linkUrl)
	if err != nil {
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

// notifyEnvNews は「先週の環境ニュース」を作る。push 購読者だけが対象。
// 記録していない人にアプリ内通知だけ作っても見られず、ベルに未読が溜まるだけになるため。
func (u *WeeklyReportNotifier) notifyEnvNews(
	ctx context.Context,
	userId string,
	weekKey string,
	fromDate, toDate time.Time,
	now time.Time,
	dryRun bool,
) (bool, error) {
	subscriptions, err := u.pushSubscriptionRepo.FindLiveByUserId(ctx, userId)
	if err != nil {
		logError(ctx, err)
		return false, err
	}
	if len(subscriptions) == 0 {
		return false, nil
	}

	linkUrl := fmt.Sprintf(envNewsLinkFormat, weekKey)

	already, err := u.alreadyNotified(ctx, userId, NotificationCategoryEnvNews, linkUrl)
	if err != nil {
		return false, err
	}
	if already {
		return false, nil
	}

	// 反応の無い人には隔週だけ送る(週末リマインドと同じガードレール)。
	// 「記録が無い」の基準は最終記録週。一度も記録していない人はゼロ値のまま
	var lastRecordedWeek time.Time
	streak, err := u.userStreakRepo.FindByUserId(ctx, userId)
	if err != nil && !errors.Is(err, apperror.ErrRecordNotFound) {
		logError(ctx, err)
		return false, err
	}
	if streak != nil {
		lastRecordedWeek = streak.LastRecordedWeek
	}

	thisMonday := mondayOf(now)
	quiet, err := isPushUnresponsive(ctx, u.pushDeliveryRepo, userId, PushCampaignEnvNews, lastRecordedWeek, thisMonday)
	if err != nil {
		logError(ctx, err)
		return false, err
	}
	if quiet && !isEvenWeek(thisMonday) {
		return false, nil
	}

	headline, err := u.envNewsFor(ctx, weekKey, fromDate, toDate)
	if err != nil {
		logError(ctx, err)
		return false, err
	}
	if headline == nil {
		// 環境データが無い週(全体で記録が無い・名前が引けない)にニュースは作れない
		slog.InfoContext(ctx, "env news skipped: no environment data for the week", slog.String("week", weekKey))
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
		NotificationCategoryEnvNews,
		envNewsTitle,
		envNewsBody(headline),
		linkUrl,
	)

	if err := u.notificationRepo.Save(ctx, notification); err != nil {
		logError(ctx, err)
		return false, err
	}

	if _, err := u.pushNotifier.Deliver(ctx, notification, PushCampaignEnvNews); err != nil {
		logWarn(ctx, err)
	}

	return true, nil
}

// alreadyNotified は、同じ週(=同じリンク先)の通知を既に作っているかを返す。
// 週をキーにしているため、cron の多重起動でも -week 指定の再実行でも二重には作らない。
// 直近N件ではなく全期間で見るのは、-week で古い週をバックフィルしたときに、その週の通知が
// 直近から外れていて二重に作られるのを防ぐため。
func (u *WeeklyReportNotifier) alreadyNotified(ctx context.Context, userId string, category string, linkUrl string) (bool, error) {
	exists, err := u.notificationRepo.ExistsByUserIdAndCategoryAndLinkUrl(ctx, userId, category, linkUrl)
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

// envNewsFor は週の環境ニュースの材料を返す(週ごとに1回だけ集計してキャッシュする)。
// 材料が作れない週は nil を返し、それもキャッシュする。
func (u *WeeklyReportNotifier) envNewsFor(ctx context.Context, weekKey string, fromDate, toDate time.Time) (*envNewsHeadline, error) {
	if headline, ok := u.envNewsCache[weekKey]; ok {
		return headline, nil
	}

	stat, err := u.weeklyDeckUsageStatRepo.FindWeeklyDeckUsageStat(ctx, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	headline, err := u.buildEnvNewsHeadline(ctx, stat)
	if err != nil {
		return nil, err
	}

	u.envNewsCache[weekKey] = headline
	return headline, nil
}

// buildEnvNewsHeadline は週次デッキ使用率から「使用率1位」と「前週からいちばん伸びたデッキ」を選び、
// スプライトの正式名で人が読める名前にする。1位の名前が引けなければ nil(ニュースにならない)。
func (u *WeeklyReportNotifier) buildEnvNewsHeadline(ctx context.Context, stat *entity.WeeklyDeckUsageStat) (*envNewsHeadline, error) {
	if stat == nil || stat.TotalVotes == 0 {
		return nil, nil
	}

	// 「その他」(指紋が空)とスプライトの無い変種は名指しできないので外す
	decks := make([]*entity.DeckUsageVariant, 0, len(stat.Decks))
	for _, d := range stat.Decks {
		if d.Fingerprint != "" && len(d.PokemonSprites) > 0 {
			decks = append(decks, d)
		}
	}
	if len(decks) == 0 {
		return nil, nil
	}

	// 集計側も使用率順で返すが、本文の根拠を並び順に依存させないため明示的に並べ替える
	sort.SliceStable(decks, func(i, j int) bool {
		if decks[i].UsageRate != decks[j].UsageRate {
			return decks[i].UsageRate > decks[j].UsageRate
		}
		return decks[i].Count > decks[j].Count
	})
	top := decks[0]

	// 前週比で最も伸びた変種。前週に個別表示されていない(新登場)変種は比較できないので外す
	var riser *entity.DeckUsageVariant
	bestDelta := 0.0
	for _, d := range decks {
		if d.PreviousUsageRate == nil {
			continue
		}
		if delta := d.UsageRate - *d.PreviousUsageRate; delta > bestDelta {
			bestDelta = delta
			riser = d
		}
	}

	ids := spriteIdsOf(top.PokemonSprites)
	if riser != nil {
		ids = append(ids, spriteIdsOf(riser.PokemonSprites)...)
	}
	names, err := u.pokemonSpriteRepo.FindNamesByIds(ctx, ids)
	if err != nil {
		return nil, err
	}

	topName := deckDisplayName(top.PokemonSprites, names)
	if topName == "" {
		return nil, nil
	}

	headline := &envNewsHeadline{TopName: topName, TopRate: top.UsageRate}
	if riser != nil {
		if name := deckDisplayName(riser.PokemonSprites, names); name != "" {
			headline.RiserName = name
			headline.RiserDelta = bestDelta
		}
	}

	return headline, nil
}

func spriteIdsOf(sprites []*entity.PokemonSprite) []string {
	ids := make([]string, 0, len(sprites))
	for _, s := range sprites {
		ids = append(ids, s.ID)
	}
	return ids
}

// deckDisplayName はスプライトの組み合わせを「メガレックウザ＋ホウオウ」のような名前にする。
// 表示枠(position)の順に並べ、名前が引けないスプライトは飛ばす。全部引けなければ空文字。
func deckDisplayName(sprites []*entity.PokemonSprite, names map[string]string) string {
	ordered := make([]*entity.PokemonSprite, len(sprites))
	copy(ordered, sprites)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Position < ordered[j].Position })

	parts := make([]string, 0, len(ordered))
	for _, s := range ordered {
		if name := strings.TrimSpace(names[s.ID]); name != "" {
			parts = append(parts, name)
		}
	}

	return strings.Join(parts, "＋")
}

// envNewsBody は環境ニュースの本文を組み立てる。
// 例: 先週の環境: 使用率1位は『ドラパルト＋ヨノワール』(6.1%)。いちばん伸びたのは『メガレックウザ＋ホウオウ』(+3.9pt)。今週1戦記録すると、来週はあなたのレポートが届きます
// 伸びたデッキが1位と同じなら1文にまとめ、前週データが無ければその文を省く。
// 名前が長すぎて VARCHAR(256) を超える場合は、伸びたデッキの文から順に落とす。
func envNewsBody(h *envNewsHeadline) string {
	const cta = "今週1戦記録すると、来週はあなたのレポートが届きます"
	top := fmt.Sprintf("先週の環境: 使用率1位は『%s』(%.1f%%)", h.TopName, h.TopRate*100)

	candidates := []string{}
	switch {
	case h.RiserName == "":
		// 伸びたデッキの文は無し
	case h.RiserName == h.TopName:
		candidates = append(candidates, fmt.Sprintf("先週の環境: 使用率1位は『%s』(%.1f%%・前週比+%.1fpt)。%s", h.TopName, h.TopRate*100, h.RiserDelta*100, cta))
	default:
		candidates = append(candidates, fmt.Sprintf("%s。いちばん伸びたのは『%s』(+%.1fpt)。%s", top, h.RiserName, h.RiserDelta*100, cta))
	}
	candidates = append(candidates, fmt.Sprintf("%s。%s", top, cta))

	for _, body := range candidates {
		if utf8.RuneCountInString(body) <= notificationBodyMaxLength {
			return body
		}
	}

	// ここまで来るのはデッキ名が異常に長い場合だけ。末尾を切って収める
	runes := []rune(candidates[len(candidates)-1])
	return string(runes[:notificationBodyMaxLength])
}
