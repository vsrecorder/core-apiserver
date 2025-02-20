package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewDeckGetResponse(
	limit int,
	offset int,
	decks []*entity.Deck,
) *dto.DeckGetResponse {
	ret := []*dto.DeckResponse{}

	for _, deck := range decks {
		ret = append(ret, &dto.DeckResponse{
			ID:             deck.ID,
			CreatedAt:      deck.CreatedAt,
			ArchivedAt:     deck.ArchivedAt,
			UserId:         deck.UserId,
			Code:           deck.Code,
			Name:           deck.Name,
			PrivateCodeFlg: deck.PrivateCodeFlg,
		})
	}

	return &dto.DeckGetResponse{
		Limit:  limit,
		Offset: offset,
		Decks:  ret,
	}
}

func NewDeckGetByIdResponse(
	deck *entity.Deck,
) *dto.DeckGetByIdResponse {
	return &dto.DeckGetByIdResponse{
		DeckResponse: dto.DeckResponse{
			ID:             deck.ID,
			CreatedAt:      deck.CreatedAt,
			ArchivedAt:     deck.ArchivedAt,
			UserId:         deck.UserId,
			Code:           deck.Code,
			Name:           deck.Name,
			PrivateCodeFlg: deck.PrivateCodeFlg,
		},
	}
}

func NewDeckGetByUserIdResponse(
	archived bool,
	limit int,
	offset int,
	decks []*entity.Deck,
) *dto.DeckGetByUserIdResponse {
	ret := []*dto.DeckResponse{}

	for _, deck := range decks {
		ret = append(ret, &dto.DeckResponse{
			ID:             deck.ID,
			CreatedAt:      deck.CreatedAt,
			ArchivedAt:     deck.ArchivedAt,
			UserId:         deck.UserId,
			Code:           deck.Code,
			Name:           deck.Name,
			PrivateCodeFlg: deck.PrivateCodeFlg,
		})
	}

	return &dto.DeckGetByUserIdResponse{
		Archived: archived,
		Limit:    limit,
		Offset:   offset,
		Decks:    ret,
	}
}

func NewDeckCreateResponse(
	deck *entity.Deck,
) *dto.DeckCreateResponse {
	return &dto.DeckCreateResponse{
		DeckResponse: dto.DeckResponse{
			ID:             deck.ID,
			CreatedAt:      deck.CreatedAt,
			ArchivedAt:     deck.ArchivedAt,
			UserId:         deck.UserId,
			Code:           deck.Code,
			Name:           deck.Name,
			PrivateCodeFlg: deck.PrivateCodeFlg,
		},
	}
}

func NewDeckUpdateResponse(
	deck *entity.Deck,
) *dto.DeckUpdateResponse {
	return &dto.DeckUpdateResponse{
		DeckResponse: dto.DeckResponse{
			ID:             deck.ID,
			CreatedAt:      deck.CreatedAt,
			ArchivedAt:     deck.ArchivedAt,
			UserId:         deck.UserId,
			Code:           deck.Code,
			Name:           deck.Name,
			PrivateCodeFlg: deck.PrivateCodeFlg,
		},
	}
}

func NewDeckArchiveResponse(
	deck *entity.Deck,
) *dto.DeckArchiveResponse {
	return &dto.DeckArchiveResponse{
		DeckResponse: dto.DeckResponse{
			ID:             deck.ID,
			CreatedAt:      deck.CreatedAt,
			ArchivedAt:     deck.ArchivedAt,
			UserId:         deck.UserId,
			Code:           deck.Code,
			Name:           deck.Name,
			PrivateCodeFlg: deck.PrivateCodeFlg,
		},
	}
}

func NewDeckUnarchiveResponse(
	deck *entity.Deck,
) *dto.DeckUnarchiveResponse {
	return &dto.DeckUnarchiveResponse{
		DeckResponse: dto.DeckResponse{
			ID:             deck.ID,
			CreatedAt:      deck.CreatedAt,
			ArchivedAt:     deck.ArchivedAt,
			UserId:         deck.UserId,
			Code:           deck.Code,
			Name:           deck.Name,
			PrivateCodeFlg: deck.PrivateCodeFlg,
		},
	}
}
