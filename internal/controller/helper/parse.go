package helper

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	DefaultLimit      = 10
	DefaultOffset     = 0
	DefaultTypeId     = 0
	DefaultLeagueType = 0
	DefaultArchived   = false

	DateLayout = "2006-01-02"
)

func ParseQueryLimit(ctx *gin.Context) (int, error) {
	query := GetQueryLimit(ctx)

	if query == "" {
		return DefaultLimit, nil
	}

	limit, err := strconv.Atoi(query)

	if err != nil { // 取得したクエリパラメータが数値か否か
		return -1, err
	} else if limit <= 0 {
		return DefaultLimit, nil
	}

	return limit, nil
}

func ParseQueryOffset(ctx *gin.Context) (int, error) {
	query := GetQueryOffset(ctx)

	if query == "" {
		return DefaultOffset, nil
	}

	offset, err := strconv.Atoi(query)

	if err != nil { // 取得したクエリパラメータが数値か否か
		return -1, err
	} else if offset <= 0 {
		return DefaultOffset, nil
	}

	return offset, nil
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
		startDate, err = time.Parse(DateLayout, query)

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
		endDate, err = time.Parse(DateLayout, query)

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
		return DefaultTypeId, nil
	}

	typeId, err := strconv.Atoi(query)

	if err != nil { // 取得したクエリパラメータが数値か否か
		return 0, err
	} else if typeId <= 0 {
		return uint(DefaultTypeId), nil
	} else if typeId == 5 || typeId >= 7 { // 大型大会(typeId: 1) / シティ(typeId: 2) / トレリ(typeId: 3) / ジムイベント(typeId: 4) / オーガナイザーイベント(typeId: 6) 以外の場合
		return uint(DefaultTypeId), errors.New("bad query parameter")
	}

	return uint(typeId), nil
}

func ParseQueryLeagueType(ctx *gin.Context) (uint, error) {
	query := GetQueryLeagueType(ctx)

	if query == "" {
		return DefaultLeagueType, nil
	}

	leagueType, err := strconv.Atoi(query)

	if err != nil { // 取得したクエリパラメータが数値か否か
		return 0, err
	} else if leagueType <= 0 {
		return uint(DefaultLeagueType), nil
	} else if leagueType >= 5 { // オープン(leagueType: 1) / ジュニア(leagueType: 2) / シニア(leagueType: 3) / マスター(leagueType: 4) 以外の場合
		return uint(DefaultLeagueType), errors.New("bad query parameter")
	}

	return uint(leagueType), nil
}

func ParseQueryArchive(ctx *gin.Context) (bool, error) {
	query := GetQueryArchived(ctx)

	if query == "" {
		return DefaultArchived, nil
	}

	ret, err := strconv.ParseBool(query)

	if err != nil {
		return DefaultArchived, err
	}

	return ret, nil
}
