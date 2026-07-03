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
}
