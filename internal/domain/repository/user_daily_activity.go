package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserDailyActivityInterface interface {
	// Touch は (user_id, date, category) を1行ずつ upsert する。
	// 同日同カテゴリの2回目以降は signal_count を加算する。
	// カテゴリ数に依存しないよう複数件をまとめて受け取る(カテゴリが増えても実装は不変)。
	Touch(
		ctx context.Context,
		entities []*entity.UserDailyActivity,
	) error

	// DeleteByUserId は退会時に、そのユーザの日別アクティビティをまとめて削除する。
	// このテーブルは論理削除を持たないため行ごと物理削除する。
	DeleteByUserId(
		ctx context.Context,
		uid string,
	) error
}
