package infrastructure

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type UserDailyActivity struct {
	db *gorm.DB
}

func NewUserDailyActivity(
	db *gorm.DB,
) repository.UserDailyActivityInterface {
	return &UserDailyActivity{db}
}

// Touch は (user_id, date, category) を1行ずつ upsert する。
// 同日同カテゴリの2回目以降は signal_count を送られてきたぶんだけ加算する。
//
// 複数カテゴリをまとめて1文で発行するため、カテゴリが増えてもこの実装は変更不要
// (カテゴリ名を知っているのは entity のレジストリだけ。→ USER_DAILY_ACTIVITIES_PLAN.md §3.3)。
func (i *UserDailyActivity) Touch(
	ctx context.Context,
	entities []*entity.UserDailyActivity,
) error {
	if len(entities) == 0 {
		return nil
	}

	models := make([]*model.UserDailyActivity, 0, len(entities))
	for _, e := range entities {
		models = append(models, &model.UserDailyActivity{
			UserId:      e.UserId,
			Date:        e.Date,
			Category:    e.Category,
			SignalCount: 1,
			UpdatedAt:   e.UpdatedAt,
		})
	}

	// excluded は「今回INSERTしようとした値」。バルクINSERTでも行ごとの値が使われるため、
	// 固定値を渡すのではなく excluded を参照する。
	tx := dbFromContext(ctx, i.db).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "date"}, {Name: "category"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"signal_count": gorm.Expr("user_daily_activities.signal_count + excluded.signal_count"),
			"updated_at":   gorm.Expr("excluded.updated_at"),
		}),
	}).Create(&models)

	if tx.Error != nil {
		logError(ctx, tx.Error)
		return wrapError(tx.Error)
	}

	return nil
}

// DeleteByUserId は退会時に、そのユーザの日別アクティビティを行ごと削除する。
// このテーブルは論理削除を持たないため、残すと参照先の無い行になる。
func (i *UserDailyActivity) DeleteByUserId(
	ctx context.Context,
	uid string,
) error {
	if tx := dbFromContext(ctx, i.db).Where("user_id = ?", uid).Delete(&model.UserDailyActivity{}); tx.Error != nil {
		logError(ctx, tx.Error)
		return tx.Error
	}

	return nil
}
