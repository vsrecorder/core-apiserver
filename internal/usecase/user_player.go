package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type UserPlayerCreateParam struct {
	UserId   string
	PlayerId string
}

func NewUserPlayerCreateParam(
	userId string,
	playerId string,
) *UserPlayerCreateParam {
	return &UserPlayerCreateParam{
		UserId:   userId,
		PlayerId: playerId,
	}
}

type UserPlayerInterface interface {
	FindByUserId(
		ctx context.Context,
		userId string,
	) (*entity.UserPlayer, error)

	Create(
		ctx context.Context,
		param *UserPlayerCreateParam,
	) (*entity.UserPlayer, error)

	Verify(
		ctx context.Context,
		playerId string,
	) (*PlayerAccount, error)
}

type UserPlayer struct {
	repository         repository.UserPlayerInterface
	transactionManager repository.TransactionManager
}

func NewUserPlayer(
	repository repository.UserPlayerInterface,
	transactionManager repository.TransactionManager,
) UserPlayerInterface {
	return &UserPlayer{repository, transactionManager}
}

func (u *UserPlayer) FindByUserId(
	ctx context.Context,
	userId string,
) (*entity.UserPlayer, error) {
	userPlayer, err := u.repository.FindByUserId(ctx, userId)

	if err != nil {
		return nil, err
	}

	return userPlayer, nil
}

func (u *UserPlayer) Verify(
	ctx context.Context,
	playerId string,
) (*PlayerAccount, error) {
	return fetchPlayerAccount(playerId)
}

func (u *UserPlayer) Create(
	ctx context.Context,
	param *UserPlayerCreateParam,
) (*entity.UserPlayer, error) {
	existing, err := u.repository.FindByUserId(ctx, param.UserId)
	if err != nil && !errors.Is(err, apperror.ErrRecordNotFound) {
		return nil, err
	}

	// 現在の紐付けと同じ player_id が指定された場合は変更不要
	if existing != nil && existing.PlayerId == param.PlayerId {
		return existing, nil
	}

	now := time.Now().Local()

	// 既に有効な紐付けがあり、かつ紐付けから1ヶ月経過していない場合は変更不可
	if existing != nil && now.Before(existing.LockedUntil()) {
		return nil, apperror.ErrLocked
	}

	// player_id が既に別ユーザーの有効な紐付けに使われていないか確認
	inUse, err := u.repository.ExistsActiveByPlayerId(ctx, param.PlayerId)
	if err != nil {
		return nil, err
	}
	if inUse {
		return nil, apperror.ErrAlreadyExists
	}

	id, err := generateId()
	if err != nil {
		return nil, err
	}

	userPlayer := entity.NewUserPlayer(
		id,
		now,
		param.UserId,
		param.PlayerId,
	)

	err = u.transactionManager.Do(ctx, func(ctx context.Context) error {
		// 既存の紐付けがある場合(=1ヶ月経過後の変更)は旧レコードをsoft deleteしてから新規作成する
		if existing != nil {
			if err := u.repository.Delete(ctx, existing.ID); err != nil {
				return err
			}
		}

		return u.repository.Save(ctx, userPlayer)
	})
	if err != nil {
		return nil, err
	}

	return userPlayer, nil
}
