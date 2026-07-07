// backfill-badges は、オンボーディング系バッジ(はじめの一歩: signup/first_deck/
// first_record/first_match)の機能導入前から既に達成条件を満たしていたユーザーに対し、
// user_badges へ遡ってバッジを付与するための一回限りの初期投入バッチ。
//
// オンボーディング系バッジは usecase.BadgeEvaluation.award がユーザー登録・デッキ登録・
// 記録作成・対戦作成のAPIリクエスト処理内でリアルタイムに評価・付与する仕様のため、
// 機能導入前から該当条件を満たしていた既存ユーザーには一度も付与されない。本バッチは
// その欠落分を、award() 呼び出し側(badge_evaluation.go)と同じ基準で計算した実際の
// 達成日時を使って補完する。
//
// 正しい achieved_at の求め方は badge_evaluation.go の award() 呼び出し側と同じ基準を使う:
//   - signup:       users.created_at
//   - deck_count:   criteria_value 番目に古い deck の created_at
//   - record_count: criteria_value 番目に古い record の created_at(event_dateは使わない。
//     backfill入力等でachieved_atが過去日にずれるのを避けるための仕様、EvaluateOnRecordCreated参照)
//   - match_count:  criteria_value 番目に古い match の created_at
//
// 冪等性: UserBadgeInterface.Save は単純な INSERT(upsertではない)ため、既に user_badges
// に行があるユーザー・バッジの組は事前にスキップし、重複INSERTしない。
//
// 通知(notifications)は作成しない。通知履歴の補完は cmd/backfill-notification-history が
// 別途 user_badges を読んで行う(役割分担)。
//
// 使い方:
//
//	# 変更内容を書き込まずに確認するだけ(デフォルト)
//	go run ./cmd/backfill-badges
//
//	# 実際に user_badges へ反映する
//	go run ./cmd/backfill-badges -dry-run=false
//
//	# 特定ユーザーのみ対象にする(調査・検証用)
//	go run ./cmd/backfill-badges -user-id=xxxxx -dry-run=false
package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
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

const badgeCategoryOnboarding = "onboarding"

var entropy = rand.New(rand.NewSource(time.Now().UnixNano()))

func generateId() (string, error) {
	ms := ulid.Timestamp(time.Now())
	id, err := ulid.New(ms, entropy)

	return id.String(), err
}

func main() {
	dryRun := flag.Bool("dry-run", true, "true の場合、書き込みは行わず差分の確認のみ行う")
	targetUserId := flag.String("user-id", "", "指定した場合、そのユーザーのみを対象にする(未指定なら全ユーザー)")
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

	q := db.Model(&model.User{})
	if *targetUserId != "" {
		q = q.Where("id = ?", *targetUserId)
	}

	var users []*model.User
	if tx := q.Order("id ASC").Find(&users); tx.Error != nil {
		log.Printf("failed to list users: %v\n", tx.Error)
		os.Exit(ExitCodeNG)
	}

	if *dryRun {
		log.Printf("[dry-run] checking %d users against %d onboarding badge definitions (書き込みは行いません)\n", len(users), len(onboardingDefs))
	} else {
		log.Printf("backfilling badges for %d users against %d onboarding badge definitions\n", len(users), len(onboardingDefs))
	}

	userBadgeRepo := infrastructure.NewUserBadge(db)

	backfilled := 0
	for _, user := range users {
		created, err := backfillUser(context.Background(), db, userBadgeRepo, user, onboardingDefs, *dryRun)
		if err != nil {
			log.Printf("failed to backfill user=%s: %v\n", user.ID, err)
			continue
		}
		if created > 0 {
			backfilled++
		}
	}

	if *dryRun {
		log.Printf("[dry-run] completed: %d/%d users have badges to backfill\n", backfilled, len(users))
	} else {
		log.Printf("completed: backfilled %d/%d users\n", backfilled, len(users))
	}

	os.Exit(ExitCodeOK)
}

