package usecase

import (
	"context"
	"log/slog"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type CalendarInterface interface {
	GetCalendar(
		ctx context.Context,
		userId string,
	) (*entity.Calendar, error)
}

type Calendar struct {
	logger            *slog.Logger
	calendarRepo      repository.CalendarInterface
	tonamelEventStore repository.TonamelEventStoreInterface
}

func NewCalendar(
	logger *slog.Logger,
	calendarRepo repository.CalendarInterface,
	tonamelEventStore repository.TonamelEventStoreInterface,
) CalendarInterface {
	return &Calendar{
		logger:            logger,
		calendarRepo:      calendarRepo,
		tonamelEventStore: tonamelEventStore,
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

// fetchTonamelEvents は記録から参照されているTonamelイベントをDBからまとめて取得する。
//
// 大会情報は記録作成時に tonamel_events へ保存済みのため、ここでは外部サイトを引かず
// 1クエリでまとめて取得する(以前は大会ごとに tonamel.com を引いており、記録数に比例して
// 外部リクエストが増えるN+1になっていた)。既存記録ぶんは cmd/backfill-tonamel-events で
// 投入する。まだ保存されていない大会(バックフィル前・作成時取得失敗)は結果に含まれず、
// 呼び出し側はタイトル不明として扱う(以前の取得失敗時と同じ寛容な挙動)。
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

	tonamelEvents, err := u.tonamelEventStore.FindByIds(ctx, ids)
	if err != nil {
		u.logger.Warn(
			"failed to fetch tonamel events for calendar",
			slog.String("error_message", err.Error()),
		)

		return nil
	}

	return tonamelEvents
}
