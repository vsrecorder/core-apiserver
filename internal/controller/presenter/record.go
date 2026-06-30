package presenter

import (
	"encoding/base64"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// encodeCursor は event_date と created_at をコンポジットカーソル文字列に変換する。
// フォーマット: base64("eventDate_RFC3339|createdAt_RFC3339")
func encodeCursor(eventDate, createdAt time.Time) string {
	return base64.StdEncoding.EncodeToString(
		[]byte(eventDate.Format(time.RFC3339) + "|" + createdAt.Format(time.RFC3339)),
	)
}

func NewRecordGetResponse(
	limit int,
	offset int,
	cursorEventDate time.Time,
	cursorCreatedAt time.Time,
	records []*entity.Record,
) *dto.RecordGetResponse {
	ret := []*dto.RecordData{}

	for _, record := range records {
		ret = append(ret, &dto.RecordData{
			Cursor: encodeCursor(record.EventDate, record.CreatedAt),
			Data: &dto.RecordResponse{
				ID:                record.ID,
				CreatedAt:         record.CreatedAt,
				OfficialEventId:   record.OfficialEventId,
				TonamelEventId:    record.TonamelEventId,
				FriendId:          record.FriendId,
				UnofficialEventId: record.UnofficialEventId,
				UserId:            record.UserId,
				DeckId:            record.DeckId,
				DeckCodeId:        record.DeckCodeId,
				EventDate:         record.EventDate,
				PrivateFlg:        record.PrivateFlg,
				TCGMeisterURL:     record.TCGMeisterURL,
				Memo:              record.Memo,
			},
		})
	}

	return &dto.RecordGetResponse{
		Limit:   limit,
		Offset:  offset,
		Cursor:  encodeCursor(cursorEventDate, cursorCreatedAt),
		Records: ret,
	}
}

func NewRecordGetByIdResponse(
	record *entity.Record,
) *dto.RecordGetByIdResponse {
	return &dto.RecordGetByIdResponse{
		RecordResponse: dto.RecordResponse{
			ID:                record.ID,
			CreatedAt:         record.CreatedAt,
			OfficialEventId:   record.OfficialEventId,
			TonamelEventId:    record.TonamelEventId,
			FriendId:          record.FriendId,
			UnofficialEventId: record.UnofficialEventId,
			UserId:            record.UserId,
			DeckId:            record.DeckId,
			DeckCodeId:        record.DeckCodeId,
			EventDate:         record.EventDate,
			PrivateFlg:        record.PrivateFlg,
			TCGMeisterURL:     record.TCGMeisterURL,
			Memo:              record.Memo,
		},
	}
}

func NewRecordGetByUserIdResponse(
	limit int,
	offset int,
	cursorEventDate time.Time,
	cursorCreatedAt time.Time,
	records []*entity.Record,
) *dto.RecordGetByUserIdResponse {
	ret := []*dto.RecordData{}

	for _, record := range records {
		ret = append(ret, &dto.RecordData{
			Cursor: encodeCursor(record.EventDate, record.CreatedAt),
			Data: &dto.RecordResponse{
				ID:                record.ID,
				CreatedAt:         record.CreatedAt,
				OfficialEventId:   record.OfficialEventId,
				TonamelEventId:    record.TonamelEventId,
				FriendId:          record.FriendId,
				UnofficialEventId: record.UnofficialEventId,
				UserId:            record.UserId,
				DeckId:            record.DeckId,
				DeckCodeId:        record.DeckCodeId,
				EventDate:         record.EventDate,
				PrivateFlg:        record.PrivateFlg,
				TCGMeisterURL:     record.TCGMeisterURL,
				Memo:              record.Memo,
			},
		})
	}

	return &dto.RecordGetByUserIdResponse{
		Limit:   limit,
		Offset:  offset,
		Cursor:  encodeCursor(cursorEventDate, cursorCreatedAt),
		Records: ret,
	}
}

func NewRecordCreateResponse(
	record *entity.Record,
) *dto.RecordCreateResponse {
	return &dto.RecordCreateResponse{
		RecordResponse: dto.RecordResponse{
			ID:                record.ID,
			CreatedAt:         record.CreatedAt,
			OfficialEventId:   record.OfficialEventId,
			TonamelEventId:    record.TonamelEventId,
			FriendId:          record.FriendId,
			UnofficialEventId: record.UnofficialEventId,
			UserId:            record.UserId,
			DeckId:            record.DeckId,
			DeckCodeId:        record.DeckCodeId,
			EventDate:         record.EventDate,
			PrivateFlg:        record.PrivateFlg,
			TCGMeisterURL:     record.TCGMeisterURL,
			Memo:              record.Memo,
		},
	}
}

func NewRecordUpdateResponse(
	record *entity.Record,
) *dto.RecordUpdateResponse {
	return &dto.RecordUpdateResponse{
		RecordResponse: dto.RecordResponse{
			ID:                record.ID,
			CreatedAt:         record.CreatedAt,
			OfficialEventId:   record.OfficialEventId,
			TonamelEventId:    record.TonamelEventId,
			FriendId:          record.FriendId,
			UnofficialEventId: record.UnofficialEventId,
			UserId:            record.UserId,
			DeckId:            record.DeckId,
			DeckCodeId:        record.DeckCodeId,
			EventDate:         record.EventDate,
			PrivateFlg:        record.PrivateFlg,
			TCGMeisterURL:     record.TCGMeisterURL,
			Memo:              record.Memo,
		},
	}
}
