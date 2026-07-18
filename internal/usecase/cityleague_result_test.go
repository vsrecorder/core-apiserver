package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// stubCityleagueResultRepository はシティリーグ結果リポジトリのスタブ。
// mock_repositoryにCityleagueResult用のモックが存在しないため手書きする。
type stubCityleagueResultRepository struct {
	events  []*entity.CityleagueResultEvent
	result  *entity.CityleagueResult
	results []*entity.CityleagueResult
	err     error
}

func (s stubCityleagueResultRepository) FindEvents(ctx context.Context, leagueType uint, fromDate time.Time, toDate time.Time) ([]*entity.CityleagueResultEvent, error) {
	return s.events, s.err
}

func (s stubCityleagueResultRepository) FindByOfficialEventId(ctx context.Context, officialEventId uint) (*entity.CityleagueResult, error) {
	return s.result, s.err
}

func (s stubCityleagueResultRepository) FindByCityleagueScheduleId(ctx context.Context, leagueType uint, cityleagueScheduleId string) ([]*entity.CityleagueResult, error) {
	return s.results, s.err
}

func (s stubCityleagueResultRepository) FindByDate(ctx context.Context, leagueType uint, date time.Time) ([]*entity.CityleagueResult, error) {
	return s.results, s.err
}

func (s stubCityleagueResultRepository) FindByTerm(ctx context.Context, leagueType uint, fromDate time.Time, toDate time.Time) ([]*entity.CityleagueResult, error) {
	return s.results, s.err
}

func TestCityleagueResultUsecase(t *testing.T) {
	now := time.Now().Local()

	t.Run("正常系_FindEventsはイベント一覧をそのまま返す", func(t *testing.T) {
		events := []*entity.CityleagueResultEvent{{}}
		usecase := NewCityleagueResult(stubCityleagueResultRepository{events: events})

		ret, err := usecase.FindEvents(context.Background(), 4, now, now)

		require.NoError(t, err)
		require.Equal(t, events, ret)
	})

	t.Run("正常系_FindByOfficialEventIdは指定イベントの結果を返す", func(t *testing.T) {
		result := &entity.CityleagueResult{}
		usecase := NewCityleagueResult(stubCityleagueResultRepository{result: result})

		ret, err := usecase.FindByOfficialEventId(context.Background(), 606466)

		require.NoError(t, err)
		require.Equal(t, result, ret)
	})

	t.Run("正常系_FindByCityleagueScheduleIdは日程内の結果一覧を返す", func(t *testing.T) {
		results := []*entity.CityleagueResult{{}}
		usecase := NewCityleagueResult(stubCityleagueResultRepository{results: results})

		ret, err := usecase.FindByCityleagueScheduleId(context.Background(), 4, "2026_s1")

		require.NoError(t, err)
		require.Equal(t, results, ret)
	})

	t.Run("正常系_FindByDateは指定日の結果一覧を返す", func(t *testing.T) {
		results := []*entity.CityleagueResult{{}}
		usecase := NewCityleagueResult(stubCityleagueResultRepository{results: results})

		ret, err := usecase.FindByDate(context.Background(), 4, now)

		require.NoError(t, err)
		require.Equal(t, results, ret)
	})

	t.Run("正常系_FindByTermは指定期間の結果一覧を返す", func(t *testing.T) {
		results := []*entity.CityleagueResult{{}}
		usecase := NewCityleagueResult(stubCityleagueResultRepository{results: results})

		ret, err := usecase.FindByTerm(context.Background(), 4, now, now)

		require.NoError(t, err)
		require.Equal(t, results, ret)
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		usecase := NewCityleagueResult(stubCityleagueResultRepository{err: errors.New("")})

		_, err := usecase.FindEvents(context.Background(), 4, now, now)
		require.Error(t, err)

		_, err = usecase.FindByOfficialEventId(context.Background(), 606466)
		require.Error(t, err)

		_, err = usecase.FindByCityleagueScheduleId(context.Background(), 4, "2026_s1")
		require.Error(t, err)

		_, err = usecase.FindByDate(context.Background(), 4, now)
		require.Error(t, err)

		_, err = usecase.FindByTerm(context.Background(), 4, now, now)
		require.Error(t, err)
	})
}