// backfillUser は1ユーザー分のオンボーディング系バッジを補完する。作成した
// (dry-runなら作成予定の)件数を返す。
func backfillUser(
	ctx context.Context,
	db *gorm.DB,
	userBadgeRepo repository.UserBadgeInterface,
	user *model.User,
	onboardingDefs []*model.BadgeDefinition,
	dryRun bool,
) (int, error) {
	var existing []*model.UserBadge
	if tx := db.Where("user_id = ?", user.ID).Find(&existing); tx.Error != nil {
		return 0, tx.Error
	}
	achieved := make(map[string]bool, len(existing))
	for _, ub := range existing {
		achieved[ub.BadgeDefinitionId] = true
	}

	created := 0
	for _, def := range onboardingDefs {
		if achieved[def.ID] {
			continue
		}

		achievedAt, recordId, ok, err := achievedAtForCriteria(db, user, def)
		if err != nil {
			return created, err
		}
		if !ok {
			continue
		}

		if dryRun {
			log.Printf("[dry-run] user=%s badge=%s 未付与(達成日=%s)\n", user.ID, def.Code, achievedAt.Format(time.RFC3339))
			created++
			continue
		}

		id, err := generateId()
		if err != nil {
			return created, err
		}

		userBadge := entity.NewUserBadge(id, time.Now().Local(), user.ID, def.ID, recordId, achievedAt)
		if err := userBadgeRepo.Save(ctx, userBadge); err != nil {
			return created, err
		}

		log.Printf("user=%s badge=%s BACKFILLED achieved_at=%s\n", user.ID, def.Code, achievedAt.Format(time.RFC3339))
		created++
	}

	return created, nil
}

// achievedAtForCriteria は badge_definitions.criteria_type に応じて、そのユーザーが
// criteria_value 番目の条件を実際に満たした日時(と紐づく record_id)を求める。
// 該当する記録が criteria_value 件に満たない場合(まだ未達成)は ok=false を返す。
func achievedAtForCriteria(
	db *gorm.DB,
	user *model.User,
	def *model.BadgeDefinition,
) (time.Time, string, bool, error) {
	switch def.CriteriaType {
	case usecase.BadgeCriteriaTypeSignup:
		return user.CreatedAt, "", true, nil

	case usecase.BadgeCriteriaTypeDeckCount:
		var decks []*model.Deck
		if tx := db.Where("user_id = ? AND deleted_at IS NULL", user.ID).
			Order("created_at ASC").
			Find(&decks); tx.Error != nil {
			return time.Time{}, "", false, tx.Error
		}
		if len(decks) < def.CriteriaValue {
			return time.Time{}, "", false, nil
		}
		// デッキ起点のバッジ獲得のため、紐づく record は存在しない(EvaluateOnDeckCreated参照)。
		return decks[def.CriteriaValue-1].CreatedAt, "", true, nil

	case usecase.BadgeCriteriaTypeRecordCount:
		var records []*model.Record
		if tx := db.Where("user_id = ? AND deleted_at IS NULL", user.ID).
			Order("created_at ASC").
			Find(&records); tx.Error != nil {
			return time.Time{}, "", false, tx.Error
		}
		if len(records) < def.CriteriaValue {
			return time.Time{}, "", false, nil
		}
		r := records[def.CriteriaValue-1]
		return r.CreatedAt, r.ID, true, nil

	case usecase.BadgeCriteriaTypeMatchCount:
		var matches []*model.Match
		if tx := db.Where("user_id = ? AND deleted_at IS NULL", user.ID).
			Order("created_at ASC").
			Find(&matches); tx.Error != nil {
			return time.Time{}, "", false, tx.Error
		}
		if len(matches) < def.CriteriaValue {
			return time.Time{}, "", false, nil
		}
		m := matches[def.CriteriaValue-1]
		return m.CreatedAt, m.RecordId, true, nil

	default:
		return time.Time{}, "", false, nil
	}
}
