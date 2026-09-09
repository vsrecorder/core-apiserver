package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type OpponentDeckUsageStatInterface interface {
	GetOpponentDeckUsageStat(
		ctx context.Context,
		userId string,
		// week は週(月曜始まり)内の任意日 YYYY-MM-DD。指定時は year_month / season より優先する
		week string,
		yearMonth string,
		environmentId string,
		season string,
		standardRegulationId string,
		regulationId uint,
		deckId string,
	) (*entity.OpponentDeckUsageStat, error)
}

type OpponentDeckUsageStat struct {
	opponentDeckUsageStatRepo    repository.OpponentDeckUsageStatInterface
	environmentRepo              repository.EnvironmentInterface
	officialEventEnvironmentRepo repository.OfficialEventEnvironmentInterface
	standardRegulationRepo       repository.StandardRegulationInterface
	championshipSeriesRepo       repository.ChampionshipSeriesInterface
}

func NewOpponentDeckUsageStat(
	opponentDeckUsageStatRepo repository.OpponentDeckUsageStatInterface,
	environmentRepo repository.EnvironmentInterface,
	officialEventEnvironmentRepo repository.OfficialEventEnvironmentInterface,
	standardRegulationRepo repository.StandardRegulationInterface,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
) OpponentDeckUsageStatInterface {
	return &OpponentDeckUsageStat{
		opponentDeckUsageStatRepo:    opponentDeckUsageStatRepo,
		environmentRepo:              environmentRepo,
		officialEventEnvironmentRepo: officialEventEnvironmentRepo,
		standardRegulationRepo:       standardRegulationRepo,
		championshipSeriesRepo:       championshipSeriesRepo,
	}
}

func (u *OpponentDeckUsageStat) GetOpponentDeckUsageStat(
	ctx context.Context,
	userId string,
	week string,
	yearMonth string,
	environmentId string,
	season string,
	standardRegulationId string,
	regulationId uint,
	deckId string,
) (*entity.OpponentDeckUsageStat, error) {
	var fromDate, toDate time.Time

	if week != "" {
		var err error
		fromDate, toDate, err = weekRange(week, timeNow().Local())
		if err != nil {
			logError(ctx, err)
			return nil, err
		}
	} else if yearMonth != "" {
		t, err := time.Parse("2006-01", yearMonth)
		if err != nil {
			logError(ctx, err)
			return nil, err
		}
		fromDate = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
		toDate = fromDate.AddDate(0, 1, 0)
	} else if season != "" {
		var err error
		fromDate, toDate, err = seasonRange(ctx, u.championshipSeriesRepo, season, time.Now().Local())
		if err != nil {
			logError(ctx, err)
			return nil, err
		}
	}

	if standardRegulationId != "" {
		reg, err := u.standardRegulationRepo.FindById(ctx, standardRegulationId)
		if err != nil {
			logError(ctx, err)
			return nil, err
		}

		regFrom := time.Date(reg.FromDate.Year(), reg.FromDate.Month(), reg.FromDate.Day(), 0, 0, 0, 0, time.Local)
		regTo := time.Date(reg.ToDate.Year(), reg.ToDate.Month(), reg.ToDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)

		if fromDate.IsZero() || regFrom.After(fromDate) {
			fromDate = regFrom
		}
		if toDate.IsZero() || regTo.Before(toDate) {
			toDate = regTo
		}
	}

	// 環境は期間だけでは表せない(大型大会は開催日が次の環境の期間に入っていても前の環境の
	// カードプールで行われる)ため、期間の交差と例外イベントをまとめて組み立てる。
	// standard_regulation より後に置いているのは、例外イベントを拾い直す範囲(BaseFrom/
	// BaseTo)を環境以外の条件だけで決める必要があるため(期間の交差自体は順序に依らない)。
	period, err := buildStatPeriod(ctx, u.environmentRepo, u.officialEventEnvironmentRepo, environmentId, fromDate, toDate)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	// yearMonth/season/environmentId/standard_regulation_idのいずれも未指定の場合は、
	// 期間をゼロ値のまま渡し「全期間」として扱う
	// （repository側はゼロ値の場合event_dateによる絞り込みを行わない）
	return u.opponentDeckUsageStatRepo.FindOpponentDeckUsageStat(ctx, userId, period, deckId, regulationId)
}
