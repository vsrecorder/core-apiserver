package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type SiteStatInterface interface {
	// Find はユーザ数・対戦数・デッキコード数の現在値を返す。
	// 論理削除済みの行は数えない。デッキコードは code が発行済みのものだけを数える。
	Find(
		ctx context.Context,
	) (*entity.SiteStat, error)
}
