// backfill-notification-history は、通知機能の導入前から既にバッジ・称号・ランクを
// 達成していたユーザーに対し、その実績を「既読済みの通知履歴」として遡って作成するための
// 一回限りの初期投入バッチ。
//
// オンボーディング系バッジ(user_badgesに永続化済み)は実際の achieved_at をそのまま
// created_at/read_at に使う。マイルストーン系・週次ストリーク系バッジ、称号・ランクは
// シーズンごとにライブ集計する仕様(過去の達成日時を保持しない)ため、シーズン内の
// records/matches/decksの日付を遡って走査し、閾値・tierへ実際に到達した日を
// created_at/read_at として1件だけ作成する(見つからない場合のみ本バッチ実行時刻を使う)。
//
// 冪等性: 対象ユーザーが既に1件でも notifications 行を持つ場合はスキップする。これにより
// 誤って複数回実行しても通知が重複しない(backfill-badges が再実行で通知を重複生成しうる
// 問題を踏まえた設計)。ただし裏を返すと、対象ユーザーが導入後に何らかの通知を既に
// 受け取っている場合はバックフィル対象外になる(導入前に一度だけ実行する運用を想定)。
//
// 使い方:
//
//	# 変更内容を書き込まずに確認するだけ(デフォルト)
//	go run ./cmd/backfill-notification-history
//
//	# 実際に notifications へ反映する
//	go run ./cmd/backfill-notification-history -dry-run=false
//
//	# 特定ユーザーのみ対象にする(調査・検証用)
//	go run ./cmd/backfill-notification-history -user-id=xxxxx -dry-run=false
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

	if *dryRun {
		log.Printf("[dry-run] checking notification history for %d users (書き込みは行いません)\n", len(userIds))
	} else {
		log.Printf("backfilling notification history for %d users\n", len(userIds))
	}

	backfilled := 0
	for _, userId := range userIds {
		created, err := backfillUser(
			ctx, db, notificationRepo, badgeStatsRepo, badgeUsecase, designationUsecase, designationEvaluation,
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

// backfillUser は1ユーザー分の通知履歴を作成する。作成した(dry-runなら作成予定の)件数を返す。
func backfillUser(
	ctx context.Context,
	db *gorm.DB,
	notificationRepo repository.NotificationInterface,
	badgeStatsRepo repository.BadgeStatsInterface,
	badgeUsecase usecase.BadgeInterface,
	designationUsecase usecase.DesignationInterface,
	designationEvaluation usecase.DesignationEvaluationInterface,
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

	// バッジ(1・2)は個別の重複チェックを持たないため、従来通り「1件でも通知があれば
	// 丸ごとスキップ」する。称号・ランク(3)は、既にバッジ通知だけ受け取っている
	// ユーザーの称号履歴だけが欠落する問題(達成tierを飛び越えた場合に最終tierしか
	// 通知されなかった等)があるため、existingCountに関わらず個別に判定する。
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
		// 「実際に達成した日」を遡って求めるため、シーズン内のrecord/match/deckの日付を
		// 昇順に並べ、閾値に到達した時点の日付を使う(週次ストリークはStreakWeeksAchievedAtで求める)。
		badgeViews, err := badgeUsecase.GetByUserId(ctx, userId, "")
		if err != nil {
			return created, err
		}

		hasMilestoneOrStreak := false
		for _, view := range badgeViews {
			if view.Definition.Category != usecase.BadgeCategoryOnboarding && view.Achieved {
				hasMilestoneOrStreak = true
				break
			}
		}

		var recordDates, matchDates, deckDates []time.Time
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

			deckDates, err = badgeStatsRepo.FindDeckDatesByUserId(ctx, userId, seasonFromDate, seasonToDate)
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

			achievedAt := milestoneAchievedAt(view.Definition, recordDates, matchDates, deckDates, streakAchievedAt, now)

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
	designationCreated, err := backfillDesignationHistory(
		ctx, db, notificationRepo, designationUsecase, designationEvaluation,
		userId, seasonLabel, seasonFromDate, seasonToDate, now, dryRun,
	)
	if err != nil {
		return created, err
	}
	created += designationCreated

	return created, nil
}

// backfillDesignationHistory はユーザーが現在のシーズンで通過済みの称号tier・ランクの
// うち、まだ通知(notifications)が無いものだけを個別に補完する。称号名・ランク名は
// tierごとに一意なため、既存通知のbody文字列にその名前が含まれるかで重複判定する
// (backfillUser本体と異なり、他カテゴリ(バッジ)の通知が既にあるユーザーでも対象にする)。
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

	candidateDates, err := seasonRecordEventDates(db, userId, seasonFromDate, seasonToDate)
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

		exists, err := notificationBodyContainsExists(db, userId, usecase.NotificationCategoryDesignation, def.Name)
		if err != nil {
			return created, err
		}
		if !exists {
			achievedAt := designationAchievedAt(
				ctx, designationEvaluation, userId, candidateDates,
				func(t int) bool { return t >= tier },
				now,
			)

			if err := saveNotification(
				ctx, notificationRepo, dryRun,
				userId, usecase.NotificationCategoryDesignation, "称号を獲得しました",
				fmt.Sprintf("%sシーズンで称号「%s %s」を獲得しました！", seasonLabel, def.Emoji, def.Name),
				achievedAt,
			); err != nil {
				return created, err
			}
			created++
		}

		rankName := usecase.RankNameForTier(tier)
		if rankName == "" || notifiedRanks[rankName] {
			continue
		}
		notifiedRanks[rankName] = true

		rankExists, err := notificationBodyContainsExists(db, userId, usecase.NotificationCategoryRank, rankName)
		if err != nil {
			return created, err
		}
		if rankExists {
			continue
		}

		rankAt := designationAchievedAt(
			ctx, designationEvaluation, userId, candidateDates,
			func(t int) bool { return usecase.RankNameForTier(t) == rankName },
			now,
		)

		if err := saveNotification(
			ctx, notificationRepo, dryRun,
			userId, usecase.NotificationCategoryRank, "ランクが上がりました",
			fmt.Sprintf("%sシーズンでランクが「%s」に上がりました！", seasonLabel, rankName),
			rankAt,
		); err != nil {
			return created, err
		}
		created++
	}

	return created, nil
}

