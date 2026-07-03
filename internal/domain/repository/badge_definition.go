package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type BadgeDefinitionInterface interface {
	FindAll(
		ctx context.Context,
	) ([]*entity.BadgeDefinition, error)
}
