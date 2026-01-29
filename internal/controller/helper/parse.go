package helper

import (
	"encoding/base64"
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	DefaultLimit           = 10
	DefaultOffset          = 0
	DefaultOfficialEventId = 0
	DefaultTypeId          = 0
	DefaultLeagueType      = 0
	DefaultEventType       = ""
	DefaultArchived        = false

	DateLayout = time.DateOnly
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
	query := GetQueryCursor(ctx)

	if query == "" {
		return time.Time{}, nil
	}

	decodedQuery, err := base64.StdEncoding.DecodeString(query)
	if err != nil {
		return time.Time{}, err
	}

	return time.Parse(time.RFC3339, string(decodedQuery))
}

func ParseQueryDate(ctx *gin.Context) (date time.Time, err error) {
	query := GetQueryDate(ctx)

	if query == "" {
		date = time.Time{}
	} else {
		date, err = time.Parse(DateLayout, query)

		if err != nil {
			return time.Time{}, err
		}

		date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	}

	return date, nil
}

func ParseQueryFromDate(ctx *gin.Context) (fromDate time.Time, err error) {
	query := GetQueryFromDate(ctx)

	if query == "" {
		fromDate = time.Time{}
	} else {
		fromDate, err = time.Parse(DateLayout, query)

		if err != nil {
			return time.Time{}, err
		}

		fromDate = time.Date(fromDate.Year(), fromDate.Month(), fromDate.Day(), 0, 0, 0, 0, time.Local)
	}

	return fromDate, nil
}

func ParseQueryToDate(ctx *gin.Context) (toDate time.Time, err error) {
	query := GetQueryToDate(ctx)

	if query == "" {
		toDate = time.Time{}
	} else {
		toDate, err = time.Parse(DateLayout, query)

		if err != nil {
			return time.Time{}, err
		}

		toDate = time.Date(toDate.Year(), toDate.Month(), toDate.Day(), 0, 0, 0, 0, time.Local)
	}

	return toDate, nil
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

func ParseQueryOfficialEventId(ctx *gin.Context) (uint, error) {
	query := GetQueryOfficialEventId(ctx)

	if query == "" {
		return DefaultOfficialEventId, nil
	}

	officialEventId, err := strconv.Atoi(query)

	if err != nil { // 取得したクエリパラメータが数値か否か
		return DefaultOfficialEventId, err
	} else if officialEventId < 0 {
		return uint(officialEventId), errors.New("bad query parameter")
	}

	return uint(officialEventId), nil
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
	} else if typeId == 5 || typeId >= 8 { // 大型大会(typeId: 1) / シティ(typeId: 2) / トレリ(typeId: 3) / ジムイベント(typeId: 4) / オーガナイザーイベント(typeId: 6) / その他(typeId: 7)以外の場合
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

func ParseQueryEventType(ctx *gin.Context) (string, error) {
	query := GetQueryEventType(ctx)

	if query == "" {
		return DefaultEventType, nil
	}

	switch query {
	case "official":
		return "official", nil
	case "tonamel":
		return "tonamel", nil
	default:
		return DefaultEventType, nil
	}
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
