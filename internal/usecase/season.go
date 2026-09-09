package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// championshipSeriesIdPrefix は championship_series.id の接頭辞。season 識別子
// (championship_series.id から接頭辞を除いた文字列。例:"2026")との相互変換に使う。
const championshipSeriesIdPrefix = "series_"

// CurrentSeasonLabel は championship_series テーブルを参照し、now が属するシーズンの
// 識別子(championship_series.id から championshipSeriesIdPrefix を除いた文字列、
// 例:"2026")を返す。該当するシーズンが championship_series に存在しない場合はエラーを返す。
func CurrentSeasonLabel(
	ctx context.Context,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
	now time.Time,
) (string, error) {
	cs, err := championshipSeriesRepo.FindByDate(ctx, now)
	if err != nil {
		logError(ctx, err)
		return "", err
	}

	return strings.TrimPrefix(cs.ID, championshipSeriesIdPrefix), nil
}

// seasonRange は season(championship_series.id から championshipSeriesIdPrefix を除いた
// 識別子。空文字なら now が属する現在のシーズン)を、championship_series テーブルの
// from_date〜to_date の期間に変換する(toDate は翌日0時のexclusive上限)。
func seasonRange(
	ctx context.Context,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
	season string,
	now time.Time,
) (fromDate time.Time, toDate time.Time, err error) {
	var cs *entity.ChampionshipSeries

	if season == "" {
		cs, err = championshipSeriesRepo.FindByDate(ctx, now)
	} else {
		cs, err = championshipSeriesRepo.FindById(ctx, championshipSeriesIdPrefix+season)
	}
	if err != nil {
		logError(ctx, err)
		return time.Time{}, time.Time{}, err
	}

	return championshipSeriesDateRange(cs, now.Location())
}

// CurrentSeasonDateRange は seasonRange(season="")と同じ結果(nowが属する現在のシーズンの
// from_date〜to_date)を返す、package外(cmd/配下のバッチ等)向けのエクスポート版。
func CurrentSeasonDateRange(
	ctx context.Context,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
	now time.Time,
) (fromDate time.Time, toDate time.Time, err error) {
	return seasonRange(ctx, championshipSeriesRepo, "", now)
}

// StatPeriodFor は environmentId・season・regulationId の指定から統計の期間条件を決定する、
// package外(cmd/配下のバッチ等)向けのエクスポート版。DeckUsageStat.GetDeckUsageStat 等と
// 同じ考え方で、複数指定された場合は期間の交差(intersection)を取る。
// いずれも空文字ならゼロ値(期間の絞り込みなし)を返す。
//
// 環境は期間だけでは表せない(大型大会は開催日が次の環境の期間に入っていても前の環境の
// カードプールで行われる)ため、戻り値は期間ではなく例外イベントを含む StatPeriod。
// 環境を絞り込みに使う集計は、期間だけを見ずにこの StatPeriod をそのまま
// infrastructure へ渡すこと。
func StatPeriodFor(
	ctx context.Context,
	environmentRepo repository.EnvironmentInterface,
	officialEventEnvironmentRepo repository.OfficialEventEnvironmentInterface,
	standardRegulationRepo repository.StandardRegulationInterface,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
	environmentId string,
	season string,
	regulationId string,
	now time.Time,
) (repository.StatPeriod, error) {
	var fromDate, toDate time.Time

	if season != "" {
		var err error
		fromDate, toDate, err = seasonRange(ctx, championshipSeriesRepo, season, now)
		if err != nil {
			logError(ctx, err)
			return repository.StatPeriod{}, err
		}
	}

	if regulationId != "" {
		reg, err := standardRegulationRepo.FindById(ctx, regulationId)
		if err != nil {
			logError(ctx, err)
			return repository.StatPeriod{}, err
		}

		// レギュレーションの期間(to_dateは含む日付なので翌日0時をexclusive上限とする)
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

	// 環境は最後に適用する。例外イベントを拾い直す範囲(BaseFrom/BaseTo)を環境以外の条件
	// だけで決める必要があるため(期間の交差自体は順序に依らない)。
	return buildStatPeriod(ctx, environmentRepo, officialEventEnvironmentRepo, environmentId, fromDate, toDate)
}

// previousSeasonRange は season(空文字なら現在のシーズン)のひとつ前(championship_series上で
// from_dateが直前に終わる)シーズンの期間を返す。「前シーズンに引き続き」といった、シーズンを
// またいだ継続条件の判定に使う。
//
// 最古のシーズン(championship_seriesの先頭行)を指定した場合、そのひとつ前のシーズンは
// テーブルに存在しない。これは異常ではなく「前シーズンの実績が0件」として扱うべき正常系なので、
// エラーではなく exists=false を返す(呼び出し側で0件として扱う)。
func previousSeasonRange(
	ctx context.Context,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
	season string,
	now time.Time,
) (fromDate time.Time, toDate time.Time, exists bool, err error) {
	currentFromDate, _, err := seasonRange(ctx, championshipSeriesRepo, season, now)
	if err != nil {
		logError(ctx, err)
		return time.Time{}, time.Time{}, false, err
	}

	cs, err := championshipSeriesRepo.FindByDate(ctx, currentFromDate.AddDate(0, 0, -1))
	if err != nil {
		logError(ctx, err)
		if errors.Is(err, apperror.ErrRecordNotFound) {
			return time.Time{}, time.Time{}, false, nil
		}

		return time.Time{}, time.Time{}, false, err
	}

	fromDate, toDate, err = championshipSeriesDateRange(cs, now.Location())
	if err != nil {
		logError(ctx, err)
		return time.Time{}, time.Time{}, false, err
	}

	return fromDate, toDate, true, nil
}

// championshipSeriesDateRange は championship_series の1行を、from_date(0時始まり)〜
// to_date翌日0時(exclusive上限)の期間に変換する。
func championshipSeriesDateRange(
	cs *entity.ChampionshipSeries,
	loc *time.Location,
) (fromDate time.Time, toDate time.Time, err error) {
	fromDate = time.Date(cs.FromDate.Year(), cs.FromDate.Month(), cs.FromDate.Day(), 0, 0, 0, 0, loc)
	toDate = time.Date(cs.ToDate.Year(), cs.ToDate.Month(), cs.ToDate.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)

	return fromDate, toDate, nil
}