// notificationBodyContainsExists は、指定カテゴリの通知でbodyにneedleを含むものが
// 既に存在するかを返す(称号名・ランク名はtierごとに一意なため、これで重複判定できる)。
func notificationBodyContainsExists(db *gorm.DB, userId string, category string, needle string) (bool, error) {
	var count int64
	if tx := db.Model(&model.Notification{}).
		Where("user_id = ? AND category = ?", userId, category).
		Where("body LIKE ?", "%"+needle+"%").
		Count(&count); tx.Error != nil {
		return false, tx.Error
	}

	return count > 0, nil
}

// milestoneAchievedAt はマイルストーン系・週次ストリーク系バッジ定義について、
// シーズン内で実際に閾値へ到達した日付を返す(求められない場合はfallbackを返す)。
func milestoneAchievedAt(
	def *entity.BadgeDefinition,
	recordDates []time.Time,
	matchDates []time.Time,
	deckDates []time.Time,
	streakAchievedAt map[int]time.Time,
	fallback time.Time,
) time.Time {
	switch def.CriteriaType {
	case usecase.BadgeCriteriaTypeRecordCount:
		return nthDate(recordDates, def.CriteriaValue, fallback)
	case usecase.BadgeCriteriaTypeMatchCount:
		return nthDate(matchDates, def.CriteriaValue, fallback)
	case usecase.BadgeCriteriaTypeDeckCount:
		return nthDate(deckDates, def.CriteriaValue, fallback)
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

// seasonRecordEventDates はシーズン内でevent_dateが設定されているrecordの日付を
// 重複を除いて昇順で返す。称号・ランクの実際の達成日を遡って求めるための候補日として使う
// (DesignationStatsInterfaceの各種Countメソッドがevent_date基準で集計するため、
// event_dateがNULLの記録は称号判定に影響しない=候補日にする必要がない)。
func seasonRecordEventDates(db *gorm.DB, userId string, fromDate time.Time, toDate time.Time) ([]time.Time, error) {
	var dates []time.Time
	tx := db.Table("records").
		Where("user_id = ? AND deleted_at IS NULL AND event_date IS NOT NULL", userId).
		Where("event_date >= ? AND event_date < ?", fromDate, toDate).
		Distinct().
		Order("event_date ASC").
		Pluck("event_date", &dates)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return dates, nil
}

// designationAchievedAt は candidateDates(シーズン内のrecord.event_date、昇順)を先頭から
// 走査し、その日までの実績で判定したtier(TierAsOf)がsatisfiesを満たす最初の日を返す。
// tierはシーズン内で記録が増えるほど単調非減少のため、最初に見つかった日が実際の達成日になる。
// 見つからない場合(通常発生しない)はfallbackを返す。
func designationAchievedAt(
	ctx context.Context,
	designationEvaluation usecase.DesignationEvaluationInterface,
	userId string,
	candidateDates []time.Time,
	satisfies func(tier int) bool,
	fallback time.Time,
) time.Time {
	for _, d := range candidateDates {
		// TierAsOfのtoDateはexclusive上限のため、dの記録を含めるには翌日0時を渡す。
		cutoff := d.AddDate(0, 0, 1)

		tier, err := designationEvaluation.TierAsOf(ctx, userId, cutoff)
		if err != nil {
			continue
		}
		if satisfies(tier) {
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
