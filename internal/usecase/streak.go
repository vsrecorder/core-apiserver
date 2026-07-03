package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type StreakInterface interface {
	GetByUserId(
		ctx context.Context,
		userId string,
	) (*entity.UserStreak, error)
}

type Streak struct {
	repository repository.UserStreakInterface
}

func NewStreak(
	repository repository.UserStreakInterface,
) StreakInterface {
	return &Streak{repository}
}

func (u *Streak) GetByUserId(
	ctx context.Context,
	userId string,
) (*entity.UserStreak, error) {
	streak, err := u.repository.FindByUserId(ctx, userId)
	if err != nil {
		if errors.Is(err, apperror.ErrRecordNotFound) {
			// まだ一度も記録していないユーザーは0件のストリークとして返す
			return entity.NewUserStreak(userId, 0, 0, 0, time.Time{}, time.Time{}), nil
		}

		return nil, err
	}

	return streak, nil
}
