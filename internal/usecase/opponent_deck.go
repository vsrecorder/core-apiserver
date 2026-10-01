package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// OpponentDeckSpecParam は相手デッキの表記とスプライトの指定。
type OpponentDeckSpecParam struct {
	OpponentsDeckInfo string
	PokemonSprites    []*PokemonSpriteParam
}

func NewOpponentDeckSpecParam(
	opponentsDeckInfo string,
	pokemonSprites []*PokemonSpriteParam,
) *OpponentDeckSpecParam {
	return &OpponentDeckSpecParam{
		OpponentsDeckInfo: opponentsDeckInfo,
		PokemonSprites:    pokemonSprites,
	}
}

// OpponentDeckReplaceParam は相手デッキの一括置き換えの指定。From と同じ組み合わせの対戦を To にする。
type OpponentDeckReplaceParam struct {
	From *OpponentDeckSpecParam
	To   *OpponentDeckSpecParam
}

func NewOpponentDeckReplaceParam(
	from *OpponentDeckSpecParam,
	to *OpponentDeckSpecParam,
) *OpponentDeckReplaceParam {
	return &OpponentDeckReplaceParam{
		From: from,
		To:   to,
	}
}

type OpponentDeckInterface interface {
	// FindByUserId は uid 自身の対戦結果に付けた相手デッキの組み合わせを、対戦の多い順に返す。
	FindByUserId(
		ctx context.Context,
		uid string,
	) ([]*entity.OpponentDeck, error)

	// Replace は uid 自身の対戦結果のうち、相手デッキが param.From と同じものを param.To に
	// 置き換え、置き換えた対戦の数を返す。
	Replace(
		ctx context.Context,
		uid string,
		param *OpponentDeckReplaceParam,
	) (int, error)
}

type OpponentDeck struct {
	repository repository.OpponentDeckInterface
}

func NewOpponentDeck(
	repository repository.OpponentDeckInterface,
) OpponentDeckInterface {
	return &OpponentDeck{repository}
}

func (u *OpponentDeck) FindByUserId(
	ctx context.Context,
	uid string,
) ([]*entity.OpponentDeck, error) {
	decks, err := u.repository.FindByUserId(ctx, uid)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return decks, nil
}

func (u *OpponentDeck) Replace(
	ctx context.Context,
	uid string,
	param *OpponentDeckReplaceParam,
) (int, error) {
	from := opponentDeckSpecOf(param.From)
	to := opponentDeckSpecOf(param.To)

	updated, err := u.repository.Replace(ctx, uid, from, to)
	if err != nil {
		logError(ctx, err)
		return 0, err
	}

	return updated, nil
}

// opponentDeckSpecOf は指定を entity にする。position の省略(0)は、対戦結果の保存と同じく
// 配列の並び順(1, 2)で採番する(省略と指定の混在は validation で弾いている)。
func opponentDeckSpecOf(param *OpponentDeckSpecParam) *entity.OpponentDeckSpec {
	sprites := make([]*entity.PokemonSprite, 0, len(param.PokemonSprites))
	for i, sprite := range param.PokemonSprites {
		position := sprite.Position
		if position == 0 {
			position = uint(i + 1)
		}
		sprites = append(sprites, entity.NewPokemonSpriteWithPosition(sprite.ID, position))
	}

	return entity.NewOpponentDeckSpec(param.OpponentsDeckInfo, sprites)
}
