package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewDeckCodeGetByIdResponse(
	deckcode *entity.DeckCode,
) *dto.DeckCodeGetByIdResponse {
	return &dto.DeckCodeGetByIdResponse{
		DeckCodeResponse: dto.DeckCodeResponse{
			ID:             deckcode.ID,
			CreatedAt:      deckcode.CreatedAt,
			UserId:         deckcode.UserId,
			DeckId:         deckcode.DeckId,
			Code:           deckcode.Code,
			PrivateCodeFlg: deckcode.PrivateCodeFlg,
			Memo:           deckcode.Memo,
		},
	}
}

func NewDeckCodeGetByDeckIdResponse(
	deckcodes []*entity.DeckCode,
) []*dto.DeckCodeResponse {
	ret := []*dto.DeckCodeResponse{}

	for _, deckcode := range deckcodes {
		ret = append(ret, &dto.DeckCodeResponse{
			ID:             deckcode.ID,
			CreatedAt:      deckcode.CreatedAt,
			UserId:         deckcode.UserId,
			DeckId:         deckcode.DeckId,
			Code:           deckcode.Code,
			PrivateCodeFlg: deckcode.PrivateCodeFlg,
			Memo:           deckcode.Memo,
		})
	}

	return ret
}

func NewDeckCodeCreateResponse(
	deckcode *entity.DeckCode,
) *dto.DeckCodeCreateResponse {
	return &dto.DeckCodeCreateResponse{
		DeckCodeResponse: dto.DeckCodeResponse{
			ID:             deckcode.ID,
			CreatedAt:      deckcode.CreatedAt,
			UserId:         deckcode.UserId,
			DeckId:         deckcode.DeckId,
			Code:           deckcode.Code,
			PrivateCodeFlg: deckcode.PrivateCodeFlg,
			Memo:           deckcode.Memo,
		},
	}
}

func NewDeckCodeUpdateResponse(
	deckcode *entity.DeckCode,
) *dto.DeckCodeUpdateResponse {
	return &dto.DeckCodeUpdateResponse{
		DeckCodeResponse: dto.DeckCodeResponse{
			ID:             deckcode.ID,
			CreatedAt:      deckcode.CreatedAt,
			UserId:         deckcode.UserId,
			DeckId:         deckcode.DeckId,
			Code:           deckcode.Code,
			PrivateCodeFlg: deckcode.PrivateCodeFlg,
			Memo:           deckcode.Memo,
		},
	}
}
