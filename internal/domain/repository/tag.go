package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type TagInterface interface {
	// FindByUserId は uid 自身のタグを新しい順で返す(オートコンプリート候補・一覧用)。
	// プリセットタグ(preset_flg=true)は含めない。
	FindByUserId(
		ctx context.Context,
		uid string,
	) ([]*entity.Tag, error)

	// FindPresets は全ユーザー共通のプリセットタグ(preset_flg=true)を、
	// 作成順(=ACE SPEC の card id 昇順、≒収録順)で返す。
	FindPresets(
		ctx context.Context,
	) ([]*entity.Tag, error)

	// FindById は1件取得する。所有者チェック(authorization)などで使う。
	// 存在しなければ apperror.ErrRecordNotFound。
	FindById(
		ctx context.Context,
		id string,
	) (*entity.Tag, error)

	// FindAttachableByIds は ids のうち uid が付与できる有効なタグだけを返す。
	// 付与できるのは「uid 自身のタグ」または「プリセットタグ」。
	// デッキ/デッキコードへタグを付与する際、他人のタグIDや存在しないIDを弾くために使う。
	FindAttachableByIds(
		ctx context.Context,
		ids []string,
		uid string,
	) ([]*entity.Tag, error)

	// FindByUserIdAndName は同名タグの有無を確認する(重複作成の抑止)。
	// 無ければ apperror.ErrRecordNotFound。
	FindByUserIdAndName(
		ctx context.Context,
		uid string,
		name string,
	) (*entity.Tag, error)

	Save(
		ctx context.Context,
		entity *entity.Tag,
	) error

	// Delete はタグを論理削除し、併せてそのタグの中間テーブル(deck_tags 等)の
	// 行を物理削除する。付与先エンティティが削除済みタグを参照し続けないようにするため、
	// 1トランザクションでまとめて行う。
	Delete(
		ctx context.Context,
		id string,
	) error

	// ReplaceDeckTags は deckId の付与タグを tagIds の集合に一致させる(差分INSERT/DELETE)。
	ReplaceDeckTags(
		ctx context.Context,
		deckId string,
		tagIds []string,
	) error

	// ReplaceDeckCodeTags は deckCodeId の付与タグを tagIds の集合に一致させる。
	ReplaceDeckCodeTags(
		ctx context.Context,
		deckCodeId string,
		tagIds []string,
	) error

	// ReplaceMatchTags は matchId(対戦結果)の付与タグを tagIds の集合に一致させる。
	ReplaceMatchTags(
		ctx context.Context,
		matchId string,
		tagIds []string,
	) error
}
