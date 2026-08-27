package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserPlayerInterface interface {
	FindByUserId(
		ctx context.Context,
		userId string,
	) (*entity.UserPlayer, error)

	Save(
		ctx context.Context,
		entity *entity.UserPlayer,
	) error

	Delete(
		ctx context.Context,
		id string,
	) error

	// DeleteByUserId は退会時に、そのユーザのプレイヤーID紐付けをまとめて論理削除する。
	// 紐付けは1ユーザーにつき有効な行が最大1件の想定だが、
	// 何らかの理由で2件以上あっても消し残さないよう user_id でまとめて消す。
	DeleteByUserId(
		ctx context.Context,
		uid string,
	) error
}
