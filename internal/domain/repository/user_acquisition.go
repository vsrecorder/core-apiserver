package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserAcquisitionInterface interface {
	// Create は流入元を1件保存する。既に行がある場合は何もしない(初回タッチ優先)。
	// 登録直後の1回しか呼ばれない想定だが、webapp のリトライや同時ログインで
	// 二重に届いても後勝ちで上書きしないよう、衝突は無視する。
	Create(
		ctx context.Context,
		entity *entity.UserAcquisition,
	) error

	// DeleteByUserId は退会時に、そのユーザの流入元を削除する。
	// このテーブルは論理削除を持たないため行ごと物理削除する。
	DeleteByUserId(
		ctx context.Context,
		uid string,
	) error
}
