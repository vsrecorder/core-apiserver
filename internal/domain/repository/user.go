package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.User, error)

	// IsWithdrawn は退会済み(論理削除済み)のユーザーが残っているかを返す。
	// FindById は有効なユーザーしか見ないため、
	// 「未登録」と「退会済み」を区別したい場合はこちらを使う。
	IsWithdrawn(
		ctx context.Context,
		id string,
	) (bool, error)

	Save(
		ctx context.Context,
		entity *entity.User,
	) error

	Delete(
		ctx context.Context,
		id string,
	) error
}
