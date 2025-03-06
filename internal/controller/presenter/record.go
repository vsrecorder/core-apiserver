package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewRecordGetResponse(
	limit int,
	offset int,
	records []*entity.Record,
) *dto.RecordGetResponse {
	ret := []*dto.RecordResponse{}

	for _, record := range records {
		ret = append(ret, &dto.RecordResponse{
			ID:              record.ID,
			CreatedAt:       record.CreatedAt,
			OfficialEventId: record.OfficialEventId,
			TonamelEventId:  record.TonamelEventId,
			FriendId:        record.FriendId,
			UserId:          record.UserId,
			DeckId:          record.DeckId,
			PrivateFlg:      record.PrivateFlg,
			TCGMeisterURL:   record.TCGMeisterURL,
			Memo:            record.Memo,
		})
	}

	return &dto.RecordGetResponse{
		Limit:   limit,
		Offset:  offset,
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
			PrivateFlg:      record.PrivateFlg,
			TCGMeisterURL:   record.TCGMeisterURL,
			Memo:            record.Memo,
		},
	}
}

func NewRecordGetByUserIdResponse(
	limit int,
	offset int,
	records []*entity.Record,
) *dto.RecordGetByUserIdResponse {
	ret := []*dto.RecordResponse{}

	for _, record := range records {
		ret = append(ret, &dto.RecordResponse{
			ID:              record.ID,
			CreatedAt:       record.CreatedAt,
			OfficialEventId: record.OfficialEventId,
			TonamelEventId:  record.TonamelEventId,
			FriendId:        record.FriendId,
			UserId:          record.UserId,
			DeckId:          record.DeckId,
			PrivateFlg:      record.PrivateFlg,
			TCGMeisterURL:   record.TCGMeisterURL,
			Memo:            record.Memo,
		})
	}

	return &dto.RecordGetByUserIdResponse{
		Limit:   limit,
		Offset:  offset,
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
			PrivateFlg:      record.PrivateFlg,
			TCGMeisterURL:   record.TCGMeisterURL,
			Memo:            record.Memo,
		},
	}
}
