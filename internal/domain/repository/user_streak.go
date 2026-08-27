package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserStreakInterface interface {
	// FindByUserId は該当ユーザーのストリーク状態が存在しない場合、
	// apperror.ErrRecordNotFound を返す。
	FindByUserId(
		ctx context.Context,
		userId string,
	) (*entity.UserStreak, error)

	Save(
		ctx context.Context,
		entity *entity.UserStreak,
	) error

	// DeleteByUserId は退会時に、そのユーザのストリーク状態をまとめて削除する。
	// このテーブルは論理削除を持たないため行ごと物理削除する。
	DeleteByUserId(
		ctx context.Context,
		uid string,
	) error
}
