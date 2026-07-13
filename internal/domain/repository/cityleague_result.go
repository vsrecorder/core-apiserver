package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type CityleagueResultInterface interface {
	// FindEvents は結果が登録されているイベントを、入賞者を含めずに返す。
	// leagueType が 0 の場合は全リーグ、fromDate と toDate が共にゼロ値の場合は全期間を対象とする。
	FindEvents(
		ctx context.Context,
		leagueType uint,
		fromDate time.Time,
		toDate time.Time,
	) ([]*entity.CityleagueResultEvent, error)

	FindByOfficialEventId(
		ctx context.Context,
		officialEventId uint,
	) (*entity.CityleagueResult, error)

	FindByCityleagueScheduleId(
		ctx context.Context,
		leagueType uint,
		cityleagueIdScheduleId string,
	) ([]*entity.CityleagueResult, error)

	FindByDate(
		ctx context.Context,
		leagueType uint,
		date time.Time,
	) ([]*entity.CityleagueResult, error)

	FindByTerm(
		ctx context.Context,
		leagueType uint,
		fromDate time.Time,
		toDate time.Time,
	) ([]*entity.CityleagueResult, error)
}
