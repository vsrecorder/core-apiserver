// repair-badge-achieved-at は、backfill-badges の不具合(usecase.BadgeEvaluation.award が
// achieved_at に条件達成時点の日時ではなく常に time.Now() を書き込んでいた)によって
// user_badges.achieved_at が誤った日付(バッチ実行日)になってしまったオンボーディング系
// バッジ(はじめの一歩: signup/first_deck/first_record/first_match)を、実際の達成日時に
// 修正するための一回限りの復旧バッチ。
//
// マイルストーン系・週次ストリーク系は一覧取得時にライブ集計する仕様(usecase/badge.go)で
// user_badges に永続化されないため、本ツールの対象は category="onboarding" のみで良い。
//
// backfill-badges 自体は award() が「未獲得のバッジのみ新規INSERT」するロジックのため、
// 既に誤った日付で入っている行は再実行しても上書きされない(UserBadge.Save は
// db.Create のみで upsert ではない)。そのため本ツールで明示的に UPDATE する。
//
// 正しい achieved_at の求め方は badge_evaluation.go の award() 呼び出し側と同じ基準を使う:
//   - signup:       users.created_at
//   - deck_count:   criteria_value 番目に古い deck の created_at
//   - record_count: criteria_value 番目に古い record の created_at(記録した日時。event_dateは
//     過去の対戦日を表す入力値であり見ない)
//   - match_count:  criteria_value 番目に古い match の created_at
//
// 使い方:
//
//	# 変更内容を書き込まずに確認するだけ(デフォルト)
//	go run ./cmd/repair-badge-achieved-at
//
//	# 実際に user_badges へ反映する
//	go run ./cmd/repair-badge-achieved-at -dry-run=false
//
//	# 特定ユーザーのみ対象にする(調査・検証用)
//	go run ./cmd/repair-badge-achieved-at -user-id=xxxxx -dry-run=false
package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

const badgeCategoryOnboarding = "onboarding"

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

	var onboardingDefs []*model.BadgeDefinition
	if tx := db.Where("category = ?", badgeCategoryOnboarding).Find(&onboardingDefs); tx.Error != nil {
		log.Printf("failed to list onboarding badge definitions: %v\n", tx.Error)
		os.Exit(ExitCodeNG)
	}

	defById := make(map[string]*model.BadgeDefinition, len(onboardingDefs))
	defIds := make([]string, 0, len(onboardingDefs))
	for _, def := range onboardingDefs {
		defById[def.ID] = def
		defIds = append(defIds, def.ID)
	}

	q := db.Where("badge_definition_id IN ?", defIds)
	if *targetUserId != "" {
		q = q.Where("user_id = ?", *targetUserId)
	}

	var userBadges []*model.UserBadge
	if tx := q.Order("user_id ASC").Find(&userBadges); tx.Error != nil {
		log.Printf("failed to list user_badges: %v\n", tx.Error)
		os.Exit(ExitCodeNG)
	}

	if *dryRun {
		log.Printf("[dry-run] checking %d onboarding user_badges rows (書き込みは行いません)\n", len(userBadges))
	} else {
		log.Printf("repairing %d onboarding user_badges rows\n", len(userBadges))
	}

	fixed := 0
	for _, ub := range userBadges {
		def := defById[ub.BadgeDefinitionId]
		if def == nil {
			continue
		}

		correct, ok, err := correctAchievedAt(db, ub.UserId, def)
		if err != nil {
			log.Printf("failed to compute correct achieved_at: user=%s badge=%s err=%v\n", ub.UserId, def.Code, err)
			continue
		}
		if !ok {
			log.Printf("skip: user=%s badge=%s に対応する達成イベントが見つかりません\n", ub.UserId, def.Code)
			continue
		}

		if correct.Equal(ub.AchievedAt) {
			continue
		}

		fixed++

		if *dryRun {
			log.Printf("[dry-run] user=%s badge=%s MISMATCH before=%s after=%s\n",
				ub.UserId, def.Code, ub.AchievedAt.Format(time.RFC3339), correct.Format(time.RFC3339))
			continue
		}

		if tx := db.Model(&model.UserBadge{}).Where("id = ?", ub.ID).Update("achieved_at", correct); tx.Error != nil {
			log.Printf("failed to update user_badges id=%s: %v\n", ub.ID, tx.Error)
			continue
		}

		log.Printf("user=%s badge=%s REPAIRED before=%s after=%s\n",
			ub.UserId, def.Code, ub.AchievedAt.Format(time.RFC3339), correct.Format(time.RFC3339))
	}

	if *dryRun {
		log.Printf("[dry-run] completed: %d/%d rows have mismatched achieved_at\n", fixed, len(userBadges))
	} else {
		log.Printf("completed: repaired %d/%d rows\n", fixed, len(userBadges))
	}

	os.Exit(ExitCodeOK)
}

// correctAchievedAt は badge_definitions.criteria_type に応じて、そのユーザーが本来の
// criteria_value 番目の条件を満たした実際の日時を求める。該当する記録が
// criteria_value 件に満たない場合(データ不整合等)は ok=false を返す。
func correctAchievedAt(
	db *gorm.DB,
	userId string,
	def *model.BadgeDefinition,
) (time.Time, bool, error) {
	switch def.CriteriaType {
	case usecase.BadgeCriteriaTypeSignup:
		var user model.User
		if tx := db.Where("id = ?", userId).First(&user); tx.Error != nil {
			if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
				return time.Time{}, false, nil
			}
			return time.Time{}, false, tx.Error
		}
		return user.CreatedAt, true, nil

	case usecase.BadgeCriteriaTypeDeckCount:
		var decks []*model.Deck
		if tx := db.Where("user_id = ? AND deleted_at IS NULL", userId).
			Order("created_at ASC").
			Find(&decks); tx.Error != nil {
			return time.Time{}, false, tx.Error
		}
		if len(decks) < def.CriteriaValue {
			return time.Time{}, false, nil
		}
		return decks[def.CriteriaValue-1].CreatedAt, true, nil

	case usecase.BadgeCriteriaTypeRecordCount:
		var records []*model.Record
		if tx := db.Where("user_id = ? AND deleted_at IS NULL", userId).
			Order("created_at ASC").
			Find(&records); tx.Error != nil {
			return time.Time{}, false, tx.Error
		}
		if len(records) < def.CriteriaValue {
			return time.Time{}, false, nil
		}
		return records[def.CriteriaValue-1].CreatedAt, true, nil

	case usecase.BadgeCriteriaTypeMatchCount:
		var matches []*model.Match
		if tx := db.Where("user_id = ? AND deleted_at IS NULL", userId).
			Order("created_at ASC").
			Find(&matches); tx.Error != nil {
			return time.Time{}, false, tx.Error
		}
		if len(matches) < def.CriteriaValue {
			return time.Time{}, false, nil
		}
		return matches[def.CriteriaValue-1].CreatedAt, true, nil

	default:
		return time.Time{}, false, nil
	}
}
