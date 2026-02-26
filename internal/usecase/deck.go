package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/gorm"
)

type DeckCreateParam struct {
	UserId             string
	Name               string
	PrivateFlg         bool
	DeckCode           string
	PrivateDeckCodeFlg bool
}

func NewDeckCreateParam(
	userId string,
	name string,
	privateFlg bool,
	deckcode string,
	privateDeckCodeFlg bool,
) *DeckCreateParam {
	return &DeckCreateParam{
		UserId:             userId,
		Name:               name,
		PrivateFlg:         privateFlg,
		DeckCode:           deckcode,
		PrivateDeckCodeFlg: privateDeckCodeFlg,
	}
}

type DeckUpdateParam struct {
	Name           string
	PrivateFlg     bool
	PokemonSprites []*PokemonSpriteParam
}

func NewDeckUpdateParam(
	name string,
	privateFlg bool,
	pokemonSprites []*PokemonSpriteParam,
) *DeckUpdateParam {
	return &DeckUpdateParam{
		Name:           name,
		PrivateFlg:     privateFlg,
		PokemonSprites: pokemonSprites,
	}
}

type DeckInterface interface {
	Find(
		ctx context.Context,
		limit int,
		offset int,
	) ([]*entity.Deck, error)

	FindAll(
		ctx context.Context,
		uid string,
	) ([]*entity.Deck, error)

	FindOnCursor(
		ctx context.Context,
		limit int,
		cursor time.Time,
	) ([]*entity.Deck, error)

	FindById(
		ctx context.Context,
		id string,
	) (*entity.Deck, error)

	FindByUserId(
		ctx context.Context,
		uid string,
		archivedFlg bool,
		limit int,
		offset int,
	) ([]*entity.Deck, error)

	FindByUserIdOnCursor(
		ctx context.Context,
		uid string,
		archivedFlg bool,
		limit int,
		cursor time.Time,
	) ([]*entity.Deck, error)

	Create(
		ctx context.Context,
		param *DeckCreateParam,
	) (*entity.Deck, error)

	Update(
		ctx context.Context,
		id string,
		param *DeckUpdateParam,
	) (*entity.Deck, error)

	Archive(
		ctx context.Context,
		id string,
	) (*entity.Deck, error)

	Unarchive(
		ctx context.Context,
		id string,
	) (*entity.Deck, error)

	Delete(
		ctx context.Context,
		id string,
	) error
}

type Deck struct {
	repository repository.DeckInterface
}

func NewDeck(
	repository repository.DeckInterface,
) DeckInterface {
	return &Deck{repository}
}

func (u *Deck) Find(
	ctx context.Context,
	limit int,
	offset int,
) ([]*entity.Deck, error) {
	decks, err := u.repository.Find(ctx, limit, offset)

	if err != nil {
		return nil, err
	}

	return decks, nil
}

func (u *Deck) FindAll(
	ctx context.Context,
	uid string,
) ([]*entity.Deck, error) {
	decks, err := u.repository.FindAll(ctx, uid)

	if err != nil {
		return nil, err
	}

	return decks, nil
}

func (u *Deck) FindOnCursor(
	ctx context.Context,
	limit int,
	cursor time.Time,
) ([]*entity.Deck, error) {
	decks, err := u.repository.FindOnCursor(ctx, limit, cursor)

	if err != nil {
		return nil, err
	}

	return decks, nil
}

func (u *Deck) FindById(
	ctx context.Context,
	id string,
) (*entity.Deck, error) {
	deck, err := u.repository.FindById(ctx, id)

	if err != nil {
		return nil, err
	}

	return deck, nil
}

func (u *Deck) FindByUserId(
	ctx context.Context,
	uid string,
	archivedFlg bool,
	limit int,
	offset int,
) ([]*entity.Deck, error) {
	decks, err := u.repository.FindByUserId(ctx, uid, archivedFlg, limit, offset)

	if err != nil {
		return nil, err
	}

	return decks, nil
}

