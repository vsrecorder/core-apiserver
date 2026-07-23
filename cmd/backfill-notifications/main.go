// backfill-notifications は、通知機能の導入前から既にバッジ・称号・ランク・環境バッジを
// 達成していたユーザーに対し、その実績を「既読済みの通知履歴」として遡って作成するための
// 一回限りの初期投入バッチ。
//
// オンボーディング系バッジ(user_badgesに永続化済み)は実際の achieved_at をそのまま
// created_at/read_at に使う。マイルストーン系・週次ストリーク系バッジ、称号・ランクは
// シーズンごとにライブ集計する仕様(過去の達成日時を保持しない)ため、シーズン内の
// records/matches/decksの日付を遡って走査し、閾値・tierへ実際に到達した日を
// created_at/read_at として1件だけ作成する(見つからない場合のみ本バッチ実行時刻を使う)。
//
// 環境バッジ(user_environment_badgesに永続化済み)も、オンボーディング系バッジと同様に
// 実際の achieved_at(正確にはcreated_at。usecase.EnvironmentBadgeEvaluation.NotifyAchieved
// 参照)をそのまま使う。ただし、バッジ行自体は backfill-user-environment-badges が先に
// 作成する運用のため、ここでは notification_id が空の行に対してのみ通知を作成し、生成した
// 通知IDを user_environment_badges.notification_id へ書き戻す(次回以降の重複作成を防ぐ)。
//
// 称号・ランク(3)は「はじめの一歩」バッジ(オンボーディング系、user_badges)が全て
// 揃っていないユーザーはスキップする。称号・ランクは対戦記録の蓄積が前提の実績のため
// オンボーディング完了より先に到達すること自体はあり得るが、ユーザーが最初に受け取る
// 通知は「はじめの一歩」から順に見せたいための意図的な制約。未達成分は次回以降の
// 実行時、オンボーディングが揃った時点でまとめて補完される。
//
// 冪等性: 対象ユーザーが既に1件でも notifications 行を持つ場合、1(オンボーディング系
// バッジ)・2(マイルストーン・ストリーク系バッジ)はスキップする。これにより誤って複数回
// 実行しても通知が重複しない(backfill-badges が再実行で通知を重複生成しうる問題を踏まえた
// 設計)。ただし裏を返すと、対象ユーザーが導入後に何らかの通知を既に受け取っている場合は
// 1・2のバックフィル対象外になる(導入前に一度だけ実行する運用を想定)。3(称号・ランク)・
// 4(環境バッジ)はこの制約を受けず、常に個別の重複判定(称号・ランクはbody文字列、環境バッジは
// notification_idが指す通知の実在)で補完する。
//
// notificationsを全削除してから本バッチを実行すると、上記1〜4のすべてが対象になり、通知履歴を
// ゼロから作り直せる(集計ロジックの変更で過去の達成判定が変わった場合などに、辻褄を合わせる
// ための運用)。その際、user_environment_badges.notification_id は notifications とは別テーブル
// のため削除されずに残るが、4は「IDが入っているか」ではなく「そのIDの通知が実在するか」で
// 判定するため、環境バッジの通知も正しく作り直される(事前にnotification_idをクリアする必要はない)。
//
// ただし、称号喪失(notifyDesignationLost)・ランクダウン(notifyRankDown)の通知だけは作り直せない。
// これらは「過去のある時点で条件を満たさなくなった」というイベントであり、現在のデータからは
// 「今達成していない」ことしか分からず、いつ失ったかを遡れないため。全削除するとこの2種類の
// 通知履歴は失われる。
//
// 使い方:
//
//	# 変更内容を書き込まずに確認するだけ(デフォルト)
//	go run ./cmd/backfill-notifications
//
//	# 実際に notifications へ反映する
//	go run ./cmd/backfill-notifications -dry-run=false
//
//	# 特定ユーザーのみ対象にする(調査・検証用)
//	go run ./cmd/backfill-notifications -user-id=xxxxx -dry-run=false
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/joho/godotenv"
	ulid "github.com/oklog/ulid/v2"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

const notificationLinkUrl = "/users"

// entropy はULID生成用の乱数源。称号・ランクアップ等、同じachievedAtで複数件の
// 通知を連続作成するケースで生成順とID順を一致させるため、単調増加なDefaultEntropyを使う
// (internal/usecase/util.go の同名変数と同じ意図)。
var entropy = ulid.DefaultEntropy()

func generateId() (string, error) {
	ms := ulid.Timestamp(time.Now())
	id, err := ulid.New(ms, entropy)

	return id.String(), err
}

