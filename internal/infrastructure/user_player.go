package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type UserPlayer struct {
	db *gorm.DB
}

func NewUserPlayer(
	db *gorm.DB,
) repository.UserPlayerInterface {
	return &UserPlayer{db}
}

func (i *UserPlayer) FindByUserId(
	ctx context.Context,
	userId string,
) (*entity.UserPlayer, error) {
	var userPlayer *model.UserPlayer

	if tx := dbFromContext(ctx, i.db).Where("user_id = ?", userId).First(&userPlayer); tx.Error != nil {
		return nil, wrapError(tx.Error)
	}

	entity := entity.NewUserPlayer(
		userPlayer.ID,
		userPlayer.CreatedAt,
		userPlayer.UserId,
		userPlayer.PlayerId,
	)

	return entity, nil
}

func (i *UserPlayer) ExistsActiveByPlayerId(
	ctx context.Context,
	playerId string,
) (bool, error) {
	var count int64

	if tx := dbFromContext(ctx, i.db).Model(&model.UserPlayer{}).Where("player_id = ?", playerId).Count(&count); tx.Error != nil {
		return false, tx.Error
	}

	return count > 0, nil
}

func (i *UserPlayer) Save(
	ctx context.Context,
	entity *entity.UserPlayer,
) error {
	userPlayer := model.NewUserPlayer(
		entity.ID,
		entity.CreatedAt,
		entity.UserId,
		entity.PlayerId,
	)

	return dbFromContext(ctx, i.db).Save(userPlayer).Error
}

func (i *UserPlayer) Delete(
	ctx context.Context,
	id string,
) error {
	if tx := dbFromContext(ctx, i.db).Where("id = ?", id).Delete(&model.UserPlayer{}); tx.Error != nil {
		return tx.Error
	}

	return nil
}
