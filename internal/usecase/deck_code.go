package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/gorm"
)

type DeckCodeCreateParam struct {
	UserId     string
	DeckId     string
	Code       string
	PrivateFlg bool
}

type DeckCodeUpdateParam struct {
	PrivateFlg bool
}

func NewDeckCodeCreateParam(
	userId string,
	deckId string,
	code string,
	privateFlg bool,
) *DeckCodeCreateParam {
	return &DeckCodeCreateParam{
		UserId:     userId,
		DeckId:     deckId,
		Code:       code,
		PrivateFlg: privateFlg,
	}
}

func NewDeckCodeUpdateParam(
	privateFlg bool,
) *DeckCodeUpdateParam {
	return &DeckCodeUpdateParam{
		PrivateFlg: privateFlg,
	}
}

type DeckCodeInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.DeckCode, error)

	FindByDeckId(
		ctx context.Context,
		deckId string,
	) ([]*entity.DeckCode, error)

	Create(
		ctx context.Context,
		param *DeckCodeCreateParam,
	) (*entity.DeckCode, error)

	Update(
		ctx context.Context,
		id string,
		param *DeckCodeUpdateParam,
	) (*entity.DeckCode, error)

	Delete(
		ctx context.Context,
		id string,
	) error
}

type DeckCode struct {
	repository repository.DeckCodeInterface
}

func NewDeckCode(
	repository repository.DeckCodeInterface,
) DeckCodeInterface {
	return &DeckCode{repository}
}

func (u *DeckCode) FindById(
	ctx context.Context,
	id string,
) (*entity.DeckCode, error) {
	deckcode, err := u.repository.FindById(ctx, id)

	if err != nil {
		return nil, err
	}

	return deckcode, nil
}

func (u *DeckCode) FindByDeckId(
	ctx context.Context,
	deckId string,
) ([]*entity.DeckCode, error) {
	deckcodes, err := u.repository.FindByDeckId(ctx, deckId)

	if err != nil {
		return nil, err
	}

	return deckcodes, nil
}

func (u *DeckCode) Create(
	ctx context.Context,
	param *DeckCodeCreateParam,
) (*entity.DeckCode, error) {

	// TODO: param.DeckIdが存在するか確認する(or 外部キー制約を利用する)

	id, err := generateId()
	if err != nil {
		return nil, err
	}

	createdAt := time.Now().Local()

	deckcode := entity.NewDeckCode(
		id,
		createdAt,
		param.UserId,
		param.DeckId,
		param.Code,
		param.PrivateFlg,
	)

	if deckcode.Code != "" {
		if err := uploadDeckImage(deckcode.Code); err != nil {
			return nil, err
		}
	}

	if err := u.repository.Save(ctx, deckcode); err != nil {
		return nil, err
	}

	return deckcode, nil
}

func (u *DeckCode) Update(
	ctx context.Context,
	id string,
	param *DeckCodeUpdateParam,
) (*entity.DeckCode, error) {
	// 指定されたidのDeckCodeが存在するか確認
	ret, err := u.repository.FindById(ctx, id)
	if err == gorm.ErrRecordNotFound {
		return nil, err
	} else if err != nil {
		return nil, err
	}

	deckcode := entity.NewDeckCode(
		id,
		ret.CreatedAt,
		ret.UserId,
		ret.DeckId,
		ret.Code,
		param.PrivateFlg,
	)

	if err := u.repository.Save(ctx, deckcode); err != nil {
		return nil, err
	}

	return deckcode, nil
}

func (u *DeckCode) Delete(
	ctx context.Context,
	id string,
) error {
	err := u.repository.Delete(ctx, id)

	if err != nil {
		return err
	}

	return nil
}
