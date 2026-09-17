package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type RecordInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.Record, error)

	Find(
		ctx context.Context,
		limit int,
		offset int,
		eventType string,
	) ([]*entity.Record, error)

	FindOnCursor(
		ctx context.Context,
		limit int,
		cursorEventDate time.Time,
		cursorCreatedAt time.Time,
		eventType string,
	) ([]*entity.Record, error)

	FindByUserId(
		ctx context.Context,
		uid string,
		limit int,
		offset int,
		eventType string,
	) ([]*entity.Record, error)

	FindByUserIdOnCursor(
		ctx context.Context,
		uid string,
		limit int,
		cursorEventDate time.Time,
		cursorCreatedAt time.Time,
		eventType string,
	) ([]*entity.Record, error)

	FindByOfficialEventId(
		ctx context.Context,
		officialEventId uint,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	FindByTonamelEventId(
		ctx context.Context,
		tonamelEventId string,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	// FindByDeckId / FindByDeckIdOnCursor / FindByDeckCodeId は、いずれも uid の記録に限って返す。
	// デッキIDは公開デッキ一覧やみんなの公開デッキから誰でも取得できるため、
	// deck_id だけで絞ると他人の非公開記録(メモ等)まで読めてしまう。
	// 記録のデッキ参照は本人のデッキに限られる(usecase が保存時に検証する)ので、
	// 「そのデッキの記録」と「本人のそのデッキの記録」は同じ集合になる。
	FindByDeckId(
		ctx context.Context,
		uid string,
		deckId string,
		limit int,
		offset int,
		eventType string,
	) ([]*entity.Record, error)

	FindByDeckIdOnCursor(
		ctx context.Context,
		uid string,
		deckId string,
		limit int,
		cursorEventDate time.Time,
		cursorCreatedAt time.Time,
		eventType string,
	) ([]*entity.Record, error)

	FindByDeckCodeId(
		ctx context.Context,
		uid string,
		deckCodeId string,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	// DeleteByUserId は退会時に、そのユーザの記録と、記録に紐づく対戦結果・対局・
	// 自由形式イベントをまとめて論理削除する。
	// 記録を1件ずつ Delete すると記録数(と対戦数)に比例してクエリが増え、
	// 1トランザクションの保持時間がそのまま延びるため、退会処理ではこちらを使う。
	DeleteByUserId(
		ctx context.Context,
		uid string,
	) error

	Save(
		ctx context.Context,
		entity *entity.Record,
	) error

	Delete(
		ctx context.Context,
		id string,
	) error
}
