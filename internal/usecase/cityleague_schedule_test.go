package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// stubCityleagueScheduleRepository はシティリーグ日程リポジトリのスタブ。
// mock_repositoryにCityleagueSchedule用のモックが存在しないため手書きする。
type stubCityleagueScheduleRepository struct {
	schedules []*entity.CityleagueSchedule
	schedule  *entity.CityleagueSchedule
	err       error
}

func (s stubCityleagueScheduleRepository) Find(ctx context.Context) ([]*entity.CityleagueSchedule, error) {
	return s.schedules, s.err
}

func (s stubCityleagueScheduleRepository) FindById(ctx context.Context, id string) (*entity.CityleagueSchedule, error) {
	return s.schedule, s.err
}

func (s stubCityleagueScheduleRepository) FindByDate(ctx context.Context, date time.Time) (*entity.CityleagueSchedule, error) {
	return s.schedule, s.err
}

func TestCityleagueScheduleUsecase(t *testing.T) {
	schedule := &entity.CityleagueSchedule{ID: "2026_s1"}

	t.Run("正常系_Findは日程一覧をそのまま返す", func(t *testing.T) {
		usecase := NewCityleagueSchedule(stubCityleagueScheduleRepository{schedules: []*entity.CityleagueSchedule{schedule}})

		ret, err := usecase.Find(context.Background())

		require.NoError(t, err)
		require.Len(t, ret, 1)
		require.Equal(t, schedule, ret[0])
	})

	t.Run("正常系_FindByIdは指定IDの日程を返す", func(t *testing.T) {
		usecase := NewCityleagueSchedule(stubCityleagueScheduleRepository{schedule: schedule})

		ret, err := usecase.FindById(context.Background(), "2026_s1")

		require.NoError(t, err)
		require.Equal(t, schedule, ret)
	})

	t.Run("正常系_FindByDateは指定日の日程を返す", func(t *testing.T) {
		usecase := NewCityleagueSchedule(stubCityleagueScheduleRepository{schedule: schedule})

		ret, err := usecase.FindByDate(context.Background(), time.Now().Local())

		require.NoError(t, err)
		require.Equal(t, schedule, ret)
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		usecase := NewCityleagueSchedule(stubCityleagueScheduleRepository{err: errors.New("")})

		_, err := usecase.Find(context.Background())
		require.Error(t, err)

		_, err = usecase.FindById(context.Background(), "2026_s1")
		require.Error(t, err)

		_, err = usecase.FindByDate(context.Background(), time.Now().Local())
		require.Error(t, err)
	})
}
