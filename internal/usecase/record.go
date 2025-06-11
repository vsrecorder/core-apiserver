package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/gorm"
)

type RecordParam struct {
	officialEventId uint
	tonamelEventId  string
	friendId        string
	userId          string
	deckId          string
	privateFlg      bool
	tcgMeisterURL   string
	memo            string
}

func NewRecordParam(
	officialEventId uint,
	tonamelEventId string,
	friendId string,
	userId string,
	deckId string,
	privateFlg bool,
	tcgMeisterURL string,
	memo string,
) *RecordParam {
	return &RecordParam{
		officialEventId: officialEventId,
		tonamelEventId:  tonamelEventId,
		friendId:        friendId,
		userId:          userId,
		deckId:          deckId,
		privateFlg:      privateFlg,
		tcgMeisterURL:   tcgMeisterURL,
		memo:            memo,
	}
}

type RecordInterface interface {
	Find(
		ctx context.Context,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	FindOnCursor(
		ctx context.Context,
		limit int,
		cursor time.Time,
	) ([]*entity.Record, error)

	FindById(
		ctx context.Context,
		id string,
	) (*entity.Record, error)

	FindByUserId(
		ctx context.Context,
		uid string,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	FindByUserIdOnCursor(
		ctx context.Context,
		uid string,
		limit int,
		cursor time.Time,
	) ([]*entity.Record, error)

	FindByOfficialEventId(
		ctx context.Context,
		officialEventId uint,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	FindByTonamelEventId(
		ctx context.Context,
		tonamelEventId string,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	FindByDeckId(
		ctx context.Context,
		deckId string,
		limit int,
		offset int,
	) ([]*entity.Record, error)

	Create(
		ctx context.Context,
		param *RecordParam,
	) (*entity.Record, error)

	Update(
		ctx context.Context,
		id string,
		param *RecordParam,
	) (*entity.Record, error)

	Delete(
		ctx context.Context,
		id string,
	) error
}

type Record struct {
	repository repository.RecordInterface
}

func NewRecord(
	repository repository.RecordInterface,
) RecordInterface {
	return &Record{repository}
}

func (u *Record) Find(
	ctx context.Context,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	records, err := u.repository.Find(ctx, limit, offset)

	if err != nil {
		return nil, err
	}

	return records, nil
}

func (u *Record) FindOnCursor(
	ctx context.Context,
	limit int,
	cursor time.Time,
) ([]*entity.Record, error) {
	records, err := u.repository.FindOnCursor(ctx, limit, cursor)

	if err != nil {
		return nil, err
	}

	return records, nil
}

func (u *Record) FindById(
	ctx context.Context,
	id string,
) (*entity.Record, error) {
	record, err := u.repository.FindById(ctx, id)

	if err != nil {
		return nil, err
	}

	return record, nil
}

func (u *Record) FindByUserId(
	ctx context.Context,
	uid string,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByUserId(ctx, uid, limit, offset)

	if err != nil {
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByUserIdOnCursor(
	ctx context.Context,
	uid string,
	limit int,
	cursor time.Time,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByUserIdOnCursor(ctx, uid, limit, cursor)

	if err != nil {
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByOfficialEventId(
	ctx context.Context,
	officialEventId uint,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByOfficialEventId(ctx, officialEventId, limit, offset)

	if err != nil {
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByTonamelEventId(
	ctx context.Context,
	tonamelEventId string,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByTonamelEventId(ctx, tonamelEventId, limit, offset)

	if err != nil {
		return nil, err
	}

	return records, nil
}

func (u *Record) FindByDeckId(
	ctx context.Context,
	deckId string,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	records, err := u.repository.FindByDeckId(ctx, deckId, limit, offset)

	if err != nil {
		return nil, err
	}

	return records, nil
}

func (u *Record) Create(
	ctx context.Context,
	param *RecordParam,
) (*entity.Record, error) {
	id, err := generateId()
	if err != nil {
		return nil, err
	}

	createdAt := time.Now().UTC().Truncate(0)

	record := entity.NewRecord(
		id,
		createdAt,
		param.officialEventId,
		param.tonamelEventId,
		param.friendId,
		param.userId,
		param.deckId,
		param.privateFlg,
		param.tcgMeisterURL,
		param.memo,
	)

	if err := u.repository.Save(ctx, record); err != nil {
		return nil, err
	}

	return record, nil
}

func (u *Record) Update(
	ctx context.Context,
	id string,
	param *RecordParam,
) (*entity.Record, error) {
	// 指定されたidのRecordが存在するか確認
	ret, err := u.repository.FindById(ctx, id)
	if err == gorm.ErrRecordNotFound {
		return nil, err
	} else if err != nil {
		return nil, err
	}

	record := entity.NewRecord(
		id,
		ret.CreatedAt,
		param.officialEventId,
		param.tonamelEventId,
		param.friendId,
		param.userId,
		param.deckId,
		param.privateFlg,
		param.tcgMeisterURL,
		param.memo,
	)

	if err := u.repository.Save(ctx, record); err != nil {
		return nil, err
	}

	return record, nil
}

func (u *Record) Delete(
	ctx context.Context,
	id string,
) error {
	err := u.repository.Delete(ctx, id)

	if err != nil {
		return err
	}

	return nil
}