func (u *Deck) FindByUserIdOnCursor(
	ctx context.Context,
	uid string,
	archivedFlg bool,
	limit int,
	cursor time.Time,
) ([]*entity.Deck, error) {
	decks, err := u.repository.FindByUserIdOnCursor(ctx, uid, archivedFlg, limit, cursor)

	if err != nil {
		return nil, err
	}

	return decks, nil
}

func (u *Deck) Create(
	ctx context.Context,
	param *DeckCreateParam,
) (*entity.Deck, error) {
	deckId, err := generateId()
	if err != nil {
		return nil, err
	}

	createdAt := time.Now().Local()
	archivedAt := time.Time{}
	var LatestDeckCode *entity.DeckCode

	if param.DeckCode != "" {
		deckCodeId, err := generateId()
		if err != nil {
			return nil, err
		}

		memo := ""
		LatestDeckCode = entity.NewDeckCode(
			deckCodeId,
			createdAt,
			param.UserId,
			deckId,
			param.DeckCode,
			param.PrivateDeckCodeFlg,
			memo,
		)

		if LatestDeckCode.Code != "" {
			if err := uploadDeckImage(LatestDeckCode.Code); err != nil {
				return nil, err
			}
		}
	}

	var pokemonSprites []*entity.PokemonSprite

	deck := entity.NewDeck(
		deckId,
		createdAt,
		archivedAt,
		param.UserId,
		param.Name,
		param.PrivateFlg,
		LatestDeckCode,
		pokemonSprites,
	)

	if err := u.repository.Save(ctx, deck); err != nil {
		return nil, err
	}

	return deck, nil
}

func (u *Deck) Update(
	ctx context.Context,
	id string,
	param *DeckUpdateParam,
) (*entity.Deck, error) {
	// 指定されたidのDeckが存在するか確認
	ret, err := u.repository.FindById(ctx, id)
	if err == gorm.ErrRecordNotFound {
		return nil, err
	} else if err != nil {
		return nil, err
	}

	var pokemonSprites []*entity.PokemonSprite
	for _, pokemonSprite := range param.PokemonSprites {
		pokemonSprites = append(pokemonSprites, entity.NewPokemonSprite(pokemonSprite.ID))
	}

	deck := entity.NewDeck(
		id,
		ret.CreatedAt,
		ret.ArchivedAt,
		ret.UserId,
		param.Name,
		param.PrivateFlg,
		ret.LatestDeckCode,
		pokemonSprites,
	)

	if err := u.repository.Save(ctx, deck); err != nil {
		return nil, err
	}

	return deck, nil
}

func (u *Deck) Archive(
	ctx context.Context,
	id string,
) (*entity.Deck, error) {
	// 指定されたidのDeckが存在するか確認
	ret, err := u.repository.FindById(ctx, id)
	if err == gorm.ErrRecordNotFound {
		return nil, err
	}

	archivedAt := time.Now().Local()

	deck := entity.NewDeck(
		id,
		ret.CreatedAt,
		archivedAt,
		ret.UserId,
		ret.Name,
		ret.PrivateFlg,
		ret.LatestDeckCode,
		ret.PokemonSprites,
	)

	if err := u.repository.Save(ctx, deck); err != nil {
		return nil, err
	}

	return deck, nil
}

func (u *Deck) Unarchive(
	ctx context.Context,
	id string,
) (*entity.Deck, error) {
	// 指定されたidのDeckが存在するか確認
	ret, err := u.repository.FindById(ctx, id)
	if err == gorm.ErrRecordNotFound {
		return nil, err
	}

	archivedAt := time.Time{}

	deck := entity.NewDeck(
		id,
		ret.CreatedAt,
		archivedAt,
		ret.UserId,
		ret.Name,
		ret.PrivateFlg,
		ret.LatestDeckCode,
		ret.PokemonSprites,
	)

	if err := u.repository.Save(ctx, deck); err != nil {
		return nil, err
	}

	return deck, nil
}

func (u *Deck) Delete(
	ctx context.Context,
	id string,
) error {
	err := u.repository.Delete(ctx, id)

	if err != nil {
		return err
	}

	return nil
}
