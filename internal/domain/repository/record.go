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

	FindByDeckId(
		ctx context.Context,
		deckId string,
		limit int,
		offset int,
		eventType string,
	) ([]*entity.Record, error)

	FindByDeckIdOnCursor(
		ctx context.Context,
		deckId string,
		limit int,
		cursorEventDate time.Time,
		cursorCreatedAt time.Time,
		eventType string,
	) ([]*entity.Record, error)

	FindByDeckCodeId(
		ctx context.Context,
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
