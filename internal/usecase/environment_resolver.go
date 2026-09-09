package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// ResolveEnvironmentForOfficialEvent は公式イベントに紐づく対戦の環境を返す。
//
// 環境は通常、対戦の基準日時(basisTime。RecordBasisTime参照)が属する期間から引く。
// ただし大型大会(チャンピオンズリーグ・PJCS)は開催日時点の最新弾がカードプールに
// 入らないことがあり、日付から引くと実際に対戦した環境とズレる。そのため
// official_event_environments に例外登録があればそちらを優先する。
//
// officialEventId が0(公式イベントに紐づかない記録)の場合や例外登録が無い場合は、
// 従来どおり basisTime から引く(EnvironmentInterface.FindByDate と同じ判定)。
//
// 環境の判定を必要とする箇所(環境バッジ・統計・イベント表示)はこの関数か
// EnvironmentScopeFor を通し、日付だけで環境を決めないこと。
func ResolveEnvironmentForOfficialEvent(
	ctx context.Context,
	environmentRepo repository.EnvironmentInterface,
	officialEventEnvironmentRepo repository.OfficialEventEnvironmentInterface,
	officialEventId uint,
	basisTime time.Time,
) (*entity.Environment, error) {
	if officialEventId != 0 && officialEventEnvironmentRepo != nil {
		environmentId, err := officialEventEnvironmentRepo.FindEnvironmentIdByOfficialEventId(ctx, officialEventId)

		switch {
		case err == nil:
			return environmentRepo.FindById(ctx, environmentId)
		case errors.Is(err, apperror.ErrRecordNotFound):
			// 例外登録が無いのが通常の状態。開催日からの判定にフォールバックする。
		default:
			logError(ctx, err)
			return nil, err
		}
	}

	return environmentRepo.FindByDate(ctx, basisTime)
}
