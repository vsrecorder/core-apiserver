package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// EnvironmentBadgeView は環境(entity.Environment)に、対象ユーザーの獲得状況を重ねた読み取り専用モデル。
type EnvironmentBadgeView struct {
	Environment *entity.Environment
	Achieved    bool
	AchievedAt  time.Time
}

type EnvironmentBadgeInterface interface {
	// GetByUserId は環境一覧に、指定ユーザーの獲得状況を重ねて返す。
	GetByUserId(
		ctx context.Context,
		userId string,
	) ([]*EnvironmentBadgeView, error)
}

type EnvironmentBadge struct {
	environmentRepo          repository.EnvironmentInterface
	userEnvironmentBadgeRepo repository.UserEnvironmentBadgeInterface
}

func NewEnvironmentBadge(
	environmentRepo repository.EnvironmentInterface,
	userEnvironmentBadgeRepo repository.UserEnvironmentBadgeInterface,
) EnvironmentBadgeInterface {
	return &EnvironmentBadge{
		environmentRepo:          environmentRepo,
		userEnvironmentBadgeRepo: userEnvironmentBadgeRepo,
	}
}

func (u *EnvironmentBadge) GetByUserId(
	ctx context.Context,
	userId string,
) ([]*EnvironmentBadgeView, error) {
	environments, err := u.environmentRepo.Find(ctx)
	if err != nil {
		return nil, err
	}

	userEnvironmentBadges, err := u.userEnvironmentBadgeRepo.FindByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	achievedMap := make(map[string]*entity.UserEnvironmentBadge, len(userEnvironmentBadges))
	for _, ub := range userEnvironmentBadges {
		achievedMap[ub.EnvironmentId] = ub
	}

	views := make([]*EnvironmentBadgeView, 0, len(environments))
	for _, env := range environments {
		view := &EnvironmentBadgeView{
			Environment: env,
		}
		if ub, ok := achievedMap[env.ID]; ok {
			view.Achieved = true
			view.AchievedAt = ub.AchievedAt
		}
		views = append(views, view)
	}

	return views, nil
}
