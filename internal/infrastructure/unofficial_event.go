package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type UnofficialEvent struct {
	db *gorm.DB
}

func NewUnofficialEvent(
	db *gorm.DB,
) repository.UnofficialEventInterface {
	return &UnofficialEvent{db}
}

func (i *UnofficialEvent) FindById(
	ctx context.Context,
	id string,
) (*entity.UnofficialEvent, error) {
	var model model.UnofficialEvent

	if tx := i.db.Where("id = ?", id).First(&model); tx.Error != nil {
		return nil, wrapError(tx.Error)
	}

	entity := entity.NewUnofficialEvent(
		model.ID,
		model.UserId,
		model.Title,
		model.Date,
	)
	entity.CreatedAt = model.CreatedAt

	return entity, nil
}

func (i *UnofficialEvent) Save(
	ctx context.Context,
	entity *entity.UnofficialEvent,
) error {
	model := model.NewUnofficialEvent(
		entity.ID,
		entity.UserId,
		entity.Title,
		entity.Date,
	)
	// 更新時に created_at を現在時刻で潰さないよう、取得済みの値をそのまま書き戻す。
	// 新規作成時はゼロ値のままGORMのautoCreateTimeに任せる。
	model.CreatedAt = entity.CreatedAt

	if tx := i.db.Save(model); tx.Error != nil {
		return tx.Error
	}

	return nil
}

// 記録から参照されなくなった自由形式イベントを削除する。
// 自由形式イベントは記録と1対1で作られるため、記録側の参照を外した後に呼ぶ想定
// (記録そのものの削除では Record.Delete が同じトランザクションの中で消している)。
func (i *UnofficialEvent) Delete(
	ctx context.Context,
	id string,
) error {
	db := dbFromContext(ctx, i.db)

	if tx := db.Where("id = ?", id).Delete(&model.UnofficialEvent{}); tx.Error != nil {
		return tx.Error
	}

	return nil
}
