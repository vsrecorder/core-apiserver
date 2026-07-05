package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type PokemonAvatar struct {
	db *gorm.DB
}

func NewPokemonAvatar(
	db *gorm.DB,
) repository.PokemonAvatarInterface {
	return &PokemonAvatar{db}
}

func (i *PokemonAvatar) FindRandomExcludingImageURL(
	ctx context.Context,
	imageURL string,
) (*entity.PokemonAvatar, error) {
	var avatar *model.PokemonAvatar

	tx := dbFromContext(ctx, i.db).
		Where("image_url <> ?", imageURL).
		Order("RANDOM()").
		First(&avatar)
	if tx.Error != nil {
		return nil, wrapError(tx.Error)
	}

	return entity.NewPokemonAvatar(
		avatar.ID,
		avatar.Title,
		avatar.ImageURL,
		avatar.Detail,
	), nil
}
