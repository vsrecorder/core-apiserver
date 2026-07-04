// backfill-badges は施策D(記録ストリーク・実績バッジ)の導入時に、既存ユーザーの過去の
// decks/records/matches からストリーク状態(StreakPanel用の全期間ストリーク)と
// オンボーディング系実績バッジを遡及計算するための一回限りのバッチ。
// マイルストーン系・週次ストリーク系バッジ、および称号はシーズンごとに一覧取得時
// (GET)にライブ集計する仕様のため、ここでの遡及計算は不要。
// badge_definitions/user_badges/user_streaks テーブル作成後、本番投入前に
// 一度だけ実行することを想定している(BADGE_STREAK_PLAN.md フェーズ1)。
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

func main() {
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

	badgeEvaluation := usecase.NewBadgeEvaluation(
		infrastructure.NewBadgeDefinition(db),
		infrastructure.NewUserBadge(db),
		infrastructure.NewUserStreak(db),
		infrastructure.NewBadgeStats(db),
	)

	ctx := context.Background()

	userIds, err := findUserIdsToBackfill(db)
	if err != nil {
		log.Printf("failed to list users: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	log.Printf("backfilling badges/streaks for %d users\n", len(userIds))

	for _, userId := range userIds {
		if err := backfillUser(ctx, db, badgeEvaluation, userId); err != nil {
			log.Printf("failed to backfill user=%s: %v\n", userId, err)
			continue
		}
	}

	log.Println("backfill completed")
	os.Exit(ExitCodeOK)
}

// findUserIdsToBackfill は users/records/matches/decks のいずれかに履歴を持つユーザーIDを
// 重複なく返す(デッキだけ登録して記録がまだ無いユーザーの「初デッキ」や、何も記録していない
// ユーザーの「バトレコユーザー」も遡及対象に含めるため)。
func findUserIdsToBackfill(db *gorm.DB) ([]string, error) {
	seen := make(map[string]struct{})
	var userIds []string

	// users テーブルのみ主キー列が "id"、他は "user_id" のため列名を出し分ける
	tablesAndColumns := map[string]string{
		"users":   "id",
		"records": "user_id",
		"matches": "user_id",
		"decks":   "user_id",
	}
	for table, column := range tablesAndColumns {
		var ids []string
		if tx := db.Table(table).
			Where("deleted_at IS NULL").
			Distinct(column).
			Pluck(column, &ids); tx.Error != nil {
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

// backfillUser は対象ユーザーの decks/records/matches を時系列順に再生し、
// ストリーク状態(週単位の連続数)とバッジ獲得状況を本番と同じロジックで再計算する。
// record_count 等の閾値系はDB上の現在の集計値を都度参照するため実行順に依存しないが、
// ストリークは週の遷移に依存するステートフルな計算のため昇順再生が必須。
func backfillUser(
	ctx context.Context,
	db *gorm.DB,
	badgeEvaluation usecase.BadgeEvaluationInterface,
	userId string,
) error {
	var deckModels []*model.Deck
	if tx := db.Where("user_id = ? AND deleted_at IS NULL", userId).
		Order("created_at ASC").
		Find(&deckModels); tx.Error != nil {
		return tx.Error
	}

	var recordModels []*model.Record
	if tx := db.Where("user_id = ? AND deleted_at IS NULL", userId).
		Order("event_date ASC NULLS FIRST, created_at ASC").
		Find(&recordModels); tx.Error != nil {
		return tx.Error
	}

	var matchModels []*model.Match
	if tx := db.Where("user_id = ? AND deleted_at IS NULL", userId).
		Order("created_at ASC").
		Find(&matchModels); tx.Error != nil {
		return tx.Error
	}

	userCreatedAt, err := userCreatedAtForBackfill(db, userId, deckModels, recordModels, matchModels)
	if err != nil {
		return err
	}

	if _, err := badgeEvaluation.EvaluateOnUserCreated(ctx, userId, userCreatedAt); err != nil {
		return err
	}

	for _, m := range deckModels {
		deck := entity.NewDeck(
			m.ID,
			m.CreatedAt,
			m.ArchivedAt.Time,
			m.UserId,
			m.Name,
			m.PrivateFlg,
			nil,
			nil,
		)

		if _, err := badgeEvaluation.EvaluateOnDeckCreated(ctx, userId, deck); err != nil {
			return err
		}
	}

	for _, m := range recordModels {
		record := entity.NewRecord(
			m.ID,
			m.CreatedAt,
			m.OfficialEventId,
			m.TonamelEventId,
			m.FriendId,
			m.UnofficialEventId,
			m.UserId,
			m.DeckId,
			m.DeckCodeId,
			m.EventDate,
			m.PrivateFlg,
			m.TCGMeisterURL,
			m.Memo,
		)

		if _, err := badgeEvaluation.EvaluateOnRecordCreated(ctx, userId, record); err != nil {
			return err
		}
	}

	for _, m := range matchModels {
		match := entity.NewMatch(
			m.ID,
			m.CreatedAt,
			m.RecordId,
			m.DeckId,
			m.DeckCodeId,
			m.UserId,
			m.OpponentsUserId,
			m.BO3Flg,
			m.GroupMatchFlg,
			m.QualifyingRoundFlg,
			m.FinalTournamentFlg,
			m.DefaultVictoryFlg,
			m.DefaultDefeatFlg,
			m.VictoryFlg,
			m.GroupMatchVictoryFlg,
			m.OpponentsDeckInfo,
			m.Memo,
			nil,
			nil,
		)

		if _, err := badgeEvaluation.EvaluateOnMatchCreated(ctx, userId, match); err != nil {
			return err
		}
	}

	slog.Default().Info("backfilled user", "user_id", userId, "decks", len(deckModels), "records", len(recordModels), "matches", len(matchModels))

	return nil
}

// userCreatedAtForBackfill はユーザーの「ユーザ登録」バッジの達成日として使う実際の登録日時を返す。
// users テーブルに行が無い場合(削除済み等)は、決め手として残っているdecks/records/matchesの
// うち最も古い作成日時で代用する。それも無ければ現在時刻にフォールバックする。
func userCreatedAtForBackfill(
	db *gorm.DB,
	userId string,
	deckModels []*model.Deck,
	recordModels []*model.Record,
	matchModels []*model.Match,
) (time.Time, error) {
	var userModel model.User
	tx := db.Where("id = ?", userId).First(&userModel)
	if tx.Error == nil {
		return userModel.CreatedAt, nil
	}
	if !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return time.Time{}, tx.Error
	}

	earliest := time.Time{}
	consider := func(t time.Time) {
		if earliest.IsZero() || t.Before(earliest) {
			earliest = t
		}
	}
	if len(deckModels) > 0 {
		consider(deckModels[0].CreatedAt)
	}
	if len(recordModels) > 0 {
		consider(recordModels[0].CreatedAt)
	}
	if len(matchModels) > 0 {
		consider(matchModels[0].CreatedAt)
	}
	if earliest.IsZero() {
		earliest = time.Now().Local()
	}

	return earliest, nil
}
