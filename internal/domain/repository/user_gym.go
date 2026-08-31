package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserGymInterface interface {
	// FindByUserId は uid のMyジムを、登録した順(古い順)に店舗情報つきで返す。
	//
	// 一覧は必ず店舗名と所在地を伴って表示するため、shops を結合した View を返す。
	// 上限に達しているかの判定にも使うので、並び順と件数を保証する。
	FindByUserId(
		ctx context.Context,
		uid string,
	) ([]*entity.UserGymView, error)

	// LockByUserId は uid のMyジムを更新する間、同じユーザによる同時更新を待たせる。
	//
	// 登録は「今の件数を数えてから1件足す」ため、その間に別のリクエストが割り込むと
	// 双方が上限未満と判断して上限を超えて登録できてしまう(並列リクエストで再現する)。
	// 行ロックでは1件も登録が無いときにロックする対象が存在しないため、
	// ユーザ単位のロックで直列化する。トランザクション内でのみ意味を持ち、
	// コミット/ロールバックで自動的に解放される。
	LockByUserId(
		ctx context.Context,
		uid string,
	) error

	// Create はMyジムを1件登録する。
	// 同じ店舗の重複登録は主キー違反になるため、呼び出し側で登録済みかを確認する。
	Create(
		ctx context.Context,
		entity *entity.UserGym,
	) error

	// Delete は uid が登録したMyジムのうち shopId のものを解除する。
	Delete(
		ctx context.Context,
		uid string,
		shopId uint,
	) error

	// DeleteByUserId は退会時に、そのユーザのMyジムをまとめて削除する。
	// このテーブルは論理削除を持たないため行ごと物理削除する。
	DeleteByUserId(
		ctx context.Context,
		uid string,
	) error
}
