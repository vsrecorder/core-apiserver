package usecase

import (
	"context"
	"strings"
	"time"

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
		return time.Time{}, time.Time{}, err
	}

	return championshipSeriesDateRange(cs, now.Location())
}

// previousSeasonRange は season(空文字なら現在のシーズン)のひとつ前(championship_series上で
// from_dateが直前に終わる)シーズンの期間を返す。「前シーズンに引き続き」といった、シーズンを
// またいだ継続条件の判定に使う。
func previousSeasonRange(
	ctx context.Context,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
	season string,
	now time.Time,
) (fromDate time.Time, toDate time.Time, err error) {
	currentFromDate, _, err := seasonRange(ctx, championshipSeriesRepo, season, now)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	cs, err := championshipSeriesRepo.FindByDate(ctx, currentFromDate.AddDate(0, 0, -1))
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return championshipSeriesDateRange(cs, now.Location())
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
