package presenter

import (
	"encoding/base64"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// newDeckResponse は entity.Deck を DeckResponse に変換する共通処理。
// 各レスポンス(一覧/詳細/作成/更新/アーカイブ/お気に入り)で同じマッピングを使う。
func newDeckResponse(deck *entity.Deck) dto.DeckResponse {
	pokemonSpritesResponse := []*dto.PokemonSpriteResponse{}
	for _, pokemonSprite := range deck.PokemonSprites {
		pokemonSpritesResponse = append(pokemonSpritesResponse, &dto.PokemonSpriteResponse{
			ID:       pokemonSprite.ID,
			Position: pokemonSprite.Position,
		})
	}

	return dto.DeckResponse{
		ID:          deck.ID,
		CreatedAt:   deck.CreatedAt,
		ArchivedAt:  deck.ArchivedAt,
		FavoritedAt: deck.FavoritedAt,
		UserId:      deck.UserId,
		Name:        deck.Name,
		PrivateFlg:  deck.PrivateFlg,
		LatestDeckCode: dto.DeckCodeResponse{
			ID:             deck.LatestDeckCode.ID,
			CreatedAt:      deck.LatestDeckCode.CreatedAt,
			UserId:         deck.LatestDeckCode.UserId,
			DeckId:         deck.LatestDeckCode.DeckId,
			Code:           deck.LatestDeckCode.Code,
			PrivateCodeFlg: deck.LatestDeckCode.PrivateCodeFlg,
			Memo:           deck.LatestDeckCode.Memo,
			// 新バージョン作成時のタグ継承などで使うため、最新バージョンの付与タグも返す。
			Tags: newTagResponses(deck.LatestDeckCode.Tags),
		},
		PokemonSprites: pokemonSpritesResponse,
		Tags:           newTagResponses(deck.Tags),
	}
}

// newDeckDataList はカーソルページング用の DeckData 配列を作る。
func newDeckDataList(decks []*entity.Deck) []*dto.DeckData {
	ret := []*dto.DeckData{}
	for _, deck := range decks {
		res := newDeckResponse(deck)
		ret = append(ret, &dto.DeckData{
			Cursor: base64.StdEncoding.EncodeToString([]byte(deck.CreatedAt.Format(time.RFC3339))),
			Data:   &res,
		})
	}
	return ret
}

func NewDeckGetResponse(
	limit int,
	offset int,
	cursor time.Time,
	decks []*entity.Deck,
) *dto.DeckGetResponse {
	return &dto.DeckGetResponse{
		Limit:  limit,
		Offset: offset,
		Cursor: base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339))),
		Decks:  newDeckDataList(decks),
	}
}

func NewDeckGetAllResponse(
	decks []*entity.Deck,
) *dto.DeckGetAllResponse {
	ret := dto.DeckGetAllResponse{}
	for _, deck := range decks {
		ret = append(ret, newDeckResponse(deck))
	}

	return &ret
}

func NewDeckGetByIdResponse(
	deck *entity.Deck,
) *dto.DeckGetByIdResponse {
	return &dto.DeckGetByIdResponse{
		DeckResponse: newDeckResponse(deck),
	}
}

func NewDeckGetByUserIdResponse(
	archived bool,
	limit int,
	offset int,
	cursor time.Time,
	decks []*entity.Deck,
) *dto.DeckGetByUserIdResponse {
	return &dto.DeckGetByUserIdResponse{
		Archived: archived,
		Limit:    limit,
		Offset:   offset,
		Cursor:   base64.StdEncoding.EncodeToString([]byte(cursor.Format(time.RFC3339))),
		Decks:    newDeckDataList(decks),
	}
}

func NewDeckCreateResponse(
	deck *entity.Deck,
) *dto.DeckCreateResponse {
	return &dto.DeckCreateResponse{
		DeckResponse: newDeckResponse(deck),
	}
}

func NewDeckUpdateResponse(
	deck *entity.Deck,
) *dto.DeckUpdateResponse {
	return &dto.DeckUpdateResponse{
		DeckResponse: newDeckResponse(deck),
	}
}

func NewDeckArchiveResponse(
	deck *entity.Deck,
) *dto.DeckArchiveResponse {
	return &dto.DeckArchiveResponse{
		DeckResponse: newDeckResponse(deck),
	}
}

func NewDeckFavoriteResponse(
	deck *entity.Deck,
) *dto.DeckFavoriteResponse {
	return &dto.DeckFavoriteResponse{
		DeckResponse: newDeckResponse(deck),
	}
}

func NewDeckUnfavoriteResponse(
	deck *entity.Deck,
) *dto.DeckUnfavoriteResponse {
	return &dto.DeckUnfavoriteResponse{
		DeckResponse: newDeckResponse(deck),
	}
}

func NewDeckUnarchiveResponse(
	deck *entity.Deck,
) *dto.DeckUnarchiveResponse {
	return &dto.DeckUnarchiveResponse{
		DeckResponse: newDeckResponse(deck),
	}
}
