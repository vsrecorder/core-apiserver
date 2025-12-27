package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/joho/godotenv"
	"github.com/oklog/ulid/v2"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"gorm.io/gorm"
)

var (
	entropy = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func generateId(t time.Time) (string, error) {
	ms := ulid.Timestamp(t)
	id, err := ulid.New(ms, entropy)

	return id.String(), err
}

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
		// デッキコードが存在するすべてのDeckを取得(論理削除されたものも含む)
		var decks []*model.Deck
		if tx := db.Unscoped().Where("code != ''").Order("created_at DESC").Find(&decks); tx.Error != nil {
			return
		}

		db.Transaction(func(tx *gorm.DB) error {
			for _, deck := range decks {
				// DeckCodeが存在しなければ追加
				tx := db.Where("user_id = ? AND deck_id = ? AND code = ?", deck.UserId, deck.ID, deck.Code).First(&model.DeckCode{})

				if tx.Error == gorm.ErrRecordNotFound {
					fmt.Println(deck)

					id, err := generateId(deck.CreatedAt)
					if err != nil {
						return err
					}

					deckcode := model.DeckCode{
						ID:             id,
						CreatedAt:      deck.CreatedAt,
						UpdatedAt:      deck.CreatedAt,
						UserId:         deck.UserId,
						DeckId:         deck.ID,
						Code:           deck.Code,
						PrivateCodeFlg: deck.PrivateCodeFlg,
					}

					if tx := db.Save(&deckcode); tx.Error != nil {
						return tx.Error
					}
				} else if tx.Error != nil {
					return tx.Error
				} else {
					continue
				}
			}

			return nil
		}, &sql.TxOptions{Isolation: sql.LevelDefault})
	}
}

