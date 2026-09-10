// repair-streaks は、何らかの理由で user_streaks が現存の records と食い違ってしまった
// 場合に、全対象ユーザーの週次ストリーク状態を records から作り直すための復旧バッチ。
//
// 本ツールは EvaluateOnRecordDeleted と同じ「現存する records の日付からゼロから
// 再計算し、行ごと上書きする」ロジック(usecase.ComputeStreakState)を、削除以外の
// トリガーからも使えるようにしたものである。
//
// 使い方:
//
//	# 変更内容を書き込まずに確認するだけ(デフォルト)
//	go run ./cmd/repair-streaks
//
//	# 実際に user_streaks へ反映する
//	go run ./cmd/repair-streaks -dry-run=false
//
//	# 特定ユーザーのみ対象にする(調査・検証用)
//	go run ./cmd/repair-streaks -user-id=xxxxx -dry-run=false
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/logging"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const appName = "repair-streaks"

const (
	ExitCodeOK = iota
	ExitCodeNG
)

func main() {
	dryRun := flag.Bool("dry-run", true, "true の場合、書き込みは行わず差分の確認のみ行う")
	targetUserId := flag.String("user-id", "", "指定した場合、そのユーザーのみを対象にする(未指定なら全対象ユーザー)")
	// ログは cmd/core-apiserver と同じJSON形式に揃える。これを呼ばないと slog の
	// 既定ハンドラ(テキスト)のままになり、usecase 層のログから layer やソース位置が落ちる。
	slog.SetDefault(logging.InitLogger(logging.Config{
		Level:   "info",
		AppName: appName,
	}))

	flag.Parse()

	// .env が無くても環境変数から設定できるため、読み込み失敗は起動を止めない。
	if err := godotenv.Load(); err != nil {
		slog.Warn("failed to load .env file", logging.Err(err))
	}

	db, err := postgres.NewDB(
		os.Getenv("DB_HOSTNAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER_NAME"),
		os.Getenv("DB_USER_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	if err != nil {
		slog.Error("failed to connect database", logging.Err(err))
		os.Exit(ExitCodeNG)
	}

	badgeStatsRepo := infrastructure.NewBadgeStats(db)
	userStreakRepo := infrastructure.NewUserStreak(db)

	ctx := context.Background()

	var userIds []string
	if *targetUserId != "" {
		userIds = []string{*targetUserId}
	} else {
		userIds, err = findTargetUserIds(db)
		if err != nil {
			slog.Error("failed to list users", logging.Err(err))
			os.Exit(ExitCodeNG)
		}
	}

	// dry-run はメッセージではなく属性で出す(分岐させると grep の条件が増える)
	batchAttrs := []any{
		slog.Int("target_users", len(userIds)),
		slog.Bool("dry_run", *dryRun),
	}
	slog.Info("repairing streaks", batchAttrs...)

	mismatched := 0
	for _, userId := range userIds {
		changed, err := repairUser(ctx, badgeStatsRepo, userStreakRepo, userId, *dryRun)
		if err != nil {
			slog.Error("failed to repair user", slog.String("user_id", userId), logging.Err(err))
			continue
		}
		if changed {
			mismatched++
		}
	}

	slog.Info("completed", append(batchAttrs, slog.Int("mismatched_users", mismatched))...)

	os.Exit(ExitCodeOK)
}

// findTargetUserIds は「既に user_streaks 行を持つユーザー」と「現存する record を持つ
// ユーザー」の和集合を返す。前者は既存行が古いままになっていないかの確認対象、後者は
// user_streaks 行がまだ無い(が本来あるべき)ユーザーの取りこぼしを防ぐための対象。
func findTargetUserIds(db *gorm.DB) ([]string, error) {
	seen := make(map[string]struct{})
	var userIds []string

	tablesAndConds := map[string]string{
		"user_streaks": "",
		"records":      "deleted_at IS NULL",
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
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			userIds = append(userIds, id)
		}
	}

	return userIds, nil
}

// repairUser は指定ユーザーについて、現存する records から正しい週次ストリーク状態を
// 再計算し、既存の user_streaks と食い違っていれば(dryRun=false のときのみ)上書き保存する。
// 戻り値の bool は「既存の状態と食い違っていたか」を表す。
func repairUser(
	ctx context.Context,
	badgeStatsRepo repository.BadgeStatsInterface,
	userStreakRepo repository.UserStreakInterface,
	userId string,
	dryRun bool,
) (bool, error) {
	before, err := userStreakRepo.FindByUserId(ctx, userId)
	if err != nil {
		if !errors.Is(err, apperror.ErrRecordNotFound) {
			return false, err
		}
		before = nil
	}

	dates, err := badgeStatsRepo.FindRecordDatesByUserId(ctx, userId, time.Time{}, time.Time{})
	if err != nil {
		return false, err
	}

	currentWeeks, longestWeeks, freezeUsedCount, freezeRegenProgress, lastRecordedWeek := usecase.ComputeStreakState(dates)

	changed := before == nil ||
		before.CurrentWeeks != currentWeeks ||
		before.LongestWeeks != longestWeeks ||
		before.FreezeUsedCount != freezeUsedCount ||
		before.FreezeRegenProgress != freezeRegenProgress ||
		!before.LastRecordedWeek.Equal(lastRecordedWeek)

	if !changed {
		return false, nil
	}

	beforeState := "(なし)"
	if before != nil {
		beforeState = formatStreak(before.CurrentWeeks, before.LongestWeeks, before.FreezeUsedCount, before.FreezeRegenProgress, before.LastRecordedWeek)
	}
	afterState := formatStreak(currentWeeks, longestWeeks, freezeUsedCount, freezeRegenProgress, lastRecordedWeek)

	mismatchAttrs := []any{
		slog.String("user_id", userId),
		slog.String("before", beforeState),
		slog.String("after", afterState),
		slog.Int("live_records", len(dates)),
	}

	if dryRun {
		slog.Info("user_streaks mismatch found", append(mismatchAttrs, slog.Bool("dry_run", true))...)
		return true, nil
	}

	streak := entity.NewUserStreak(userId, currentWeeks, longestWeeks, freezeUsedCount, freezeRegenProgress, lastRecordedWeek, time.Now().Local())
	if err := userStreakRepo.Save(ctx, streak); err != nil {
		return false, err
	}

	slog.Info("user_streaks repaired", append(mismatchAttrs, slog.Bool("dry_run", false))...)
	return true, nil
}

func formatStreak(currentWeeks, longestWeeks, freezeUsedCount, freezeRegenProgress int, lastRecordedWeek time.Time) string {
	week := "なし"
	if !lastRecordedWeek.IsZero() {
		week = lastRecordedWeek.Format("2006-01-02")
	}
	return "current=" + strconv.Itoa(currentWeeks) + " longest=" + strconv.Itoa(longestWeeks) + " freeze=" + strconv.Itoa(freezeUsedCount) + " regen=" + strconv.Itoa(freezeRegenProgress) + " last_week=" + week
}