func main() {
	dryRun := flag.Bool("dry-run", true, "true の場合、書き込みは行わず差分の確認のみ行う")
	targetUserId := flag.String("user-id", "", "指定した場合、そのユーザーのみを対象にする(未指定なら全対象ユーザー)")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	db, err := postgres.NewDB(
		os.Getenv("DB_HOSTNAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER_NAME"),
		os.Getenv("DB_USER_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	if err != nil {
		log.Printf("failed to connect database: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	notificationRepo := infrastructure.NewNotification(db)
	championshipSeriesRepo := infrastructure.NewChampionshipSeries(db)
	badgeStatsRepo := infrastructure.NewBadgeStats(db)
	badgeUsecase := usecase.NewBadge(
		infrastructure.NewBadgeDefinition(db),
		infrastructure.NewUserBadge(db),
		badgeStatsRepo,
		championshipSeriesRepo,
	)
	designationUsecase := usecase.NewDesignation(
		infrastructure.NewDesignation(db),
		infrastructure.NewDesignationStats(db),
		championshipSeriesRepo,
		infrastructure.NewUserPlayer(db),
	)
	designationEvaluation := usecase.NewDesignationEvaluation(
		infrastructure.NewDesignation(db),
		infrastructure.NewDesignationStats(db),
		championshipSeriesRepo,
		notificationRepo,
		infrastructure.NewUserPlayer(db),
	)
	environmentRepo := infrastructure.NewEnvironment(db)
	userEnvironmentBadgeRepo := infrastructure.NewUserEnvironmentBadge(db)
	environmentBadgeEvaluation := usecase.NewEnvironmentBadgeEvaluation(
		environmentRepo,
		userEnvironmentBadgeRepo,
		notificationRepo,
		infrastructure.NewTransactionManager(db),
	)

	ctx := context.Background()

	now := time.Now().Local()

	// マイルストーン系・週次ストリーク系バッジ、称号・ランクの通知本文に
	// どのシーズンの実績かを明記するためのラベル(例:"2026")。
	seasonLabel, err := usecase.CurrentSeasonLabel(ctx, championshipSeriesRepo, now)
	if err != nil {
		log.Printf("failed to resolve current season label: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	// 実際の達成日を遡って求めるための、現在のシーズンの期間。
	seasonFromDate, seasonToDate, err := usecase.CurrentSeasonDateRange(ctx, championshipSeriesRepo, now)
	if err != nil {
		log.Printf("failed to resolve current season date range: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	var userIds []string
	if *targetUserId != "" {
		userIds = []string{*targetUserId}
	} else {
		userIds, err = findTargetUserIds(db)
		if err != nil {
			log.Printf("failed to list users: %v\n", err)
			os.Exit(ExitCodeNG)
		}
	}

	// 退会済み・存在しないユーザーを除外し、有効なユーザー(usersに存在し deleted_at IS NULL)
	// だけを対象にする。findTargetUserIds は user_badges からもユーザーを集めるが、退会
	// (usecase.User.Delete)では records・decks・deck_codes・user_players と users 本体は
	// 削除される一方、user_badges・user_environment_badges は残るため、退会済みユーザーが
	// 対象に混ざりうる。退会後は本人が通知を閲覧できないため除外する。
	beforeFilter := len(userIds)
	userIds, err = filterValidUserIds(db, userIds)
	if err != nil {
		log.Printf("failed to filter valid users: %v\n", err)
		os.Exit(ExitCodeNG)
	}
	if skipped := beforeFilter - len(userIds); skipped > 0 {
		log.Printf("skipped %d withdrawn/non-existent users\n", skipped)
	}

	if *dryRun {
		log.Printf("[dry-run] checking notification history for %d users (書き込みは行いません)\n", len(userIds))
	} else {
		log.Printf("backfilling notification history for %d users\n", len(userIds))
	}

	backfilled := 0
	for _, userId := range userIds {
		created, err := backfillUser(
			ctx, db, notificationRepo, badgeStatsRepo, badgeUsecase, designationUsecase, designationEvaluation,
			userEnvironmentBadgeRepo, environmentRepo, environmentBadgeEvaluation,
			userId, seasonLabel, seasonFromDate, seasonToDate, now, *dryRun,
		)
		if err != nil {
			log.Printf("failed to backfill user=%s: %v\n", userId, err)
			continue
		}
		if created > 0 {
			backfilled++
			if *dryRun {
				log.Printf("[dry-run] user=%s: %d件の通知を作成予定\n", userId, created)
			} else {
				log.Printf("user=%s: %d件の通知を作成しました\n", userId, created)
			}
		}
	}

	if *dryRun {
		log.Printf("[dry-run] completed: %d/%d users have notifications to backfill\n", backfilled, len(userIds))
	} else {
		log.Printf("completed: backfilled %d/%d users\n", backfilled, len(userIds))
	}

	os.Exit(ExitCodeOK)
}

// findTargetUserIds は「現存する record を持つユーザー」と「オンボーディング系バッジを
// 持つユーザー」の和集合を返す(称号・マイルストーン系バッジはrecordsが無ければ
// 絶対に達成できないが、サインアップ済みバッジだけを持つユーザーを取りこぼさないため)。
func findTargetUserIds(db *gorm.DB) ([]string, error) {
	seen := make(map[string]struct{})
	var userIds []string

	tablesAndConds := map[string]string{
		"records":     "deleted_at IS NULL",
		"user_badges": "",
	}
	for table, cond := range tablesAndConds {
		var ids []string
		q := db.Table(table)
		if cond != "" {
			q = q.Where(cond)
		}
		if tx := q.Distinct("user_id").Pluck("user_id", &ids); tx.Error != nil {
			return nil, tx.Error
		}
		for _, id := range ids {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				userIds = append(userIds, id)
			}
		}
	}

	return userIds, nil
}

// filterValidUserIds は与えられたユーザーIDのうち、有効なユーザー(usersテーブルに存在し
// 退会していない=deleted_at IS NULL)のものだけを、渡された順序を保って返す。
//
// findTargetUserIds は user_badges からもユーザーを集めるが、退会(usecase.User.Delete)では
// records・decks・deck_codes・user_players と users 本体は削除される一方、user_badges・
// user_environment_badges は削除されずに残る。そのため退会済みユーザーが対象に混ざりうるが、
// 退会後は本人が通知を閲覧できないため、ここで有効なユーザーだけに絞り込む。
func filterValidUserIds(db *gorm.DB, userIds []string) ([]string, error) {
	if len(userIds) == 0 {
		return userIds, nil
	}

	// db.Model(&model.User{}) には gorm のソフトデリートにより deleted_at IS NULL が
	// 自動で付くため、退会済み(および存在しない)ユーザーはここに含まれない。
	var validIds []string
	if tx := db.Model(&model.User{}).
		Where("id IN ?", userIds).
		Pluck("id", &validIds); tx.Error != nil {
		return nil, tx.Error
	}

	valid := make(map[string]struct{}, len(validIds))
	for _, id := range validIds {
		valid[id] = struct{}{}
	}

	filtered := make([]string, 0, len(userIds))
	for _, id := range userIds {
		if _, ok := valid[id]; ok {
			filtered = append(filtered, id)
		}
	}

	return filtered, nil
}

// backfillUser は1ユーザー分の通知履歴を作成する。作成した(dry-runなら作成予定の)件数を返す。
func backfillUser(
	ctx context.Context,
	db *gorm.DB,
	notificationRepo repository.NotificationInterface,
	badgeStatsRepo repository.BadgeStatsInterface,
	badgeUsecase usecase.BadgeInterface,
	designationUsecase usecase.DesignationInterface,
	designationEvaluation usecase.DesignationEvaluationInterface,
	userEnvironmentBadgeRepo repository.UserEnvironmentBadgeInterface,
	environmentRepo repository.EnvironmentInterface,
	environmentBadgeEvaluation usecase.EnvironmentBadgeEvaluationInterface,
	userId string,
	seasonLabel string,
	seasonFromDate time.Time,
	seasonToDate time.Time,
	now time.Time,
	dryRun bool,
) (int, error) {
	var existingCount int64
	if tx := db.Model(&model.Notification{}).Where("user_id = ?", userId).Count(&existingCount); tx.Error != nil {
		return 0, tx.Error
	}

	created := 0

	// 称号・ランク(3)は「はじめの一歩」バッジ(オンボーディング系)が全て揃うまで通知を
	// 作らない。称号・ランクは対戦記録の蓄積が前提の実績であり、オンボーディングより
	// 先に到達すること自体は起こり得るが、ユーザーが最初に受け取る通知は「はじめの一歩」から
	// 順に見せたいため。badgeUsecase.GetByUserId はセクション2でも使うためここで一度だけ呼ぶ。
	badgeViews, err := badgeUsecase.GetByUserId(ctx, userId, "")
	if err != nil {
		return 0, err
	}

	// onboardingAchievedAt は「はじめの一歩」(オンボーディング系バッジ)を全て達成した日時
	// (=最後に達成したオンボーディングバッジの achieved_at)。称号は「デッキが登録され、
	// 対戦結果が紐づいた記録」が前提のため、オンボーディング完了より前に称号へ到達すること
	// は達成条件上あり得ない。称号・ランクの達成日は過去のデータから遡って求める近似値で
	// あり、算出結果がこの日時より前になると通知の並び順が達成条件と逆転する(「初デッキ」
	// バッジの通知が称号獲得・ランクアップの通知より新しく表示される)ため、下限として使う。
	onboardingComplete := true
	var onboardingAchievedAt time.Time
	for _, view := range badgeViews {
		if view.Definition.Category != usecase.BadgeCategoryOnboarding {
			continue
		}
		if !view.Achieved {
			onboardingComplete = false
			continue
		}
		if view.AchievedAt.After(onboardingAchievedAt) {
			onboardingAchievedAt = view.AchievedAt
		}
	}

	// バッジ(1・2)は個別の重複チェックを持たないため、従来通り「1件でも通知があれば
	// 丸ごとスキップ」する。称号・ランク(3)・環境バッジ(4)は、既にバッジ通知だけ受け取っている
	// ユーザーの称号履歴・環境バッジ履歴だけが欠落する問題(達成tierを飛び越えた場合に最終tierしか
	// 通知されなかった、環境バッジがbackfill-user-environment-badgesで先に付与されていた等)が
	// あるため、existingCountに関わらず個別に判定する。
	if existingCount == 0 {
		// 1. オンボーディング系バッジ(永続化済み、実際のachieved_atを使う)
		var userBadges []*model.UserBadge
		if tx := db.Where("user_id = ?", userId).Find(&userBadges); tx.Error != nil {
			return created, tx.Error
		}

		if len(userBadges) > 0 {
			var defs []*model.BadgeDefinition
			if tx := db.Find(&defs); tx.Error != nil {
				return created, tx.Error
			}
			defById := make(map[string]*model.BadgeDefinition, len(defs))
			for _, d := range defs {
				defById[d.ID] = d
			}

			for _, ub := range userBadges {
				def := defById[ub.BadgeDefinitionId]
				if def == nil {
					continue
				}

				if err := saveNotification(
					ctx, notificationRepo, dryRun,
					userId, usecase.NotificationCategoryBadge, "バッジを獲得しました",
					fmt.Sprintf("「%s」バッジを獲得しました！", def.Name),
					ub.AchievedAt,
				); err != nil {
					return created, err
				}
				created++
			}
		}

		// 2. マイルストーン系・週次ストリーク系バッジ(現在のシーズンで達成済みのもののみ)。
		// 「実際に達成した日」を遡って求めるため、シーズン内のrecord/match/deck_codeの日付を
		// 昇順に並べ、閾値に到達した時点の日付を使う(週次ストリークはStreakWeeksAchievedAtで求める)。
		hasMilestoneOrStreak := false
		for _, view := range badgeViews {
			if view.Definition.Category != usecase.BadgeCategoryOnboarding && view.Achieved {
				hasMilestoneOrStreak = true
				break
			}
		}

		var recordDates, matchDates, deckCodeDates []time.Time
		var streakAchievedAt map[int]time.Time
		if hasMilestoneOrStreak {
			recordDates, err = badgeStatsRepo.FindRecordDatesByUserId(ctx, userId, seasonFromDate, seasonToDate)
			if err != nil {
				return created, err
			}
			sort.Slice(recordDates, func(i, j int) bool { return recordDates[i].Before(recordDates[j]) })

			matchDates, err = badgeStatsRepo.FindMatchDatesByUserId(ctx, userId, seasonFromDate, seasonToDate)
			if err != nil {
				return created, err
			}

			deckCodeDates, err = badgeStatsRepo.FindDeckCodeDatesByUserId(ctx, userId, seasonFromDate, seasonToDate)
			if err != nil {
				return created, err
			}

			streakAchievedAt = usecase.StreakWeeksAchievedAt(recordDates)
		}

		for _, view := range badgeViews {
			if view.Definition.Category == usecase.BadgeCategoryOnboarding || !view.Achieved {
				continue
			}

			title := "バッジを獲得しました"
			category := usecase.NotificationCategoryBadge
			if view.Definition.Category == usecase.BadgeCategoryStreak {
				category = usecase.NotificationCategoryStreak
				title = "ストリークを継続中です"
			}

			achievedAt := milestoneAchievedAt(view.Definition, recordDates, matchDates, deckCodeDates, streakAchievedAt, now)

			if err := saveNotification(
				ctx, notificationRepo, dryRun,
				userId, category, title,
				fmt.Sprintf("%sシーズンで「%s」バッジを獲得しました！", seasonLabel, view.Definition.Name),
				achievedAt,
			); err != nil {
				return created, err
			}
			created++
		}
	}

	// 3. 称号・ランク(現在のシーズンで到達済みの全tierが対象)。1回の評価でtierが
	// 複数段上がった場合に最終tierしか通知されない問題(designation_evaluation.go の
	// NotifyIfTierChanged参照)があったため、通過済みの各tier・各ランクを個別に
	// (既に通知済みかどうかをbody文字列で判定して)補完する。「実際に達成した日」を
	// 遡って求めるため、シーズン内のrecordのevent_dateを候補日として昇順に走査し、
	// TierAsOfがそのtier/ランクを満たす最初の日を使う。
	//
	// 「はじめの一歩」バッジが揃うまではスキップする(上のonboardingComplete参照)。
	// 未達成分は次回以降の実行で「はじめの一歩」が揃った時点でまとめて補完される。
	if !onboardingComplete {
		if dryRun {
			log.Printf("[dry-run] user=%s: オンボーディングバッジが未達成のため称号・ランク通知をスキップ\n", userId)
		}
	} else {
		designationCreated, err := backfillDesignationHistory(
			ctx, db, notificationRepo, designationUsecase, designationEvaluation,
			userId, seasonLabel, seasonFromDate, seasonToDate, onboardingAchievedAt, now, dryRun,
		)
		if err != nil {
			return created, err
		}
		created += designationCreated
	}

	// 4. 環境バッジ(user_environment_badgesに永続化済み)。バッジ行自体は
	// backfill-user-environment-badges が先に作成する運用のため、ここでは通知だけを
	// notification_idが空の行に対して補完し、生成した通知IDを書き戻す。
	environmentBadgeCreated, err := backfillEnvironmentBadgeNotifications(
		ctx, db, userEnvironmentBadgeRepo, environmentRepo, environmentBadgeEvaluation, userId, dryRun,
	)
	if err != nil {
		return created, err
	}
	created += environmentBadgeCreated

	return created, nil
}

// backfillEnvironmentBadgeNotifications はユーザーが獲得済みの環境バッジ
// (user_environment_badges)のうち、まだ通知(notifications)が無いもの(notification_idが
// 空の行)だけを個別に補完する。通知の内容(タイトル・本文)は環境バッジ獲得時のリアルタイム
// 通知と同じにするため、usecase.EnvironmentBadgeEvaluation.NotifyAchieved をそのまま使う
// (backfill-user-environment-badges から移植。isReadはtrueにして通知ベルに新着として
// 表示しないようにする点も同様)。生成した通知IDはuser_environment_badges.notification_idへ
// 書き戻し、再実行時に同じ行へ重複して通知が作られないようにする。
func backfillEnvironmentBadgeNotifications(
	ctx context.Context,
	db *gorm.DB,
	userEnvironmentBadgeRepo repository.UserEnvironmentBadgeInterface,
	environmentRepo repository.EnvironmentInterface,
	environmentBadgeEvaluation usecase.EnvironmentBadgeEvaluationInterface,
	userId string,
	dryRun bool,
) (int, error) {
	badges, err := userEnvironmentBadgeRepo.FindByUserId(ctx, userId)
	if err != nil {
		return 0, err
	}

	// 既存通知のID集合。notification_id が「入っているか」ではなく「指す通知が実在するか」で
	// 判定する。notifications を全削除してから本バッチで履歴を作り直す運用では、
	// user_environment_badges 側の notification_id だけが残るため、IDの有無で判定すると
	// 通知済みとみなされ環境バッジの通知だけが永久に復元されない。
	var existingNotificationIds []string
	if tx := db.Model(&model.Notification{}).
		Where("user_id = ?", userId).
		Pluck("id", &existingNotificationIds); tx.Error != nil {
		return 0, tx.Error
	}
	existingNotifications := make(map[string]struct{}, len(existingNotificationIds))
	for _, id := range existingNotificationIds {
		existingNotifications[id] = struct{}{}
	}

	created := 0
	for _, badge := range badges {
		if badge.NotificationId != "" {
			if _, ok := existingNotifications[badge.NotificationId]; ok {
				continue
			}
			// notification_id は残っているが参照先の通知が無い(通知だけ削除された)。
			// 通知を作り直し、新しいIDで notification_id を上書きする。
		}

		if dryRun {
			created++
			continue
		}

		env, err := environmentRepo.FindById(ctx, badge.EnvironmentId)
		if err != nil {
			return created, err
		}

		notificationId, err := environmentBadgeEvaluation.NotifyAchieved(ctx, userId, env, badge.CreatedAt, true)
		if err != nil {
			return created, err
		}

		if tx := db.Model(&model.UserEnvironmentBadge{}).
			Where("user_id = ? AND environment_id = ?", userId, badge.EnvironmentId).
			Update("notification_id", notificationId); tx.Error != nil {
			return created, tx.Error
		}

		created++
	}

	return created, nil
}

// backfillDesignationHistory はユーザーが現在のシーズンで通過済みの称号tier・ランクの
// うち、まだ通知(notifications)が無いものを個別に補完する。称号名・ランク名は
// tierごとに一意なため、既存通知のbody文字列にその名前が含まれるかで重複判定する
// (backfillUser本体と異なり、他カテゴリ(バッジ)の通知が既にあるユーザーでも対象にする)。
// 併せて、既に作成済みの称号・ランク通知の created_at が onboardingAchievedAt より前に
// なっている(=「はじめの一歩」完了前に称号を獲得したことになっており、達成条件上あり得ない)
// 場合は、再計算した達成日時で補正する。
func backfillDesignationHistory(
	ctx context.Context,
	db *gorm.DB,
	notificationRepo repository.NotificationInterface,
	designationUsecase usecase.DesignationInterface,
	designationEvaluation usecase.DesignationEvaluationInterface,
	userId string,
	seasonLabel string,
	seasonFromDate time.Time,
	seasonToDate time.Time,
	onboardingAchievedAt time.Time,
	now time.Time,
	dryRun bool,
) (int, error) {
	designationView, err := designationUsecase.GetByUserId(ctx, userId, "")
	if err != nil {
		return 0, err
	}

	if designationView.Current == nil {
		return 0, nil
	}

	candidateDates, achievedAtByEventDate, err := seasonRecordEventDates(db, userId, seasonFromDate, seasonToDate)
	if err != nil {
		return 0, err
	}

	created := 0
	notifiedRanks := make(map[string]bool)

	for _, item := range designationView.Ladder {
		if !item.Achieved {
			continue
		}
		def := item.Designation
		tier := def.Tier

		designationCreated, err := upsertDesignationNotification(
			ctx, db, notificationRepo, designationEvaluation, dryRun,
			userId, usecase.NotificationCategoryDesignation, "称号を獲得しました", def.Name,
			fmt.Sprintf("%sシーズンで称号「%s %s」を獲得しました！", seasonLabel, def.Emoji, def.Name),
			func(t int) bool { return t >= tier },
			candidateDates, achievedAtByEventDate, onboardingAchievedAt, now,
		)
		if err != nil {
			return created, err
		}
		created += designationCreated

		rankName := usecase.RankNameForTier(tier)
		if rankName == "" || notifiedRanks[rankName] {
			continue
		}
		notifiedRanks[rankName] = true

		rankMinTier := usecase.MinTierForRank(rankName)

		rankCreated, err := upsertDesignationNotification(
			ctx, db, notificationRepo, designationEvaluation, dryRun,
			userId, usecase.NotificationCategoryRank, "ランクが上がりました", rankName,
			fmt.Sprintf("%sシーズンでランクが「%s」に上がりました！", seasonLabel, rankName),
			func(t int) bool { return t >= rankMinTier },
			candidateDates, achievedAtByEventDate, onboardingAchievedAt, now,
		)
		if err != nil {
			return created, err
		}
		created += rankCreated
	}

	return created, nil
}

// upsertDesignationNotification は称号・ランクの通知1件について、
//   - まだ無ければ、遡って求めた達成日時(onboardingAchievedAtを下限とする)で作成する
//   - 既にあり、その created_at が onboardingAchievedAt より前(達成条件上あり得ない日時)
//     なら、再計算した達成日時で created_at を補正する
//
// 通知を新規作成した場合のみ1を返す(補正はログのみで、作成件数には数えない)。
// 既存通知の created_at が onboardingAchievedAt 以降であれば、リアルタイムに作成された
// 正しい通知(処理時刻ベース)であるため一切触らない。
func upsertDesignationNotification(
	ctx context.Context,
	db *gorm.DB,
	notificationRepo repository.NotificationInterface,
	designationEvaluation usecase.DesignationEvaluationInterface,
	dryRun bool,
	userId string,
	category string,
	title string,
	needle string,
	body string,
	satisfies func(tier int) bool,
	candidateDates []time.Time,
	achievedAtByEventDate map[time.Time]time.Time,
	onboardingAchievedAt time.Time,
	now time.Time,
) (int, error) {
	existing, err := findNotificationByBodyContains(db, userId, category, needle)
	if err != nil {
		return 0, err
	}

	if existing != nil && !existing.CreatedAt.Before(onboardingAchievedAt) {
		return 0, nil
	}

	achievedAt := designationAchievedAt(
		ctx, designationEvaluation, userId, candidateDates, achievedAtByEventDate, satisfies, now,
	)
	// 遡って求めた達成日時は近似値のため、「はじめの一歩」完了より前になることがある。
	// 称号は「はじめの一歩」(デッキ登録・記録作成・対戦結果追加)が揃って初めて到達できる
	// ため、その完了日時を下限としてクランプし、通知の並び順が達成条件と逆転しないようにする。
	if achievedAt.Before(onboardingAchievedAt) {
		achievedAt = onboardingAchievedAt
	}

	if existing == nil {
		if err := saveNotification(
			ctx, notificationRepo, dryRun, userId, category, title, body, achievedAt,
		); err != nil {
			return 0, err
		}

		return 1, nil
	}

	if dryRun {
		log.Printf(
			"[dry-run] user=%s: 通知の日時を補正予定 body=%s created_at=%s -> %s\n",
			userId, existing.Body, existing.CreatedAt.Format(time.RFC3339), achievedAt.Format(time.RFC3339),
		)

		return 0, nil
	}

	if err := notificationRepo.UpdateContent(
		ctx, existing.ID, achievedAt, existing.Title, existing.Body, existing.IsRead,
	); err != nil {
		return 0, err
	}

	log.Printf(
		"user=%s: 通知の日時を補正しました body=%s created_at=%s -> %s\n",
		userId, existing.Body, existing.CreatedAt.Format(time.RFC3339), achievedAt.Format(time.RFC3339),
	)

	return 0, nil
}

// findNotificationByBodyContains は、指定カテゴリの通知でbodyにneedleを含むものを返す
// (称号名・ランク名はtierごとに一意なため、これで重複判定できる)。無ければnilを返す。
func findNotificationByBodyContains(db *gorm.DB, userId string, category string, needle string) (*model.Notification, error) {
	var notifications []*model.Notification
	if tx := db.Model(&model.Notification{}).
		Where("user_id = ? AND category = ?", userId, category).
		Where("body LIKE ?", "%"+needle+"%").
		Order("created_at ASC").
		Limit(1).
		Find(&notifications); tx.Error != nil {
		return nil, tx.Error
	}

	if len(notifications) == 0 {
		return nil, nil
	}

	return notifications[0], nil
}

// milestoneAchievedAt はマイルストーン系・週次ストリーク系バッジ定義について、
// シーズン内で実際に閾値へ到達した日付を返す(求められない場合はfallbackを返す)。
func milestoneAchievedAt(
	def *entity.BadgeDefinition,
	recordDates []time.Time,
	matchDates []time.Time,
	deckCodeDates []time.Time,
	streakAchievedAt map[int]time.Time,
	fallback time.Time,
) time.Time {
	switch def.CriteriaType {
	case usecase.BadgeCriteriaTypeRecordCount:
		return nthDate(recordDates, def.CriteriaValue, fallback)
	case usecase.BadgeCriteriaTypeMatchCount:
		return nthDate(matchDates, def.CriteriaValue, fallback)
	case usecase.BadgeCriteriaTypeDeckCodeCount:
		return nthDate(deckCodeDates, def.CriteriaValue, fallback)
	case usecase.BadgeCriteriaTypeStreakWeeks:
		if at, ok := streakAchievedAt[def.CriteriaValue]; ok {
			return at
		}
		return fallback
	default:
		return fallback
	}
}

// nthDate は昇順の dates の n 番目(1始まり)の日付を返す(範囲外ならfallback)。
func nthDate(dates []time.Time, n int, fallback time.Time) time.Time {
	if n <= 0 || n > len(dates) {
		return fallback
	}

	return dates[n-1]
}

// seasonRecordEventDates はシーズン内でevent_dateが設定されているrecordについて、
// TierAsOfでtierの推移を辿るための候補日を重複を除いて昇順で返す(DesignationStats
// Interfaceの各種Countメソッドがevent_date基準で集計するため、event_dateがNULLの
// 記録は称号判定に影響しない=候補日にする必要がない)。
//
// 各recordの候補日(=achievedAtByEventDateで引ける実際の処理時刻)は、
// GREATEST(event_date, デッキ登録日時, その記録に紐づくmatchのcreated_atのうち最も早いもの)
// で求める。称号(駆け出し・見習い等)の達成条件「対戦結果(matches)が1件以上ある」
// 「デッキが登録されている」は、それぞれ対戦結果の追加・デッキの登録という別々の
// タイミングで満たされうる(CountRecordsAsOfByUserIdはevent_date/deck_registered_at/
// matchのcreated_atをそれぞれ独立にasOfと比較するため、record作成後にどちらか一方だけ
// 後から満たされるケースがある)。この記録が実際に両条件を満たした瞬間は、それら
// 個々のタイミングのうち最も遅いものであるため、GREATESTで求める。event_date単独や
// デッキ登録日時単独を候補にすると、他方の条件がまだ満たされていない時点を誤って
// 達成日として拾ってしまう(例: event_dateが古い日付でも、対戦結果の追加はそれより
// 後、というケース)。
//
// デッキ登録日時には records.updated_at ではなく records.deck_registered_at
// (COALESCE先はcreated_at)を使う。updated_atは記録全体のあらゆる編集(メモ修正等の
// デッキ登録と無関係な変更)でも進んでしまうため、これをそのまま使うと、複数の記録を
// まとめて後から編集しただけで無関係な達成日に化けたり、その編集日にまとめて達成日が
// 集中してしまう(=称号・ランクアップ通知が軒並み同じ日時になる)不具合になる。
//
// さらに、紐づくデッキ(deck_id)・デッキコード(deck_code_id)の created_at もGREATESTに
// 含める。deck_registered_at はカラム追加時のマイグレーションで既存記録を created_at で
// 埋めた近似値であり、デッキを後から登録した記録では「記録作成時点で既に登録済み」と
// 扱われてしまう。デッキは記録に登録されるより前に必ず作成されているため、デッキの
// 作成日時を下限に加えないと、デッキがまだ存在しない時点を達成日として拾い、称号・
// ランクアップの通知が「初デッキ」バッジの通知より古くなる(達成条件と逆転する)。
func seasonRecordEventDates(
	db *gorm.DB,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) ([]time.Time, map[time.Time]time.Time, error) {
	type row struct {
		AchievedAt time.Time
	}
	var rows []row

	tx := db.Table("records").
		Select(
			"GREATEST("+
				"records.event_date, "+
				"COALESCE(records.deck_registered_at, records.created_at), "+
				"COALESCE(MIN(matches.created_at), records.created_at), "+
				"COALESCE(decks.created_at, records.created_at), "+
				"COALESCE(deck_codes.created_at, records.created_at)"+
				") AS achieved_at",
		).
		Joins("LEFT JOIN matches ON matches.record_id = records.id AND matches.deleted_at IS NULL").
		Joins("LEFT JOIN decks ON decks.id = records.deck_id").
		Joins("LEFT JOIN deck_codes ON deck_codes.id = records.deck_code_id").
		Where("records.user_id = ? AND records.deleted_at IS NULL AND records.event_date IS NOT NULL", userId).
		Where("records.event_date >= ? AND records.event_date < ?", fromDate, toDate).
		Group("records.id, records.event_date, records.deck_registered_at, records.created_at, decks.created_at, deck_codes.created_at").
		Scan(&rows)
	if tx.Error != nil {
		return nil, nil, tx.Error
	}

	achievedAtByEventDate := make(map[time.Time]time.Time, len(rows))
	for _, r := range rows {
		achievedAtByEventDate[r.AchievedAt] = r.AchievedAt
	}

	dates := make([]time.Time, 0, len(achievedAtByEventDate))
	for d := range achievedAtByEventDate {
		dates = append(dates, d)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })

	return dates, achievedAtByEventDate, nil
}

// designationAchievedAt は candidateDates(シーズン内の各recordが実際に達成条件を
// 満たした時刻。seasonRecordEventDates参照、昇順)を先頭から走査し、その時点までの
// 実績で判定したtier(TierAsOf)がsatisfiesを満たす最初の候補を探す。tierはシーズン内で
// 記録が増えるほど単調非減少のため、最初に見つかった候補が実際の達成日になる。
// 見つからない場合(通常発生しない)はfallbackを返す。
func designationAchievedAt(
	ctx context.Context,
	designationEvaluation usecase.DesignationEvaluationInterface,
	userId string,
	candidateDates []time.Time,
	achievedAtByEventDate map[time.Time]time.Time,
	satisfies func(tier int) bool,
	fallback time.Time,
) time.Time {
	for _, d := range candidateDates {
		// TierAsOfのtoDateはexclusive上限のため、dの記録・編集を含めるには1日後を渡す。
		cutoff := d.AddDate(0, 0, 1)

		tier, err := designationEvaluation.TierAsOf(ctx, userId, cutoff)
		if err != nil {
			continue
		}
		if satisfies(tier) {
			if at, ok := achievedAtByEventDate[d]; ok {
				return at
			}
			return d
		}
	}

	return fallback
}

func saveNotification(
	ctx context.Context,
	notificationRepo repository.NotificationInterface,
	dryRun bool,
	userId string,
	category string,
	title string,
	body string,
	achievedAt time.Time,
) error {
	if dryRun {
		return nil
	}

	id, err := generateId()
	if err != nil {
		return err
	}

	notification := entity.NewNotification(id, achievedAt, userId, category, title, body, notificationLinkUrl)
	notification.IsRead = true
	notification.ReadAt = achievedAt

	return notificationRepo.Save(ctx, notification)
}
