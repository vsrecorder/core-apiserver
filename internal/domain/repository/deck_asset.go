package repository

import (
	"context"
)

// DeckAssetInterface はデッキコードに紐づく外部リソース(公式サイトのデッキ結果HTML、デッキ画像)を
// 取得してオブジェクトストレージへ配置する操作を提供する。
type DeckAssetInterface interface {
	UploadDeckResultHTML(
		ctx context.Context,
		deckCode string,
	) error

	UploadDeckImage(
		ctx context.Context,
		deckCode string,
	) error
}
