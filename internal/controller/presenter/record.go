package presenter

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// encodeCursor は event_date と created_at をコンポジットカーソル JSON の base64 に変換する。
// フォーマット: base64({"event_date":"RFC3339","created_at":"RFC3339"})
func encodeCursor(eventDate, createdAt time.Time) string {
	b, _ := json.Marshal(struct {
		EventDate string `json:"event_date"`
		CreatedAt string `json:"created_at"`
	}{
		EventDate: eventDate.Format(time.RFC3339),
		CreatedAt: createdAt.Format(time.RFC3339),
	})
	return base64.StdEncoding.EncodeToString(b)
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
				IgnoreStatsFlg:    record.IgnoreStatsFlg,
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
			IgnoreStatsFlg:    record.IgnoreStatsFlg,
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
				IgnoreStatsFlg:    record.IgnoreStatsFlg,
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
			IgnoreStatsFlg:    record.IgnoreStatsFlg,
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
			IgnoreStatsFlg:    record.IgnoreStatsFlg,
			TCGMeisterURL:     record.TCGMeisterURL,
			Memo:              record.Memo,
		},
	}
}
