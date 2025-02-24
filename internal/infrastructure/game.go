package infrastructure

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"gorm.io/gorm"
)

type Game struct {
	db *gorm.DB
}

func NewGame(
	db *gorm.DB,
) repository.GameInterface {
	return &Game{db}
}

func (i *Game) FindById(
	ctx context.Context,
	id string,
) (*entity.Game, error) {
	var model *model.Game

	if tx := i.db.Where("id = ?", id).First(&model); tx.Error != nil {
		return nil, tx.Error
	}

	entity := entity.NewGame(
		model.ID,
		model.CreatedAt,
		model.MatchId,
		model.UserId,
		model.GoFirst,
		model.WinningFlg,
		model.YourPrizeCards,
		model.OpponentsPrizeCards,
		model.Memo,
	)

	return entity, nil
}

func (i *Game) FindByMatchId(
	ctx context.Context,
	matchId string,
) ([]*entity.Game, error) {
	var models []*model.Game

	if tx := i.db.Where("match_id = ?", matchId).Order("created_at ASC").Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	var entities []*entity.Game
	for _, model := range models {
		entity := entity.NewGame(
			model.ID,
			model.CreatedAt,
			model.MatchId,
			model.UserId,
			model.GoFirst,
			model.WinningFlg,
			model.YourPrizeCards,
			model.OpponentsPrizeCards,
			model.Memo,
		)
		entities = append(entities, entity)
	}

	return entities, nil
}
