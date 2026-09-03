package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type stubChampionsleagueScheduleRepository struct {
	schedules []*entity.ChampionsleagueSchedule
	schedule  *entity.ChampionsleagueSchedule
	err       error
	gotId     string
}

func (s *stubChampionsleagueScheduleRepository) Find(ctx context.Context) ([]*entity.ChampionsleagueSchedule, error) {
	return s.schedules, s.err
}

func (s *stubChampionsleagueScheduleRepository) FindById(ctx context.Context, id string) (*entity.ChampionsleagueSchedule, error) {
	s.gotId = id

	return s.schedule, s.err
}

func TestChampionsleagueScheduleUsecase(t *testing.T) {
	t.Run("正常系_Findは大会一覧をそのまま返す", func(t *testing.T) {
		schedules := []*entity.ChampionsleagueSchedule{{ID: "cl2027_yokohama"}}

		ret, err := NewChampionsleagueSchedule(&stubChampionsleagueScheduleRepository{schedules: schedules}).Find(context.Background())

		require.NoError(t, err)
		require.Equal(t, schedules, ret)
	})

	t.Run("正常系_FindByIdはIDをそのまま渡す", func(t *testing.T) {
		schedule := &entity.ChampionsleagueSchedule{ID: "cl2027_yokohama"}
		repository := &stubChampionsleagueScheduleRepository{schedule: schedule}

		ret, err := NewChampionsleagueSchedule(repository).FindById(context.Background(), "cl2027_yokohama")

		require.NoError(t, err)
		require.Equal(t, schedule, ret)
		require.Equal(t, "cl2027_yokohama", repository.gotId)
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		_, err := NewChampionsleagueSchedule(&stubChampionsleagueScheduleRepository{err: apperror.ErrRecordNotFound}).FindById(context.Background(), "unknown")

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
	})
}
