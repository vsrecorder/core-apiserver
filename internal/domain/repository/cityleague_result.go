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

	// FindByPlayerId は playerId の入賞を、開催イベントの情報込みで新しい順に返す。
	// fromDate と toDate はシーズン期間の半開区間 [fromDate, toDate) を表し、
	// 共にゼロ値の場合は全期間を対象とする。入賞が無い場合は空スライスを返す
	// (「連携済みだが今シーズンはまだ入賞していない」は正常系のため、エラーにはしない)。
	FindByPlayerId(
		ctx context.Context,
		playerId string,
		fromDate time.Time,
		toDate time.Time,
	) ([]*entity.PlayerCityleagueResult, error)

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
