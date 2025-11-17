package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/gorm"
)

type DeckParam struct {
	UserId         string
	Name           string
	Code           string
	PrivateCodeFlg bool
}

func NewDeckParam(
	userId string,
	name string,
	code string,
	privateCodeFlg bool,
) *DeckParam {
	return &DeckParam{
		UserId:         userId,
		Name:           name,
		Code:           code,
		PrivateCodeFlg: privateCodeFlg,
	}
}

type DeckInterface interface {
	Find(
		ctx context.Context,
		limit int,
		offset int,
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
		param *DeckParam,
	) (*entity.Deck, error)

	Update(
		ctx context.Context,
		id string,
		param *DeckParam,
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
	param *DeckParam,
) (*entity.Deck, error) {
	id, err := generateId()
	if err != nil {
		return nil, err
	}

	createdAt := time.Now().Local()
	archivedAt := time.Time{}

	deck := entity.NewDeck(
		id,
		createdAt,
		archivedAt,
		param.UserId,
		param.Name,
		param.Code,
		param.PrivateCodeFlg,
	)

	if deck.Code != "" {
		if err := uploadDeckImage(deck.Code); err != nil {
			return nil, err
		}
	}

	if err := u.repository.Save(ctx, deck); err != nil {
		return nil, err
	}

	return deck, nil
}

func (u *Deck) Update(
	ctx context.Context,
	id string,
	param *DeckParam,
) (*entity.Deck, error) {
	// 指定されたidのDeckが存在するか確認
	ret, err := u.repository.FindById(ctx, id)
	if err == gorm.ErrRecordNotFound {
		return nil, err
	} else if err != nil {
		return nil, err
	}

	deck := entity.NewDeck(
		id,
		ret.CreatedAt,
		ret.ArchivedAt,
		param.UserId,
		param.Name,
		param.Code,
		param.PrivateCodeFlg,
	)

	// デッキコードの変更は禁止
	// 元のデッキコードが空の場合は変更を許可
	if ret.Code != "" && ret.Code != deck.Code {
		return nil, errors.New("deck code change is not allowed")
	}

	if deck.Code != "" {
		if err := uploadDeckImage(deck.Code); err != nil {
			return nil, err
		}
	}

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
		ret.Code,
		ret.PrivateCodeFlg,
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
		ret.Code,
		ret.PrivateCodeFlg,
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
