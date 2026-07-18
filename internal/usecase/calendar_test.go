package usecase

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// stubCalendarRepository はカレンダー取得のスタブ。
// mock_repositoryにCalendar用のモックが存在しないため手書きする。
type stubCalendarRepository struct {
	calendar *entity.Calendar
	err      error
}

func (s stubCalendarRepository) FindByUserId(ctx context.Context, userId string) (*entity.Calendar, error) {
	return s.calendar, s.err
}

// stubTonamelEventRepository はTonamelイベント取得のスタブ。
// 呼び出されたIDを記録し、errIdsに含まれるIDにはエラーを返す。
type stubTonamelEventRepository struct {
	mu        sync.Mutex
	calledIds []string
	errIds    map[string]struct{}
}

func (s *stubTonamelEventRepository) FindById(ctx context.Context, id string) (*entity.TonamelEvent, error) {
	s.mu.Lock()
	s.calledIds = append(s.calledIds, id)
	s.mu.Unlock()

	if _, ok := s.errIds[id]; ok {
		return nil, errors.New("")
	}

	return &entity.TonamelEvent{ID: id}, nil
}

func newCalendarWithTonamelRecords(tonamelEventIds ...string) *entity.Calendar {
	records := make([]*entity.CalendarRecord, 0, len(tonamelEventIds))
	for _, id := range tonamelEventIds {
		records = append(records, entity.NewCalendarRecord(&entity.Record{TonamelEventId: id}, nil))
	}

	return &entity.Calendar{Records: records}
}

func TestCalendarUsecase_GetCalendar(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	userId := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_記録が参照するTonamelイベントを重複なく取得して補完する", func(t *testing.T) {
		// "61ozP" は2つの記録から参照されているが、取得は1回にまとめられる
		calendar := newCalendarWithTonamelRecords("61ozP", "61ozP", "OakZc")
		tonamelRepo := &stubTonamelEventRepository{}

		usecase := NewCalendar(logger, stubCalendarRepository{calendar: calendar}, tonamelRepo)

		ret, err := usecase.GetCalendar(context.Background(), userId)

		require.NoError(t, err)
		require.Len(t, ret.TonamelEvents, 2)

		gotIds := []string{ret.TonamelEvents[0].ID, ret.TonamelEvents[1].ID}
		require.ElementsMatch(t, []string{"61ozP", "OakZc"}, gotIds)
		require.ElementsMatch(t, []string{"61ozP", "OakZc"}, tonamelRepo.calledIds)
	})

	t.Run("正常系_Tonamelイベント参照がなければ外部取得は行わない", func(t *testing.T) {
		calendar := newCalendarWithTonamelRecords("") // TonamelEventIdが空の記録のみ
		tonamelRepo := &stubTonamelEventRepository{}

		usecase := NewCalendar(logger, stubCalendarRepository{calendar: calendar}, tonamelRepo)

		ret, err := usecase.GetCalendar(context.Background(), userId)

		require.NoError(t, err)
		require.Empty(t, ret.TonamelEvents)
		require.Empty(t, tonamelRepo.calledIds)
	})

	t.Run("正常系_一部のTonamelイベント取得に失敗してもスキップして成功する", func(t *testing.T) {
		calendar := newCalendarWithTonamelRecords("61ozP", "OakZc")
		tonamelRepo := &stubTonamelEventRepository{errIds: map[string]struct{}{"OakZc": {}}}

		usecase := NewCalendar(logger, stubCalendarRepository{calendar: calendar}, tonamelRepo)

		ret, err := usecase.GetCalendar(context.Background(), userId)

		require.NoError(t, err)
		require.Len(t, ret.TonamelEvents, 1)
		require.Equal(t, "61ozP", ret.TonamelEvents[0].ID)
	})

	t.Run("異常系_カレンダー取得のエラーをそのまま返す", func(t *testing.T) {
		usecase := NewCalendar(logger, stubCalendarRepository{err: errors.New("")}, &stubTonamelEventRepository{})

		ret, err := usecase.GetCalendar(context.Background(), userId)

		require.Error(t, err)
		require.Nil(t, ret)
	})
}
