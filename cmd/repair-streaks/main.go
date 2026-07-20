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
	"log"
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
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

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

	badgeStatsRepo := infrastructure.NewBadgeStats(db)
	userStreakRepo := infrastructure.NewUserStreak(db)

	ctx := context.Background()

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
		log.Printf("[dry-run] checking streak consistency for %d users (書き込みは行いません)\n", len(userIds))
	} else {
		log.Printf("repairing streaks for %d users\n", len(userIds))
	}

	mismatched := 0
	for _, userId := range userIds {
		changed, err := repairUser(ctx, badgeStatsRepo, userStreakRepo, userId, *dryRun)
		if err != nil {
			log.Printf("failed to repair user=%s: %v\n", userId, err)
			continue
		}
		if changed {
			mismatched++
		}
	}

	if *dryRun {
		log.Printf("[dry-run] completed: %d/%d users have mismatched user_streaks\n", mismatched, len(userIds))
	} else {
		log.Printf("completed: repaired %d/%d users\n", mismatched, len(userIds))
	}

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

	if dryRun {
		log.Printf("[dry-run] user=%s MISMATCH before=%s after=%s live_records=%d\n", userId, beforeState, afterState, len(dates))
		return true, nil
	}

	streak := entity.NewUserStreak(userId, currentWeeks, longestWeeks, freezeUsedCount, freezeRegenProgress, lastRecordedWeek, time.Now().Local())
	if err := userStreakRepo.Save(ctx, streak); err != nil {
		return false, err
	}

	log.Printf("user=%s REPAIRED before=%s after=%s live_records=%d\n", userId, beforeState, afterState, len(dates))
	return true, nil
}

func formatStreak(currentWeeks, longestWeeks, freezeUsedCount, freezeRegenProgress int, lastRecordedWeek time.Time) string {
	week := "なし"
	if !lastRecordedWeek.IsZero() {
		week = lastRecordedWeek.Format("2006-01-02")
	}
	return "current=" + strconv.Itoa(currentWeeks) + " longest=" + strconv.Itoa(longestWeeks) + " freeze=" + strconv.Itoa(freezeUsedCount) + " regen=" + strconv.Itoa(freezeRegenProgress) + " last_week=" + week
}
