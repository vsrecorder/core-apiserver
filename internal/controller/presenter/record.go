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
	return &dto.RecordGetResponse{
		Limit:   limit,
		Offset:  offset,
		Records: records,
	}
}

func NewRecordCreateResponse(
	record *entity.Record,
) *dto.RecordCreateResponse {
	return &dto.RecordCreateResponse{
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
	}
}

func NewRecordUpdateResponse(
	record *entity.Record,
) *dto.RecordUpdateResponse {
	return &dto.RecordUpdateResponse{
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
	}
}
