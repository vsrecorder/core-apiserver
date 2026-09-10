// sync-pokemon-avatars は公式サイト(プレイヤーズクラブ)のアバター一覧API
// (https://players.pokemon-card.com/avatar_search) からavatarListを取得し、
// pokemon_avatars テーブルへ保存するバッチ。
//
// 新規アバターの追加やタイトル・画像URLの変更を追随できるよう、既存行は
// avatar_id(=pokemon_avatars.id)を基準に上書き(upsert)する。定期実行を想定しており、
// backfill-badges 等の一回限りの初期投入バッチとは異なる。
//
// 使い方:
//
//	go run ./cmd/sync-pokemon-avatars
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm/clause"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
	"github.com/vsrecorder/core-apiserver/internal/logging"
)

const appName = "sync-pokemon-avatars"

const (
	ExitCodeOK = iota
	ExitCodeNG
)

const avatarSearchURL = "https://players.pokemon-card.com/avatar_search"

type avatarSearchResponse struct {
	Code       int           `json:"code"`
	AvatarList []avatarEntry `json:"avatarList"`
}

type avatarEntry struct {
	AvatarId     int    `json:"avatar_id"`
	Title        string `json:"title"`
	AvatarImage  string `json:"avatarImage"`
	AvatarDetail string `json:"avatarDetail"`
}

func main() {
	// ログは cmd/core-apiserver と同じJSON形式に揃える。これを呼ばないと slog の
	// 既定ハンドラ(テキスト)のままになり、usecase 層のログから layer やソース位置が落ちる。
	slog.SetDefault(logging.InitLogger(logging.Config{
		Level:   "info",
		AppName: appName,
	}))

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

	avatars, err := fetchAvatarList()
	if err != nil {
		slog.Error("failed to fetch avatar list", logging.Err(err))
		os.Exit(ExitCodeNG)
	}

	slog.Info("fetched avatars", slog.Int("count", len(avatars)))

	now := time.Now().Local()
	models := make([]*model.PokemonAvatar, 0, len(avatars))
	for _, a := range avatars {
		models = append(models, model.NewPokemonAvatar(
			a.AvatarId,
			a.Title,
			a.AvatarImage,
			a.AvatarDetail,
			now,
			now,
		))
	}

	const batchSize = 200
	tx := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns(
			[]string{"title", "image_url", "detail", "updated_at"},
		),
	}).CreateInBatches(models, batchSize)
	if tx.Error != nil {
		slog.Error("failed to save avatars", logging.Err(tx.Error))
		os.Exit(ExitCodeNG)
	}

	slog.Info("saved avatars", slog.Int64("count", tx.RowsAffected))
	os.Exit(ExitCodeOK)
}

// fetchAvatarList は avatar_search API から全アバターを取得する。
func fetchAvatarList() ([]avatarEntry, error) {
	resp, err := http.Get(avatarSearchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res avatarSearchResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}

	if res.Code != http.StatusOK {
		return nil, fmt.Errorf("unexpected response code: %d", res.Code)
	}

	return res.AvatarList, nil
}
