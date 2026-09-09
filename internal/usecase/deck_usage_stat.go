package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type DeckUsageStatInterface interface {
	GetDeckUsageStat(
		ctx context.Context,
		userId string,
		// week は週(月曜始まり)内の任意日 YYYY-MM-DD。指定時は year_month / season より優先する
		week string,
		yearMonth string,
		environmentId string,
		season string,
		standardRegulationId string,
		regulationId uint,
		allTime bool,
	) (*entity.DeckUsageStat, error)
}

type DeckUsageStat struct {
	deckUsageStatRepo            repository.DeckUsageStatInterface
	environmentRepo              repository.EnvironmentInterface
	officialEventEnvironmentRepo repository.OfficialEventEnvironmentInterface
	standardRegulationRepo       repository.StandardRegulationInterface
	championshipSeriesRepo       repository.ChampionshipSeriesInterface
}

func NewDeckUsageStat(
	deckUsageStatRepo repository.DeckUsageStatInterface,
	environmentRepo repository.EnvironmentInterface,
	officialEventEnvironmentRepo repository.OfficialEventEnvironmentInterface,
	standardRegulationRepo repository.StandardRegulationInterface,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
) DeckUsageStatInterface {
	return &DeckUsageStat{
		deckUsageStatRepo:            deckUsageStatRepo,
		environmentRepo:              environmentRepo,
		officialEventEnvironmentRepo: officialEventEnvironmentRepo,
		standardRegulationRepo:       standardRegulationRepo,
		championshipSeriesRepo:       championshipSeriesRepo,
	}
}

func (u *DeckUsageStat) GetDeckUsageStat(
	ctx context.Context,
	userId string,
	week string,
	yearMonth string,
	environmentId string,
	season string,
	standardRegulationId string,
	regulationId uint,
	allTime bool,
) (*entity.DeckUsageStat, error) {
	var fromDate, toDate time.Time

	// 全期間集計が指定された場合は期間条件を一切適用しない
	// （デッキ一覧カードのように期間セレクタを持たない画面向け）。
	if allTime {
		return u.deckUsageStatRepo.FindDeckUsageStat(ctx, userId, repository.StatPeriod{}, regulationId)
	}

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

		// レギュレーションの期間（to_dateは含む日付なので翌日0時をexclusive上限とする）
		regFrom := time.Date(reg.FromDate.Year(), reg.FromDate.Month(), reg.FromDate.Day(), 0, 0, 0, 0, time.Local)
		regTo := time.Date(reg.ToDate.Year(), reg.ToDate.Month(), reg.ToDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)

		// 他の条件とレギュレーションの両方が指定された場合は期間の交差を取る
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

	// いずれも未指定の場合は当月
	if period.From.IsZero() {
		now := time.Now().Local()
		period.From = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		period.To = period.From.AddDate(0, 1, 0)
		period.BaseFrom, period.BaseTo = period.From, period.To
	}

	return u.deckUsageStatRepo.FindDeckUsageStat(ctx, userId, period, regulationId)
}
