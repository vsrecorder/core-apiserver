package helper

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	DEFAULT_LIMIT       = 10
	DEFAULT_OFFSET      = 0
	DEFAULT_TYPE_ID     = 0
	DEFAULT_LEAGUE_TYPE = 0
	DATE_LAYOUT         = "2006-01-02"
)

func ParseQueryLimit(ctx *gin.Context) (int, error) {
	query := GetQueryLimit(ctx)

	if query == "" {
		return DEFAULT_LIMIT, nil
	} else {
		limit, err := strconv.Atoi(query)

		if err != nil { // 取得したクエリパラメータが数値か否か
			return -1, err
		} else if limit <= 0 {
			return DEFAULT_LIMIT, nil
		}

		return limit, nil
	}
}

func ParseQueryOffset(ctx *gin.Context) (int, error) {
	query := GetQueryOffset(ctx)

	if query == "" {
		return DEFAULT_OFFSET, nil
	} else {
		offset, err := strconv.Atoi(query)

		if err != nil { // 取得したクエリパラメータが数値か否か
			return -1, err
		} else if offset <= 0 {
			return DEFAULT_OFFSET, nil
		}

		return offset, nil
	}
}

func ParseQueryCursor(ctx *gin.Context) (time.Time, error) {
	return time.Time{}, nil
}

func ParseQueryStartDate(ctx *gin.Context) (startDate time.Time, err error) {
	query := GetQueryStartDate(ctx)

	if query == "" {
		now := time.Now()
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	} else {
		startDate, err = time.Parse(DATE_LAYOUT, query)

		if err != nil {
			return time.Time{}, err
		}

		startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.Local)
	}

	return startDate, nil
}

func ParseQueryEndDate(ctx *gin.Context) (endDate time.Time, err error) {
	query := GetQueryEndDate(ctx)

	if query == "" {
		now := time.Now()
		endDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	} else {
		endDate, err = time.Parse(DATE_LAYOUT, query)

		if err != nil {
			return time.Time{}, err
		}

		endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, time.Local)
	}

	return endDate, nil
}

func ParseQueryTypeId(ctx *gin.Context) (uint, error) {
	query := GetQueryTypeId(ctx)

	if query == "" {
		return DEFAULT_TYPE_ID, nil
	} else {
		typeId, err := strconv.Atoi(query)

		if err != nil { // 取得したクエリパラメータが数値か否か
			return 0, err
		} else if typeId <= 0 {
			return uint(DEFAULT_TYPE_ID), nil
		} else if typeId == 5 || typeId >= 7 { // 大型大会(typeId: 1) / シティ(typeId: 2) / トレリ(typeId: 3) / ジムイベント(typeId: 4) / オーガナイザーイベント(typeId: 6) 以外の場合
			return uint(DEFAULT_TYPE_ID), errors.New("bad query parameter")
		}

		return uint(typeId), nil
	}
}

func ParseQueryLeagueType(ctx *gin.Context) (uint, error) {
	query := GetQueryLeagueType(ctx)

	if query == "" {
		return DEFAULT_LEAGUE_TYPE, nil
	} else {
		leagueType, err := strconv.Atoi(query)

		if err != nil { // 取得したクエリパラメータが数値か否か
			return 0, err
		} else if leagueType <= 0 {
			return uint(DEFAULT_LEAGUE_TYPE), nil
		} else if leagueType >= 5 { // オープン(leagueType: 1) / ジュニア(leagueType: 2) / シニア(leagueType: 3) / マスター(leagueType: 4) 以外の場合
			return uint(DEFAULT_LEAGUE_TYPE), errors.New("bad query parameter")
		}

		return uint(leagueType), nil
	}
}
