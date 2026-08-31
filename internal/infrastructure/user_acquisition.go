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

type UserAcquisition struct {
	db *gorm.DB
}

func NewUserAcquisition(
	db *gorm.DB,
) repository.UserAcquisitionInterface {
	return &UserAcquisition{db}
}

// 判明しなかった項目は NULL で持つ(model.UserAcquisition のコメント参照)。
func nullableString(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func nullableTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}

	return &value
}

// Create は流入元を1件保存する。user_id の衝突は無視して先に入っている行を残す。
// 流入元は「最初に連れてきた投稿」を知りたい値なので、後から来たものでは上書きしない。
func (i *UserAcquisition) Create(
	ctx context.Context,
	entity *entity.UserAcquisition,
) error {
	m := &model.UserAcquisition{
		UserId:            entity.UserId,
		Source:            nullableString(entity.Source),
		Medium:            nullableString(entity.Medium),
		Campaign:          nullableString(entity.Campaign),
		Content:           nullableString(entity.Content),
		Referrer:          nullableString(entity.Referrer),
		LandingPath:       nullableString(entity.LandingPath),
		LandingAt:         nullableTime(entity.LandingAt),
		SourceInferredFlg: entity.SourceInferred,
		SurveyAnswer:      nullableString(entity.SurveyAnswer),
		CreatedAt:         entity.CreatedAt,
		UpdatedAt:         entity.UpdatedAt,
	}

	tx := dbFromContext(ctx, i.db).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(m)
	if tx.Error != nil {
		logError(ctx, tx.Error)
		return wrapError(tx.Error)
	}

	return nil
}

// DeleteByUserId は退会時に、そのユーザの流入元を行ごと削除する。
// このテーブルは論理削除を持たないため、残すと参照先の無い行になる。
func (i *UserAcquisition) DeleteByUserId(
	ctx context.Context,
	uid string,
) error {
	if tx := dbFromContext(ctx, i.db).Where("user_id = ?", uid).Delete(&model.UserAcquisition{}); tx.Error != nil {
		logError(ctx, tx.Error)
		return tx.Error
	}

	return nil
}
