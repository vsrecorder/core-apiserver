package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type txCtxKey struct{}

// contextWithTx はトランザクション済みの *gorm.DB を ctx に埋め込む。
func contextWithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

// dbFromContext は ctx にトランザクションが埋め込まれていればそれを、
// なければ fallback (各リポジトリが構築時に受け取った自分自身の db) を返す。
// TransactionManager.Do を経由しない既存の呼び出し経路では fallback を使い続けるため、
// 既存の挙動と後方互換になる。
func dbFromContext(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txCtxKey{}).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return fallback
}

type TransactionManager struct {
	db *gorm.DB
}

func NewTransactionManager(db *gorm.DB) repository.TransactionManager {
	return &TransactionManager{db}
}

func (m *TransactionManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		return fn(contextWithTx(ctx, tx))
	})
}
