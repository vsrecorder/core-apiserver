package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/joho/godotenv"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"gorm.io/gorm"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	if _, err := config.LoadDefaultConfig(context.Background()); err != nil {
		log.Printf("failed to load default config: %v", err)
		return
	}

	dbHostname := os.Getenv("DB_HOSTNAME")
	dbPort := os.Getenv("DB_PORT")
	userName := os.Getenv("DB_USER_NAME")
	userPassword := os.Getenv("DB_USER_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	db, err := postgres.NewDB(dbHostname, dbPort, userName, userPassword, dbName)
	if err != nil {
		log.Fatalf("failed to connect database: %v\n", err)
	}

	{
		// deck_idが存在するすべてのRecordを取得(論理削除されたものも含む)
		var records []*model.Record
		if tx := db.Unscoped().Where("deck_id != ''").Order("created_at DESC").Find(&records); tx.Error != nil {
			return
		}

		db.Transaction(func(tx *gorm.DB) error {
			// 各レコードに対して最新のdeck_code_idを取得し、Recordに反映する
			for _, record := range records {
				// Recordに紐付けられたDeckとDeckと紐付けられた最新のDeckCodeを結合して取得する
				var deckJoinDeckCodes *model.DeckJoinDeckCode

				tx := db.Table(
					"decks",
				).Select(`
					decks.id AS deck_id,
					decks.created_at AS deck_created_at,
					decks.updated_at AS deck_updated_at,
					decks.deleted_at AS deck_deleted_at,
					decks.archived_at AS deck_archived_at,
					decks.user_id AS deck_user_id,
					decks.name AS deck_name,
					decks.code AS deck_code,
					decks.private_code_flg AS deck_private_code_flg,
					decks.private_flg AS deck_private_flg,
					deck_codes.id AS deck_code_id,
					deck_codes.created_at AS deck_code_created_at,
					deck_codes.updated_at AS deck_code_updated_at,
					deck_codes.deleted_at AS deck_code_deleted_at,
					deck_codes.user_id AS deck_code_user_id,
					deck_codes.deck_id AS deck_code_deck_id,
					deck_codes.code AS deck_code_code,
					deck_codes.private_code_flg AS deck_code_private_code_flg,
					deck_codes.memo AS deck_code_memo
				`,
				).Joins(`
					LEFT JOIN (
						SELECT DISTINCT ON (deck_id)
							id,
							created_at,
							updated_at,
							deleted_at,
							user_id,
							deck_id,
							code,
							private_code_flg,
							memo
						FROM deck_codes
						WHERE deck_id = ?
						ORDER BY deck_id, created_at DESC
					) AS deck_codes ON decks.id = deck_codes.deck_id
				`, record.DeckId,
				).Where(
					"decks.id = ? AND decks.deleted_at IS NULL", record.DeckId,
				).Scan(&deckJoinDeckCodes)

				if tx.Error != nil {
					return tx.Error
				}

				deck := entity.NewDeck(
					deckJoinDeckCodes.DeckID,
					deckJoinDeckCodes.DeckCreatedAt,
					deckJoinDeckCodes.DeckArchivedAt.Time,
					deckJoinDeckCodes.DeckUserId,
					deckJoinDeckCodes.DeckName,
					deckJoinDeckCodes.DeckCode,
					deckJoinDeckCodes.DeckPrivateCodeFlg,
					deckJoinDeckCodes.DeckPrivateFlg,
					entity.NewDeckCode(
						deckJoinDeckCodes.DeckCodeID,
						deckJoinDeckCodes.DeckCodeCreatedAt,
						deckJoinDeckCodes.DeckCodeUserId,
						deckJoinDeckCodes.DeckCodeDeckId,
						deckJoinDeckCodes.DeckCodeCode,
						deckJoinDeckCodes.DeckCodePrivateFlg,
						deckJoinDeckCodes.DeckCodeMemo,
					),
				)

				newRecord := model.NewRecord(
					record.ID,
					record.CreatedAt,
					record.OfficialEventId,
					record.TonamelEventId,
					record.FriendId,
					record.UserId,
					record.DeckId,
					deck.LatestDeckCode.ID,
					record.PrivateFlg,
					record.TCGMeisterURL,
					record.Memo,
				)

				if tx := db.Save(newRecord); tx.Error != nil {
					return tx.Error
				}
			}
			return nil
		}, &sql.TxOptions{Isolation: sql.LevelDefault})
	}
}
