package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// newDeckCodeResponse は entity.DeckCode を DeckCodeResponse に変換する共通処理。
func newDeckCodeResponse(deckcode *entity.DeckCode) dto.DeckCodeResponse {
	// 未設定でも null ではなく空配列で返す(クライアントが配列前提で扱えるように)。
	pokemonSprites := []*dto.PokemonSpriteResponse{}
	for _, pokemonSprite := range deckcode.PokemonSprites {
		pokemonSprites = append(pokemonSprites, &dto.PokemonSpriteResponse{
			ID:       pokemonSprite.ID,
			Position: pokemonSprite.Position,
		})
	}

	return dto.DeckCodeResponse{
		ID:             deckcode.ID,
		CreatedAt:      deckcode.CreatedAt,
		UserId:         deckcode.UserId,
		DeckId:         deckcode.DeckId,
		Code:           deckcode.Code,
		PrivateCodeFlg: deckcode.PrivateCodeFlg,
		Memo:           deckcode.Memo,
		Tags:           newTagResponses(deckcode.Tags),
		PokemonSprites: pokemonSprites,
	}
}

func NewDeckCodeGetByIdResponse(
	deckcode *entity.DeckCode,
) *dto.DeckCodeGetByIdResponse {
	return &dto.DeckCodeGetByIdResponse{
		DeckCodeResponse: newDeckCodeResponse(deckcode),
	}
}

func NewDeckCodeGetByDeckIdResponse(
	deckcodes []*entity.DeckCode,
) []*dto.DeckCodeResponse {
	ret := []*dto.DeckCodeResponse{}

	for _, deckcode := range deckcodes {
		res := newDeckCodeResponse(deckcode)
		ret = append(ret, &res)
	}

	return ret
}

func NewDeckCodeCreateResponse(
	deckcode *entity.DeckCode,
) *dto.DeckCodeCreateResponse {
	return &dto.DeckCodeCreateResponse{
		DeckCodeResponse: newDeckCodeResponse(deckcode),
	}
}

func NewDeckCodeUpdateResponse(
	deckcode *entity.DeckCode,
) *dto.DeckCodeUpdateResponse {
	return &dto.DeckCodeUpdateResponse{
		DeckCodeResponse: newDeckCodeResponse(deckcode),
	}
}
