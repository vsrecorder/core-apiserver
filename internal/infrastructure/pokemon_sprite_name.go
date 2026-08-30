package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type PokemonSpriteName struct {
	db *gorm.DB
}

func NewPokemonSpriteName(
	db *gorm.DB,
) repository.PokemonSpriteNameInterface {
	return &PokemonSpriteName{db}
}

func (i *PokemonSpriteName) FindNamesByIds(
	ctx context.Context,
	ids []string,
) (map[string]string, error) {
	names := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}

	var models []*model.PokemonSprite
	if tx := dbFromContext(ctx, i.db).Where("id IN ?", ids).Find(&models); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, wrapError(tx.Error)
	}

	for _, m := range models {
		names[m.ID] = m.Name
	}

	return names, nil
}
