package usecase

import (
	"context"
	"log/slog"
	"sync"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// Tonamelイベントの同時取得数の上限。
// 外部サイト(tonamel.com)への取得のため、一度に投げすぎないよう抑える。
const tonamelEventFetchConcurrency = 4

type CalendarInterface interface {
	GetCalendar(
		ctx context.Context,
		userId string,
	) (*entity.Calendar, error)
}

type Calendar struct {
	logger           *slog.Logger
	calendarRepo     repository.CalendarInterface
	tonamelEventRepo repository.TonamelEventInterface
}

func NewCalendar(
	logger *slog.Logger,
	calendarRepo repository.CalendarInterface,
	tonamelEventRepo repository.TonamelEventInterface,
) CalendarInterface {
	return &Calendar{
		logger:           logger,
		calendarRepo:     calendarRepo,
		tonamelEventRepo: tonamelEventRepo,
	}
}

func (u *Calendar) GetCalendar(
	ctx context.Context,
	userId string,
) (*entity.Calendar, error) {
	calendar, err := u.calendarRepo.FindByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	calendar.TonamelEvents = u.fetchTonamelEvents(ctx, calendar.Records)

	return calendar, nil
}

// fetchTonamelEvents は記録から参照されているTonamelイベントを外部から取得する。
//
// TonamelイベントはDBではなく外部サイトから取得するため、取得できないことがある。
// 1件でも失敗したらカレンダー全体を失敗させるのは割に合わないので、
// 失敗したものは結果に含めずスキップする(呼び出し側はタイトル不明として扱う)。
func (u *Calendar) fetchTonamelEvents(
	ctx context.Context,
	records []*entity.CalendarRecord,
) []*entity.TonamelEvent {
	idSet := make(map[string]struct{})
	for _, record := range records {
		if record.Record.TonamelEventId != "" {
			idSet[record.Record.TonamelEventId] = struct{}{}
		}
	}

	if len(idSet) == 0 {
		return nil
	}

	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	var (
		mu            sync.Mutex
		wg            sync.WaitGroup
		tonamelEvents []*entity.TonamelEvent
	)

	semaphore := make(chan struct{}, tonamelEventFetchConcurrency)

	for _, id := range ids {
		wg.Add(1)

		go func(id string) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			tonamelEvent, err := u.tonamelEventRepo.FindById(ctx, id)
			if err != nil {
				u.logger.Warn(
					"failed to fetch tonamel event for calendar",
					slog.String("tonamel_event_id", id),
					slog.String("error_message", err.Error()),
				)

				return
			}

			mu.Lock()
			tonamelEvents = append(tonamelEvents, tonamelEvent)
			mu.Unlock()
		}(id)
	}

	wg.Wait()

	return tonamelEvents
}
