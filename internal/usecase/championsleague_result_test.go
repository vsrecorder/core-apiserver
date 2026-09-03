package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// stubChampionsleagueResultRepository は大型大会結果リポジトリのスタブ。
// mock_repository のモックは usecase を import できないため(import cycle)手書きする。
type stubChampionsleagueResultRepository struct {
	events            []*entity.ChampionsleagueResultEvent
	results           []*entity.ChampionsleagueResult
	err               error
	gotScheduleId     string
	gotLeagueType     uint
	findEventsCalled  bool
	findByScheduleHit bool
}

func (s *stubChampionsleagueResultRepository) FindEvents(ctx context.Context) ([]*entity.ChampionsleagueResultEvent, error) {
	s.findEventsCalled = true

	return s.events, s.err
}

func (s *stubChampionsleagueResultRepository) FindByChampionsleagueScheduleId(ctx context.Context, leagueType uint, championsleagueScheduleId string) ([]*entity.ChampionsleagueResult, error) {
	s.findByScheduleHit = true
	s.gotScheduleId = championsleagueScheduleId
	s.gotLeagueType = leagueType

	return s.results, s.err
}

func TestChampionsleagueResultUsecase(t *testing.T) {
	t.Run("正常系_FindEventsはイベント一覧をそのまま返す", func(t *testing.T) {
		events := []*entity.ChampionsleagueResultEvent{{ChampionsleagueScheduleId: "pjcs2026"}}
		repository := &stubChampionsleagueResultRepository{events: events}

		ret, err := NewChampionsleagueResult(repository).FindEvents(context.Background())

		require.NoError(t, err)
		require.Equal(t, events, ret)
		require.True(t, repository.findEventsCalled)
	})

	t.Run("正常系_FindByChampionsleagueScheduleIdは大会IDとリーグ区分をそのまま渡す", func(t *testing.T) {
		results := []*entity.ChampionsleagueResult{{ChampionsleagueScheduleId: "pjcs2026"}}
		repository := &stubChampionsleagueResultRepository{results: results}

		ret, err := NewChampionsleagueResult(repository).FindByChampionsleagueScheduleId(context.Background(), 4, "pjcs2026")

		require.NoError(t, err)
		require.Equal(t, results, ret)
		require.Equal(t, "pjcs2026", repository.gotScheduleId)
		require.Equal(t, uint(4), repository.gotLeagueType)
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		expected := errors.New("unexpected error")
		repository := &stubChampionsleagueResultRepository{err: expected}

		_, err := NewChampionsleagueResult(repository).FindByChampionsleagueScheduleId(context.Background(), 0, "pjcs2026")

		require.ErrorIs(t, err, expected)
		require.True(t, repository.findByScheduleHit)
	})
}
