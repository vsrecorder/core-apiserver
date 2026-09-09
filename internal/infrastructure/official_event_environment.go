package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type OfficialEventEnvironment struct {
	db *gorm.DB
}

func NewOfficialEventEnvironment(
	db *gorm.DB,
) repository.OfficialEventEnvironmentInterface {
	return &OfficialEventEnvironment{db}
}

func (i *OfficialEventEnvironment) FindEnvironmentIdByOfficialEventId(
	ctx context.Context,
	officialEventId uint,
) (string, error) {
	var m model.OfficialEventEnvironment

	if tx := dbFromContext(ctx, i.db).Where(
		"official_event_id = ?", officialEventId,
	).First(&m); tx.Error != nil {
		logError(ctx, tx.Error)
		return "", wrapError(tx.Error)
	}

	return m.EnvironmentId, nil
}

func (i *OfficialEventEnvironment) FindAll(
	ctx context.Context,
) (map[uint]string, error) {
	var models []*model.OfficialEventEnvironment

	if tx := dbFromContext(ctx, i.db).Order(
		"official_event_id ASC",
	).Find(&models); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	overrides := make(map[uint]string, len(models))
	for _, m := range models {
		overrides[m.OfficialEventId] = m.EnvironmentId
	}

	return overrides, nil
}
