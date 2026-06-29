package infrastructure

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"gorm.io/gorm"
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

		decks = append(decks, entity.NewDeckUsage(r.DeckId, name, r.Count, usageRate, pokemonSprites))
	}

	return entity.NewDeckUsageStat(userId, totalRecords, decks), nil
}
