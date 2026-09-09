package usecase

import (
	"context"
	"slices"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// statPeriodOf は環境で絞らない(期間だけの)統計の期間条件を作る。
// 週次レポートのように環境の指定が無い集計で使う。
func statPeriodOf(from time.Time, to time.Time) repository.StatPeriod {
	return repository.StatPeriod{
		From:     from,
		To:       to,
		BaseFrom: from,
		BaseTo:   to,
	}
}

// buildStatPeriod は統計の期間条件(repository.StatPeriod)を組み立てる。
//
// baseFrom / baseTo は環境以外の条件(week / year_month / season / standard_regulation)で
// 決まった期間。environmentId が空ならその期間をそのまま使う。
//
// environmentId が指定された場合は、環境の期間との交差を取ったうえで、開催日と実際の
// 対戦環境がズレる公式イベント(official_event_environments)の例外を持たせる。
//   - Include: 開催日が環境の期間外だが、この環境の記録として集計するイベント
//   - Exclude: 開催日が環境の期間内だが、別の環境として登録されているイベント
//
// Exclude を「この環境以外に登録された全イベント」にしているのは、期間外のイベントIDが
// 混ざっても期間条件で既に除外されるため実害が無いから。例外テーブルは極小なので全件を
// 1回引いて振り分ける。
func buildStatPeriod(
	ctx context.Context,
	environmentRepo repository.EnvironmentInterface,
	officialEventEnvironmentRepo repository.OfficialEventEnvironmentInterface,
	environmentId string,
	baseFrom time.Time,
	baseTo time.Time,
) (repository.StatPeriod, error) {
	period := statPeriodOf(baseFrom, baseTo)

	if environmentId == "" {
		return period, nil
	}

	env, err := environmentRepo.FindById(ctx, environmentId)
	if err != nil {
		logError(ctx, err)
		return repository.StatPeriod{}, err
	}

	// 環境の期間(to_dateは含む日付なので翌日0時をexclusive上限とする)との交差を取る。
	envFrom, envTo := environmentPeriod(env)
	if period.From.IsZero() || envFrom.After(period.From) {
		period.From = envFrom
	}
	if period.To.IsZero() || envTo.Before(period.To) {
		period.To = envTo
	}

	// 例外テーブルが引けない場合は期間だけで集計するのではなくエラーにする。
	// 黙って期間だけの結果を返すと、環境の集計が静かにズレたまま表示されるため。
	overrides, err := officialEventEnvironmentRepo.FindAll(ctx)
	if err != nil {
		logError(ctx, err)
		return repository.StatPeriod{}, err
	}

	// 該当が無いときは空スライスではなくnilのままにする(期間だけの条件と同じ値になり、
	// 呼び出し側の比較が素直になる)。
	var include, exclude []uint
	for officialEventId, id := range overrides {
		if id == environmentId {
			include = append(include, officialEventId)
		} else {
			exclude = append(exclude, officialEventId)
		}
	}

	// mapの反復順は不定なので、SQLのプレースホルダの並びを固定するために整列する。
	slices.Sort(include)
	slices.Sort(exclude)

	period.IncludeOfficialEventIds = include
	period.ExcludeOfficialEventIds = exclude

	return period, nil
}
