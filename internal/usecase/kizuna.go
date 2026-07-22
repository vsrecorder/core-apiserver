package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type KizunaInterface interface {
	GetKizuna(
		ctx context.Context,
		userId string,
	) (*entity.Kizuna, error)
}

type Kizuna struct {
	kizunaRepo repository.KizunaInterface
}

func NewKizuna(
	kizunaRepo repository.KizunaInterface,
) KizunaInterface {
	return &Kizuna{kizunaRepo}
}

/*
 * GetKizuna はユーザーの全デッキのきずなLv.を返す。
 *
 * 期間で絞る口は用意していない。きずなは「これまでどう歩んできたか」であり、
 * 今月だけのきずな、という概念が無いため（勝率を期間で切る deck_usage_stat とは
 * この点が違う）。
 */
func (u *Kizuna) GetKizuna(
	ctx context.Context,
	userId string,
) (*entity.Kizuna, error) {
	aggregates, err := u.kizunaRepo.FindKizunaDeckAggregates(ctx, userId)
	if err != nil {
		return nil, err
	}

	return entity.NewKizuna(userId, entity.CalculateKizuna(aggregates)), nil
}
