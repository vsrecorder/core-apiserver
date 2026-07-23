package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type TonamelEventStore struct {
	db *gorm.DB
}

func NewTonamelEventStore(
	db *gorm.DB,
) repository.TonamelEventStoreInterface {
	return &TonamelEventStore{db}
}

func (i *TonamelEventStore) FindByIds(
	ctx context.Context,
	ids []string,
) ([]*entity.TonamelEvent, error) {
	if len(ids) == 0 {
		return []*entity.TonamelEvent{}, nil
	}

	var models []*model.TonamelEvent
	if tx := dbFromContext(ctx, i.db).
		Where("id IN ?", ids).
		Find(&models); tx.Error != nil {
		return nil, wrapError(tx.Error)
	}

	ret := make([]*entity.TonamelEvent, 0, len(models))
	for _, m := range models {
		ret = append(ret, entity.NewTonamelEvent(m.ID, m.Title, m.Description, m.Image))
	}

	return ret, nil
}

func (i *TonamelEventStore) Save(
	ctx context.Context,
	tonamelEvent *entity.TonamelEvent,
) error {
	now := time.Now().Local()

	m := model.NewTonamelEvent(
		tonamelEvent.ID,
		tonamelEvent.Title,
		tonamelEvent.Description,
		tonamelEvent.Image,
		now,
		now,
	)

	// 同じ大会IDが既にあれば上書きする(再取得時に最新の内容へ更新できるようにする)。
	// created_at は初回のまま保つため更新対象に含めない。
	if tx := dbFromContext(ctx, i.db).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns(
			[]string{"title", "description", "image", "updated_at"},
		),
	}).Create(m); tx.Error != nil {
		return wrapError(tx.Error)
	}

	return nil
}
