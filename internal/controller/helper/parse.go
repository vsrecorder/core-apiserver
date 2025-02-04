package helper

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	DEFAULT_LIMIT  = 10
	DEFAULT_OFFSET = 0
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
