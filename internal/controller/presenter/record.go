package presenter

import (
	"encoding/base64"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewRecordGetResponse(
	limit int,
	offset int,
	cursor time.Time,
	records []*entity.Record,
) *dto.RecordGetResponse {
	ret := []*dto.RecordData{}

	for _, record := range records {
		ret = append(ret, &dto.RecordData{
			Cursor: base64.StdEncoding.EncodeToString([]byte(record.CreatedAt.Format(time.RFC3339))),
			Data: &dto.RecordResponse{
				ID:              record.ID,
				CreatedAt:       record.CreatedAt,
				OfficialEventId: record.OfficialEventId,
				TonamelEventId:  record.TonamelEventId,
				FriendId:        record.FriendId,
				UserId:          record.UserId,
				DeckId:          record.DeckId,
				DeckCodeId:      record.DeckCodeId,
				PrivateFlg:      record.PrivateFlg,
				TCGMeisterURL:   record.TCGMeisterURL,
				Memo:            record.Memo,
			},
		})
	}

	return &dto.RecordGetResponse{
		Limit:   limit,
		Offset:  offset,
		Cursor:  base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339))),
		Records: ret,
	}
}

func NewRecordGetByIdResponse(
	record *entity.Record,
) *dto.RecordGetByIdResponse {
	return &dto.RecordGetByIdResponse{
		RecordResponse: dto.RecordResponse{
			ID:              record.ID,
			CreatedAt:       record.CreatedAt,
			OfficialEventId: record.OfficialEventId,
			TonamelEventId:  record.TonamelEventId,
			FriendId:        record.FriendId,
			UserId:          record.UserId,
			DeckId:          record.DeckId,
			DeckCodeId:      record.DeckCodeId,
			PrivateFlg:      record.PrivateFlg,
			TCGMeisterURL:   record.TCGMeisterURL,
			Memo:            record.Memo,
		},
	}
}

func NewRecordGetByUserIdResponse(
	limit int,
	offset int,
	cursor time.Time,
	records []*entity.Record,
) *dto.RecordGetByUserIdResponse {
	ret := []*dto.RecordData{}

	for _, record := range records {
		ret = append(ret, &dto.RecordData{
			Cursor: base64.StdEncoding.EncodeToString([]byte(record.CreatedAt.Format(time.RFC3339))),
			Data: &dto.RecordResponse{
				ID:              record.ID,
				CreatedAt:       record.CreatedAt,
				OfficialEventId: record.OfficialEventId,
				TonamelEventId:  record.TonamelEventId,
				FriendId:        record.FriendId,
				UserId:          record.UserId,
				DeckId:          record.DeckId,
				DeckCodeId:      record.DeckCodeId,
				PrivateFlg:      record.PrivateFlg,
				TCGMeisterURL:   record.TCGMeisterURL,
				Memo:            record.Memo,
			},
		})
	}

	return &dto.RecordGetByUserIdResponse{
		Limit:   limit,
		Offset:  offset,
		Cursor:  base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339))),
		Records: ret,
	}
}

func NewRecordCreateResponse(
	record *entity.Record,
) *dto.RecordCreateResponse {
	return &dto.RecordCreateResponse{
		RecordResponse: dto.RecordResponse{
			ID:              record.ID,
			CreatedAt:       record.CreatedAt,
			OfficialEventId: record.OfficialEventId,
			TonamelEventId:  record.TonamelEventId,
			FriendId:        record.FriendId,
			UserId:          record.UserId,
			DeckId:          record.DeckId,
			DeckCodeId:      record.DeckCodeId,
			PrivateFlg:      record.PrivateFlg,
			TCGMeisterURL:   record.TCGMeisterURL,
			Memo:            record.Memo,
		},
	}
}

func NewRecordUpdateResponse(
	record *entity.Record,
) *dto.RecordUpdateResponse {
	return &dto.RecordUpdateResponse{
		RecordResponse: dto.RecordResponse{
			ID:              record.ID,
			CreatedAt:       record.CreatedAt,
			OfficialEventId: record.OfficialEventId,
			TonamelEventId:  record.TonamelEventId,
			FriendId:        record.FriendId,
			UserId:          record.UserId,
			DeckId:          record.DeckId,
			DeckCodeId:      record.DeckCodeId,
			PrivateFlg:      record.PrivateFlg,
			TCGMeisterURL:   record.TCGMeisterURL,
			Memo:            record.Memo,
		},
	}
}
