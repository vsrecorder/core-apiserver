package presenter

import (
	"encoding/base64"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewDeckGetResponse(
	limit int,
	offset int,
	cursor time.Time,
	decks []*entity.Deck,
) *dto.DeckGetResponse {
	ret := []*dto.DeckData{}

	for _, deck := range decks {
		ret = append(ret, &dto.DeckData{
			Cursor: base64.StdEncoding.EncodeToString([]byte(deck.CreatedAt.Format(time.RFC3339))),
			Data: &dto.DeckResponse{
				ID:         deck.ID,
				CreatedAt:  deck.CreatedAt,
				ArchivedAt: deck.ArchivedAt,
				UserId:     deck.UserId,
				Code:       deck.Code,
				Name:       deck.Name,
				PrivateFlg: deck.PrivateFlg,
				LatestDeckCode: dto.DeckCodeResponse{
					ID:             deck.LatestDeckCode.ID,
					CreatedAt:      deck.LatestDeckCode.CreatedAt,
					UserId:         deck.LatestDeckCode.UserId,
					DeckId:         deck.LatestDeckCode.DeckId,
					Code:           deck.LatestDeckCode.Code,
					PrivateCodeFlg: deck.LatestDeckCode.PrivateCodeFlg,
					Memo:           deck.LatestDeckCode.Memo,
				},
			},
		})
	}

	return &dto.DeckGetResponse{
		Limit:  limit,
		Offset: offset,
		Cursor: base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339))),
		Decks:  ret,
	}
}

func NewDeckGetAllResponse(
	decks []*entity.Deck,
) *dto.DeckGetAllResponse {
	ret := dto.DeckGetAllResponse{}
	for _, deck := range decks {
		ret = append(ret, dto.DeckResponse{
			ID:         deck.ID,
			CreatedAt:  deck.CreatedAt,
			ArchivedAt: deck.ArchivedAt,
			UserId:     deck.UserId,
			Code:       deck.Code,
			Name:       deck.Name,
			PrivateFlg: deck.PrivateFlg,
			LatestDeckCode: dto.DeckCodeResponse{
				ID:             deck.LatestDeckCode.ID,
				CreatedAt:      deck.LatestDeckCode.CreatedAt,
				UserId:         deck.LatestDeckCode.UserId,
				DeckId:         deck.LatestDeckCode.DeckId,
				Code:           deck.LatestDeckCode.Code,
				PrivateCodeFlg: deck.LatestDeckCode.PrivateCodeFlg,
				Memo:           deck.LatestDeckCode.Memo,
			},
		})
	}

	return &ret
}

func NewDeckGetByIdResponse(
	deck *entity.Deck,
) *dto.DeckGetByIdResponse {
	return &dto.DeckGetByIdResponse{
		DeckResponse: dto.DeckResponse{
			ID:         deck.ID,
			CreatedAt:  deck.CreatedAt,
			ArchivedAt: deck.ArchivedAt,
			UserId:     deck.UserId,
			Code:       deck.Code,
			Name:       deck.Name,
			PrivateFlg: deck.PrivateFlg,
			LatestDeckCode: dto.DeckCodeResponse{
				ID:             deck.LatestDeckCode.ID,
				CreatedAt:      deck.LatestDeckCode.CreatedAt,
				UserId:         deck.LatestDeckCode.UserId,
				DeckId:         deck.LatestDeckCode.DeckId,
				Code:           deck.LatestDeckCode.Code,
				PrivateCodeFlg: deck.LatestDeckCode.PrivateCodeFlg,
				Memo:           deck.LatestDeckCode.Memo,
			},
		},
	}
}

func NewDeckGetByUserIdResponse(
	archived bool,
	limit int,
	offset int,
	cursor time.Time,
	decks []*entity.Deck,
) *dto.DeckGetByUserIdResponse {
	ret := []*dto.DeckData{}

	for _, deck := range decks {
		ret = append(ret, &dto.DeckData{
			Cursor: base64.StdEncoding.EncodeToString([]byte(deck.CreatedAt.Format(time.RFC3339))),
			Data: &dto.DeckResponse{
				ID:         deck.ID,
				CreatedAt:  deck.CreatedAt,
				ArchivedAt: deck.ArchivedAt,
				UserId:     deck.UserId,
				Code:       deck.Code,
				Name:       deck.Name,
				PrivateFlg: deck.PrivateFlg,
				LatestDeckCode: dto.DeckCodeResponse{
					ID:             deck.LatestDeckCode.ID,
					CreatedAt:      deck.LatestDeckCode.CreatedAt,
					UserId:         deck.LatestDeckCode.UserId,
					DeckId:         deck.LatestDeckCode.DeckId,
					Code:           deck.LatestDeckCode.Code,
					PrivateCodeFlg: deck.LatestDeckCode.PrivateCodeFlg,
					Memo:           deck.LatestDeckCode.Memo,
				},
			},
		})
	}

	return &dto.DeckGetByUserIdResponse{
		Archived: archived,
		Limit:    limit,
		Offset:   offset,
		Cursor:   base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339))),
		Decks:    ret,
	}
}

