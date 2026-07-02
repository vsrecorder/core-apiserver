package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type DeckUsageStat struct {
	db *gorm.DB
}

func NewDeckUsageStat(
	db *gorm.DB,
) repository.DeckUsageStatInterface {
	return &DeckUsageStat{db}
}

type deckUsageResult struct {
	DeckId string
	Name   string
	Count  int
}

type deckWinResult struct {
	DeckId  string
	Matches int
	Wins    int
}

func (i *DeckUsageStat) FindDeckUsageStat(
	ctx context.Context,
	userId string,
	fromDate time.Time,
	toDate time.Time,
) (*entity.DeckUsageStat, error) {
	var results []deckUsageResult

	// 対戦記録(records)をデッキごとに集計する。
	// デッキが削除済みでも名称を表示できるよう deleted_at は条件に含めない。
	query := i.db.Table("records").
		Select("records.deck_id AS deck_id, COALESCE(decks.name, '') AS name, COUNT(*) AS count").
		Joins("LEFT JOIN decks ON records.deck_id = decks.id").
		Where("records.user_id = ? AND records.deleted_at IS NULL AND records.deck_id != ''", userId)

	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("records.event_date < ?", toDate)
	}

	query = query.Group("records.deck_id, decks.name").Order("count DESC")

	if tx := query.Scan(&results); tx.Error != nil {
		return nil, tx.Error
	}

	totalRecords := 0
	for _, r := range results {
		totalRecords += r.Count
	}

	// デッキごとの対戦勝敗を records.deck_id 基準で集計する。
	// matches.deck_id は記録後にデッキを変更しても更新されないため、records.deck_id を正とする
	// （opponent_deck_usage_stat.go と同様の方針）。
	var winResults []deckWinResult
	winQuery := i.db.Table("matches").
		Select("records.deck_id AS deck_id, COUNT(*) AS matches, SUM(CASE WHEN matches.victory_flg THEN 1 ELSE 0 END) AS wins").
		Joins("JOIN records ON matches.record_id = records.id").
		Where("records.user_id = ? AND records.deleted_at IS NULL AND matches.deleted_at IS NULL AND records.deck_id != ''", userId)

	if !fromDate.IsZero() {
		winQuery = winQuery.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		winQuery = winQuery.Where("records.event_date < ?", toDate)
	}

	winQuery = winQuery.Group("records.deck_id")

	if tx := winQuery.Scan(&winResults); tx.Error != nil {
		return nil, tx.Error
	}

	winsByDeck := make(map[string]deckWinResult, len(winResults))
	for _, w := range winResults {
		winsByDeck[w.DeckId] = w
	}

	decks := []*entity.DeckUsage{}
	for _, r := range results {
		// デッキに設定されたポケモンスプライトを取得する
		var deckPokemonSpriteModels []*model.DeckPokemonSprite
		if tx := i.db.Where("deck_id = ?", r.DeckId).Order("position ASC").Find(&deckPokemonSpriteModels); tx.Error != nil {
			return nil, tx.Error
		}

		var pokemonSprites []*entity.PokemonSprite
		for _, m := range deckPokemonSpriteModels {
			pokemonSprites = append(pokemonSprites, entity.NewPokemonSprite(m.PokemonSpriteId))
		}

		// 削除済みなどで名称を取得できなかった場合のフォールバック
		name := r.Name
		if name == "" {
			name = "不明なデッキ"
		}

		var usageRate float64
		if totalRecords > 0 {
			usageRate = float64(r.Count) / float64(totalRecords)
		}

		wins, losses, winRate := 0, 0, 0.0
		if w, ok := winsByDeck[r.DeckId]; ok {
			wins = w.Wins
			losses = w.Matches - w.Wins
			if w.Matches > 0 {
				winRate = float64(w.Wins) / float64(w.Matches)
			}
		}

		decks = append(decks, entity.NewDeckUsage(r.DeckId, name, r.Count, usageRate, wins, losses, winRate, pokemonSprites))
	}

	return entity.NewDeckUsageStat(userId, totalRecords, decks), nil
}
