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
		pokemonSpritesResponse := []*dto.PokemonSpriteResponse{}
		for _, pokemonSprite := range deck.PokemonSprites {
			pokemonSpritesResponse = append(pokemonSpritesResponse, &dto.PokemonSpriteResponse{
				ID: pokemonSprite.ID,
			})
		}

		ret = append(ret, &dto.DeckData{
			Cursor: base64.StdEncoding.EncodeToString([]byte(deck.CreatedAt.Format(time.RFC3339))),
			Data: &dto.DeckResponse{
				ID:         deck.ID,
				CreatedAt:  deck.CreatedAt,
				ArchivedAt: deck.ArchivedAt,
				UserId:     deck.UserId,
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
				PokemonSprites: pokemonSpritesResponse,
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
		pokemonSpritesResponse := []*dto.PokemonSpriteResponse{}
		for _, pokemonSprite := range deck.PokemonSprites {
			pokemonSpritesResponse = append(pokemonSpritesResponse, &dto.PokemonSpriteResponse{
				ID: pokemonSprite.ID,
			})
		}

		ret = append(ret, dto.DeckResponse{
			ID:         deck.ID,
			CreatedAt:  deck.CreatedAt,
			ArchivedAt: deck.ArchivedAt,
			UserId:     deck.UserId,
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
			PokemonSprites: pokemonSpritesResponse,
		})
	}

	return &ret
}

func NewDeckGetByIdResponse(
	deck *entity.Deck,
) *dto.DeckGetByIdResponse {
	pokemonSpritesResponse := []*dto.PokemonSpriteResponse{}
	for _, pokemonSprite := range deck.PokemonSprites {
		pokemonSpritesResponse = append(pokemonSpritesResponse, &dto.PokemonSpriteResponse{
			ID: pokemonSprite.ID,
		})
	}

	return &dto.DeckGetByIdResponse{
		DeckResponse: dto.DeckResponse{
			ID:         deck.ID,
			CreatedAt:  deck.CreatedAt,
			ArchivedAt: deck.ArchivedAt,
			UserId:     deck.UserId,
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
			PokemonSprites: pokemonSpritesResponse,
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
		pokemonSpritesResponse := []*dto.PokemonSpriteResponse{}
		for _, pokemonSprite := range deck.PokemonSprites {
			pokemonSpritesResponse = append(pokemonSpritesResponse, &dto.PokemonSpriteResponse{
				ID: pokemonSprite.ID,
			})
		}

		ret = append(ret, &dto.DeckData{
			Cursor: base64.StdEncoding.EncodeToString([]byte(deck.CreatedAt.Format(time.RFC3339))),
			Data: &dto.DeckResponse{
				ID:         deck.ID,
				CreatedAt:  deck.CreatedAt,
				ArchivedAt: deck.ArchivedAt,
				UserId:     deck.UserId,
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
				PokemonSprites: pokemonSpritesResponse,
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

	pokemonSpritesResponse := []*dto.PokemonSpriteResponse{}
	for _, pokemonSprite := range deck.PokemonSprites {
		pokemonSpritesResponse = append(pokemonSpritesResponse, &dto.PokemonSpriteResponse{
			ID: pokemonSprite.ID,
		})
	}

	return &dto.DeckCreateResponse{
		DeckResponse: dto.DeckResponse{
			ID:         deck.ID,
			CreatedAt:  deck.CreatedAt,
			ArchivedAt: deck.ArchivedAt,
			UserId:     deck.UserId,
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
			PokemonSprites: pokemonSpritesResponse,
		},
	}
}

func NewDeckUpdateResponse(
	deck *entity.Deck,
) *dto.DeckUpdateResponse {
	pokemonSpritesResponse := []*dto.PokemonSpriteResponse{}
	for _, pokemonSprite := range deck.PokemonSprites {
		pokemonSpritesResponse = append(pokemonSpritesResponse, &dto.PokemonSpriteResponse{
			ID: pokemonSprite.ID,
		})
	}

	return &dto.DeckUpdateResponse{
		DeckResponse: dto.DeckResponse{
			ID:         deck.ID,
			CreatedAt:  deck.CreatedAt,
			ArchivedAt: deck.ArchivedAt,
			UserId:     deck.UserId,
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
			PokemonSprites: pokemonSpritesResponse,
		},
	}
}

func NewDeckArchiveResponse(
	deck *entity.Deck,
) *dto.DeckArchiveResponse {
	pokemonSpritesResponse := []*dto.PokemonSpriteResponse{}
	for _, pokemonSprite := range deck.PokemonSprites {
		pokemonSpritesResponse = append(pokemonSpritesResponse, &dto.PokemonSpriteResponse{
			ID: pokemonSprite.ID,
		})
	}

	return &dto.DeckArchiveResponse{
		DeckResponse: dto.DeckResponse{
			ID:         deck.ID,
			CreatedAt:  deck.CreatedAt,
			ArchivedAt: deck.ArchivedAt,
			UserId:     deck.UserId,
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
			PokemonSprites: pokemonSpritesResponse,
		},
	}
}

func NewDeckUnarchiveResponse(
	deck *entity.Deck,
) *dto.DeckUnarchiveResponse {
	pokemonSpritesResponse := []*dto.PokemonSpriteResponse{}
	for _, pokemonSprite := range deck.PokemonSprites {
		pokemonSpritesResponse = append(pokemonSpritesResponse, &dto.PokemonSpriteResponse{
			ID: pokemonSprite.ID,
		})
	}

	return &dto.DeckUnarchiveResponse{
		DeckResponse: dto.DeckResponse{
			ID:         deck.ID,
			CreatedAt:  deck.CreatedAt,
			ArchivedAt: deck.ArchivedAt,
			UserId:     deck.UserId,
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
			PokemonSprites: pokemonSpritesResponse,
		},
	}
}