func NewDeckCreateResponse(
	deck *entity.Deck,
) *dto.DeckCreateResponse {
	return &dto.DeckCreateResponse{
		DeckResponse: dto.DeckResponse{
			ID:         deck.ID,
			CreatedAt:  deck.CreatedAt,
			ArchivedAt: deck.ArchivedAt,
			UserId:     deck.UserId,
			Code:       deck.Code,
			Name:       deck.Name,
			PrivateFlg: deck.PrivateFlg,
			LatestDeckCode: dto.DeckCodeResponse{
				ID:             deck.LatestDeckCode.ID,
				CreatedAt:      deck.LatestDeckCode.CreatedAt,
				UserId:         deck.LatestDeckCode.UserId,
				DeckId:         deck.LatestDeckCode.DeckId,
				Code:           deck.LatestDeckCode.Code,
				PrivateCodeFlg: deck.LatestDeckCode.PrivateCodeFlg,
				Memo:           deck.LatestDeckCode.Memo,
			},
		},
	}
}

func NewDeckUpdateResponse(
	deck *entity.Deck,
) *dto.DeckUpdateResponse {
	return &dto.DeckUpdateResponse{
		DeckResponse: dto.DeckResponse{
			ID:         deck.ID,
			CreatedAt:  deck.CreatedAt,
			ArchivedAt: deck.ArchivedAt,
			UserId:     deck.UserId,
			Code:       deck.Code,
			Name:       deck.Name,
			PrivateFlg: deck.PrivateFlg,
			LatestDeckCode: dto.DeckCodeResponse{
				ID:             deck.LatestDeckCode.ID,
				CreatedAt:      deck.LatestDeckCode.CreatedAt,
				UserId:         deck.LatestDeckCode.UserId,
				DeckId:         deck.LatestDeckCode.DeckId,
				Code:           deck.LatestDeckCode.Code,
				PrivateCodeFlg: deck.LatestDeckCode.PrivateCodeFlg,
				Memo:           deck.LatestDeckCode.Memo,
			},
		},
	}
}

func NewDeckArchiveResponse(
	deck *entity.Deck,
) *dto.DeckArchiveResponse {
	return &dto.DeckArchiveResponse{
		DeckResponse: dto.DeckResponse{
			ID:         deck.ID,
			CreatedAt:  deck.CreatedAt,
			ArchivedAt: deck.ArchivedAt,
			UserId:     deck.UserId,
			Code:       deck.Code,
			Name:       deck.Name,
			PrivateFlg: deck.PrivateFlg,
			LatestDeckCode: dto.DeckCodeResponse{
				ID:             deck.LatestDeckCode.ID,
				CreatedAt:      deck.LatestDeckCode.CreatedAt,
				UserId:         deck.LatestDeckCode.UserId,
				DeckId:         deck.LatestDeckCode.DeckId,
				Code:           deck.LatestDeckCode.Code,
				PrivateCodeFlg: deck.LatestDeckCode.PrivateCodeFlg,
				Memo:           deck.LatestDeckCode.Memo,
			},
		},
	}
}

func NewDeckUnarchiveResponse(
	deck *entity.Deck,
) *dto.DeckUnarchiveResponse {
	return &dto.DeckUnarchiveResponse{
		DeckResponse: dto.DeckResponse{
			ID:         deck.ID,
			CreatedAt:  deck.CreatedAt,
			ArchivedAt: deck.ArchivedAt,
			UserId:     deck.UserId,
			Code:       deck.Code,
			Name:       deck.Name,
			PrivateFlg: deck.PrivateFlg,
			LatestDeckCode: dto.DeckCodeResponse{
				ID:             deck.LatestDeckCode.ID,
				CreatedAt:      deck.LatestDeckCode.CreatedAt,
				UserId:         deck.LatestDeckCode.UserId,
				DeckId:         deck.LatestDeckCode.DeckId,
				Code:           deck.LatestDeckCode.Code,
				PrivateCodeFlg: deck.LatestDeckCode.PrivateCodeFlg,
				Memo:           deck.LatestDeckCode.Memo,
			},
		},
	}
}
