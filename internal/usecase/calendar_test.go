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

// stubTonamelEventStore はTonamelイベントのDB参照(FindByIds)のスタブ。
// 問い合わせられたIDを記録する。events に登録したIDのみ結果として返し、
// errは FindByIds 全体のエラーとして返す。
type stubTonamelEventStore struct {
	mu           sync.Mutex
	queriedIds   []string
	events       map[string]*entity.TonamelEvent
	err          error
	savedByCalls []*entity.TonamelEvent
}

func (s *stubTonamelEventStore) FindByIds(ctx context.Context, ids []string) ([]*entity.TonamelEvent, error) {
	s.mu.Lock()
	s.queriedIds = append(s.queriedIds, ids...)
	s.mu.Unlock()

	if s.err != nil {
		return nil, s.err
	}

	ret := make([]*entity.TonamelEvent, 0, len(ids))
	for _, id := range ids {
		if e, ok := s.events[id]; ok {
			ret = append(ret, e)
		}
	}

	return ret, nil
}

func (s *stubTonamelEventStore) Save(ctx context.Context, tonamelEvent *entity.TonamelEvent) error {
	s.mu.Lock()
	s.savedByCalls = append(s.savedByCalls, tonamelEvent)
	s.mu.Unlock()
	return nil
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

	t.Run("正常系_記録が参照するTonamelイベントを重複なくまとめて取得して補完する", func(t *testing.T) {
		// "61ozP" は2つの記録から参照されているが、問い合わせは1回にまとめられる
		calendar := newCalendarWithTonamelRecords("61ozP", "61ozP", "OakZc")
		store := &stubTonamelEventStore{events: map[string]*entity.TonamelEvent{
			"61ozP": {ID: "61ozP"},
			"OakZc": {ID: "OakZc"},
		}}

		usecase := NewCalendar(logger, stubCalendarRepository{calendar: calendar}, store)

		ret, err := usecase.GetCalendar(context.Background(), userId)

		require.NoError(t, err)
		require.Len(t, ret.TonamelEvents, 2)

		gotIds := []string{ret.TonamelEvents[0].ID, ret.TonamelEvents[1].ID}
		require.ElementsMatch(t, []string{"61ozP", "OakZc"}, gotIds)
		// 重複を除いた一意なIDだけを1回の FindByIds で問い合わせる
		require.ElementsMatch(t, []string{"61ozP", "OakZc"}, store.queriedIds)
	})

	t.Run("正常系_Tonamelイベント参照がなければDB参照を行わない", func(t *testing.T) {
		calendar := newCalendarWithTonamelRecords("") // TonamelEventIdが空の記録のみ
		store := &stubTonamelEventStore{}

		usecase := NewCalendar(logger, stubCalendarRepository{calendar: calendar}, store)

		ret, err := usecase.GetCalendar(context.Background(), userId)

		require.NoError(t, err)
		require.Empty(t, ret.TonamelEvents)
		require.Empty(t, store.queriedIds)
	})

	t.Run("正常系_未保存のTonamelイベントは結果に含めず成功する", func(t *testing.T) {
		// "OakZc" はまだ tonamel_events に無い(バックフィル前・作成時取得失敗)ため返らない
		calendar := newCalendarWithTonamelRecords("61ozP", "OakZc")
		store := &stubTonamelEventStore{events: map[string]*entity.TonamelEvent{
			"61ozP": {ID: "61ozP"},
		}}

		usecase := NewCalendar(logger, stubCalendarRepository{calendar: calendar}, store)

		ret, err := usecase.GetCalendar(context.Background(), userId)

		require.NoError(t, err)
		require.Len(t, ret.TonamelEvents, 1)
		require.Equal(t, "61ozP", ret.TonamelEvents[0].ID)
	})

	t.Run("正常系_DB参照が失敗してもカレンダー自体は返す", func(t *testing.T) {
		calendar := newCalendarWithTonamelRecords("61ozP")
		store := &stubTonamelEventStore{err: errors.New("")}

		usecase := NewCalendar(logger, stubCalendarRepository{calendar: calendar}, store)

		ret, err := usecase.GetCalendar(context.Background(), userId)

		require.NoError(t, err)
		require.Empty(t, ret.TonamelEvents)
	})

	t.Run("異常系_カレンダー取得のエラーをそのまま返す", func(t *testing.T) {
		usecase := NewCalendar(logger, stubCalendarRepository{err: errors.New("")}, &stubTonamelEventStore{})

		ret, err := usecase.GetCalendar(context.Background(), userId)

		require.Error(t, err)
		require.Nil(t, ret)
	})
}
