package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type ChampionsleagueResultInterface interface {
	// FindEvents は結果が登録されているイベントを、入賞者を含めずに全件返す。
	// 大型大会は全期間でも数十イベントしかないため、期間やリーグ区分では絞らない。
	FindEvents(
		ctx context.Context,
	) ([]*entity.ChampionsleagueResultEvent, error)

	// FindByChampionsleagueScheduleId は1大会の結果を、イベント単位にまとめて返す。
	// leagueType が 0 の場合は全リーグ区分を対象とする。
	// 結果が1件も無い場合は apperror.ErrRecordNotFound を返す。
	FindByChampionsleagueScheduleId(
		ctx context.Context,
		leagueType uint,
		championsleagueScheduleId string,
	) ([]*entity.ChampionsleagueResult, error)
}
