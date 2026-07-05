package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type PokemonAvatarInterface interface {
	// FindRandomExcludingImageURL は image_url が一致しないアバターの中からランダムに1件返す。
	// アバター変更チャレンジで、現在のアバターとは異なる画像を提示するために使う。
	FindRandomExcludingImageURL(
		ctx context.Context,
		imageURL string,
	) (*entity.PokemonAvatar, error)
}
