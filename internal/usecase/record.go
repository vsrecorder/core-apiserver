package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/gorm"
)

type RecordInterface interface {
	Find(
		ctx context.Context,
		limit int,
		offset int,
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
		record *entity.Record,
	) error

	Update(
		ctx context.Context,
		id string,
		record *entity.Record,
	) error

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
) *Record {
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
	record *entity.Record,
) error {
	err := u.repository.Save(ctx, record)

	if err != nil {
		return err
	}

	return nil
}

func (u *Record) Update(
	ctx context.Context,
	id string,
	record *entity.Record,
) error {
	ret, err := u.repository.FindById(ctx, id)
	if err == gorm.ErrRecordNotFound {
		return err
	}

	record.CreatedAt = ret.CreatedAt

	if err := u.repository.Save(ctx, record); err != nil {
		return err
	}

	return nil
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
