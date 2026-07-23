package usecase

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

// stubBadgeEvaluation は usecase パッケージ自身のテストで使う BadgeEvaluationInterface のスタブ。
// mock_usecase は usecase パッケージに依存しており、usecase パッケージ内のテストから
// 直接 import すると import cycle になるため、gomock ではなくこの手書きスタブを使う。
type stubBadgeEvaluation struct{}

func (stubBadgeEvaluation) EvaluateOnRecordCreated(
	ctx context.Context,
	userId string,
	record *entity.Record,
) ([]*entity.UserBadge, error) {
	return nil, nil
}

func (stubBadgeEvaluation) EvaluateOnMatchCreated(
	ctx context.Context,
	userId string,
	match *entity.Match,
) ([]*entity.UserBadge, error) {
	return nil, nil
}

func (stubBadgeEvaluation) EvaluateOnDeckCreated(
	ctx context.Context,
	userId string,
	deck *entity.Deck,
) ([]*entity.UserBadge, error) {
	return nil, nil
}

func (stubBadgeEvaluation) EvaluateOnDeckCodeCreated(
	ctx context.Context,
	userId string,
	deckCode *entity.DeckCode,
) {
}

func (stubBadgeEvaluation) EvaluateOnUserCreated(
	ctx context.Context,
	userId string,
	createdAt time.Time,
) ([]*entity.UserBadge, error) {
	return nil, nil
}

func (stubBadgeEvaluation) EvaluateOnRecordDeleted(
	ctx context.Context,
	userId string,
) error {
	return nil
}

// stubDesignationEvaluation は usecase パッケージ自身のテストで使う
// DesignationEvaluationInterface のスタブ(stubBadgeEvaluationと同じ理由でgomockを使わない)。
type stubDesignationEvaluation struct{}

func (stubDesignationEvaluation) CurrentTier(
	ctx context.Context,
	userId string,
) (int, error) {
	return 0, nil
}

func (stubDesignationEvaluation) TierAsOf(
	ctx context.Context,
	userId string,
	asOf time.Time,
) (int, error) {
	return 0, nil
}

func (stubDesignationEvaluation) NotifyIfTierChanged(
	ctx context.Context,
	userId string,
	beforeTier int,
	achievedAt time.Time,
) {
}

func (stubDesignationEvaluation) NotifyIfTierLost(
	ctx context.Context,
	userId string,
	beforeTier int,
) {
}

// stubEnvironmentBadgeEvaluation は usecase パッケージ自身のテストで使う
// EnvironmentBadgeEvaluationInterface のスタブ(stubBadgeEvaluationと同じ理由でgomockを使わない)。
type stubEnvironmentBadgeEvaluation struct{}

func (stubEnvironmentBadgeEvaluation) EvaluateOnMatchCreated(
	ctx context.Context,
	userId string,
	match *entity.Match,
	basisTime time.Time,
) (*entity.Environment, error) {
	return nil, nil
}

func (stubEnvironmentBadgeEvaluation) NotifyAchieved(
	ctx context.Context,
	userId string,
	env *entity.Environment,
	achievedAt time.Time,
	isRead bool,
) (string, error) {
	return "", nil
}

func (stubEnvironmentBadgeEvaluation) UpdateAchievedNotification(
	ctx context.Context,
	notificationId string,
	env *entity.Environment,
	achievedAt time.Time,
) error {
	return nil
}

// spyDesignationEvaluation は usecase パッケージ自身のテストで使う、
// NotifyIfTierChanged/NotifyIfTierLost の呼び出し有無だけを記録する手書きスタブ
// (stubDesignationEvaluationと同じ理由でgomockを使わない)。
type spyDesignationEvaluation struct {
	currentTier               int
	notifyIfTierChangedCalled *bool
	notifyIfTierLostCalled    *bool
}

func (s spyDesignationEvaluation) CurrentTier(ctx context.Context, userId string) (int, error) {
	return s.currentTier, nil
}

func (s spyDesignationEvaluation) TierAsOf(ctx context.Context, userId string, asOf time.Time) (int, error) {
	return s.currentTier, nil
}

func (s spyDesignationEvaluation) NotifyIfTierChanged(ctx context.Context, userId string, beforeTier int, achievedAt time.Time) {
	*s.notifyIfTierChangedCalled = true
}

func (s spyDesignationEvaluation) NotifyIfTierLost(ctx context.Context, userId string, beforeTier int) {
	*s.notifyIfTierLostCalled = true
}

// testLogger はテストで使う破棄用ロガー。
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// stubTonamelEventFetcher は tonamel.com からの取得(TonamelEventInterface)のスタブ。
// events に登録したIDだけ返し、それ以外は ErrRecordNotFound を返す。
// err が設定されていれば常にそれを返す。呼び出されたIDを calledIds に記録する。
type stubTonamelEventFetcher struct {
	events    map[string]*entity.TonamelEvent
	err       error
	calledIds []string
}

func (s *stubTonamelEventFetcher) FindById(ctx context.Context, id string) (*entity.TonamelEvent, error) {
	s.calledIds = append(s.calledIds, id)
	if s.err != nil {
		return nil, s.err
	}
	if e, ok := s.events[id]; ok {
		return e, nil
	}
	return nil, apperror.ErrRecordNotFound
}

// newRecordUsecaseForTest は Tonamel永続化を伴わない一般的なテスト用に Record usecase を組む。
func newRecordUsecaseForTest(
	repo *mock_repository.MockRecordInterface,
	badgeEval BadgeEvaluationInterface,
	designationEval DesignationEvaluationInterface,
) RecordInterface {
	return NewRecord(
		testLogger(),
		repo,
		badgeEval,
		designationEval,
		&stubTonamelEventFetcher{},
		&stubTonamelEventStore{},
	)
}

// デッキ未登録のまま作成した記録に、あとからデッキを登録するケースでは、
// Update経由で称号のtierが変化しうる。この変化がUpdateからも通知されることを保証する
// (Createのみ通知していた過去のバグの再発防止)。
func TestRecordUsecase_Update_NotifiesDesignationChange(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockRecordInterface(mockCtrl)

	var notifyIfTierChangedCalled, notifyIfTierLostCalled bool
	usecase := newRecordUsecaseForTest(
		mockRepository,
		stubBadgeEvaluation{},
		spyDesignationEvaluation{
			currentTier:               1,
			notifyIfTierChangedCalled: &notifyIfTierChangedCalled,
			notifyIfTierLostCalled:    &notifyIfTierLostCalled,
		},
	)

	id, err := generateId()
	require.NoError(t, err)

	deckId, err := generateId()
	require.NoError(t, err)

	record := entity.NewRecord(
		id, time.Now().Local(), 0, "", "", "", "", deckId, "", time.Time{}, false, false, "", "",
	)

	param := NewRecordParam(0, "", "", "", "", deckId, "", time.Time{}, false, false, "", "")

	mockRepository.EXPECT().FindById(context.Background(), id).Return(record, nil)
	mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

	_, err = usecase.Update(context.Background(), id, param)

	require.NoError(t, err)
	require.True(t, notifyIfTierChangedCalled)
	require.True(t, notifyIfTierLostCalled)
}

// Tonamel記録を作成すると、大会情報を1度だけ取得して tonamel_events へ保存する。
func TestRecordUsecase_Create_PersistsTonamelEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockRecordInterface(mockCtrl)

	t.Run("正常系_未保存のTonamel記録は取得して保存する", func(t *testing.T) {
		fetcher := &stubTonamelEventFetcher{events: map[string]*entity.TonamelEvent{
			"61ozP": entity.NewTonamelEvent("61ozP", "テスト大会", "説明", "https://example.com/i.png"),
		}}
		store := &stubTonamelEventStore{} // 事前に保存済みのものは無い

		usecase := NewRecord(testLogger(), mockRepository, stubBadgeEvaluation{}, stubDesignationEvaluation{}, fetcher, store)

		param := NewRecordParam(0, "61ozP", "", "", "user-1", "", "", time.Time{}, false, false, "", "")
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		_, err := usecase.Create(context.Background(), param)

		require.NoError(t, err)
		require.Equal(t, []string{"61ozP"}, fetcher.calledIds)
		require.Len(t, store.savedByCalls, 1)
		require.Equal(t, "61ozP", store.savedByCalls[0].ID)
		require.Equal(t, "テスト大会", store.savedByCalls[0].Title)
	})

	t.Run("正常系_既に保存済みのTonamel記録は取得しない", func(t *testing.T) {
		fetcher := &stubTonamelEventFetcher{}
		store := &stubTonamelEventStore{events: map[string]*entity.TonamelEvent{
			"61ozP": {ID: "61ozP"}, // 既に保存済み
		}}

		usecase := NewRecord(testLogger(), mockRepository, stubBadgeEvaluation{}, stubDesignationEvaluation{}, fetcher, store)

		param := NewRecordParam(0, "61ozP", "", "", "user-1", "", "", time.Time{}, false, false, "", "")
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		_, err := usecase.Create(context.Background(), param)

		require.NoError(t, err)
		require.Empty(t, fetcher.calledIds) // 外部取得しない
		require.Empty(t, store.savedByCalls)
	})

	t.Run("正常系_Tonamel記録でなければ取得も保存もしない", func(t *testing.T) {
		fetcher := &stubTonamelEventFetcher{}
		store := &stubTonamelEventStore{}

		usecase := NewRecord(testLogger(), mockRepository, stubBadgeEvaluation{}, stubDesignationEvaluation{}, fetcher, store)

		param := NewRecordParam(0, "", "", "", "user-1", "", "", time.Time{}, false, false, "", "")
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		_, err := usecase.Create(context.Background(), param)

		require.NoError(t, err)
		require.Empty(t, store.queriedIds)
		require.Empty(t, fetcher.calledIds)
		require.Empty(t, store.savedByCalls)
	})

	t.Run("正常系_大会情報の取得に失敗しても記録作成は成功する", func(t *testing.T) {
		fetcher := &stubTonamelEventFetcher{err: errors.New("")} // tonamel.com取得に失敗
		store := &stubTonamelEventStore{}

		usecase := NewRecord(testLogger(), mockRepository, stubBadgeEvaluation{}, stubDesignationEvaluation{}, fetcher, store)

		param := NewRecordParam(0, "61ozP", "", "", "user-1", "", "", time.Time{}, false, false, "", "")
		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Create(context.Background(), param)

		require.NoError(t, err) // 記録作成自体は失敗させない
		require.NotNil(t, ret)
		require.Empty(t, store.savedByCalls) // 取得失敗のため保存されない
	})
}

func TestRecordUsecase(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockRecordInterface(mockCtrl)
	usecase := newRecordUsecaseForTest(mockRepository, stubBadgeEvaluation{}, stubDesignationEvaluation{})

	for scenario, fn := range map[string]func(
		t *testing.T,
		mockRepository *mock_repository.MockRecordInterface,
		usecase RecordInterface,
	){
		"Find":                  test_RecordUsecase_Find,
		"FindOnCursor":          test_RecordUsecase_FindOnCursor,
		"FindById":              test_RecordUsecase_FindById,
		"FindByUserId":          test_RecordUsecase_FindByUserId,
		"FindByUserIdOnCursor":  test_RecordUsecase_FindByUserIdOnCursor,
		"FindByOfficialEventId": test_RecordUsecase_FindByOfficialEventId,
		"FindByTonamelEventId":  test_RecordUsecase_FindByTonamelEventId,
		"FindByDeckId":          test_RecordUsecase_FindByDeckId,
		"FindByDeckIdOnCursor":  test_RecordUsecase_FindByDeckIdOnCursor,
		"Create":                test_RecordUsecase_Create,
		"Update":                test_RecordUsecase_Update,
		"Delete":                test_RecordUsecase_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t, mockRepository, usecase)
		})
	}
}

func test_RecordUsecase_Find(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_記録一覧をそのまま返す", func(t *testing.T) {
		limit := 10
		offset := 0
		eventType := ""

		id, err := generateId()
		require.NoError(t, err)

		record := &entity.Record{
			ID: id,
		}

		records := []*entity.Record{
			record,
		}

		mockRepository.EXPECT().Find(context.Background(), limit, offset, eventType).Return(records, nil)

		ret, err := usecase.Find(context.Background(), limit, offset, eventType)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
	})

	t.Run("正常系_該当なしの場合は空スライスを返す", func(t *testing.T) {
		limit := 10
		offset := 0
		eventType := ""

		records := []*entity.Record{}

		mockRepository.EXPECT().Find(context.Background(), limit, offset, eventType).Return(records, nil)

		ret, err := usecase.Find(context.Background(), limit, offset, eventType)

		require.NoError(t, err)
		require.Equal(t, len(records), len(ret))
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		limit := 10
		offset := 0
		eventType := ""

		mockRepository.EXPECT().Find(context.Background(), limit, offset, eventType).Return(nil, errors.New(""))

		ret, err := usecase.Find(context.Background(), limit, offset, eventType)

		require.Error(t, err)
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_FindOnCursor(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_カーソル以降の記録一覧を返す", func(t *testing.T) {
		limit := 10
		cursorEventDate := time.Now().Local()
		cursorCreatedAt := time.Now().Local()
		eventType := ""

		id, err := generateId()
		require.NoError(t, err)

		record := &entity.Record{
			ID: id,
		}

		records := []*entity.Record{
			record,
		}

		mockRepository.EXPECT().FindOnCursor(context.Background(), limit, cursorEventDate, cursorCreatedAt, eventType).Return(records, nil)

		ret, err := usecase.FindOnCursor(context.Background(), limit, cursorEventDate, cursorCreatedAt, eventType)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
	})

	t.Run("正常系_該当なしの場合は空スライスを返す", func(t *testing.T) {
		limit := 10
		cursorEventDate := time.Now().Local()
		cursorCreatedAt := time.Now().Local()
		eventType := ""

		records := []*entity.Record{}

		mockRepository.EXPECT().FindOnCursor(context.Background(), limit, cursorEventDate, cursorCreatedAt, eventType).Return(records, nil)

		ret, err := usecase.FindOnCursor(context.Background(), limit, cursorEventDate, cursorCreatedAt, eventType)

		require.NoError(t, err)
		require.Equal(t, len(records), len(ret))
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		limit := 10
		cursorEventDate := time.Now().Local()
		cursorCreatedAt := time.Now().Local()
		eventType := ""

		mockRepository.EXPECT().FindOnCursor(context.Background(), limit, cursorEventDate, cursorCreatedAt, eventType).Return(nil, errors.New(""))

		ret, err := usecase.FindOnCursor(context.Background(), limit, cursorEventDate, cursorCreatedAt, eventType)

		require.Error(t, err)
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_FindById(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_指定IDの記録を返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		record := &entity.Record{
			ID: id,
		}

		mockRepository.EXPECT().FindById(context.Background(), id).Return(record, nil)

		ret, err := usecase.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, id, ret.ID)
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

		ret, err := usecase.FindById(context.Background(), id)

		require.Error(t, err)
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_FindByUserId(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_指定ユーザの記録一覧を返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		limit := 10
		offset := 0
		eventType := ""

		record := &entity.Record{
			ID:     id,
			UserId: uid,
		}

		records := []*entity.Record{
			record,
		}

		mockRepository.EXPECT().FindByUserId(context.Background(), uid, limit, offset, eventType).Return(records, nil)

		ret, err := usecase.FindByUserId(context.Background(), uid, limit, offset, eventType)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, uid, ret[0].UserId)
	})

	t.Run("正常系_該当なしの場合は空スライスを返す", func(t *testing.T) {
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		limit := 10
		offset := 0
		eventType := ""

		records := []*entity.Record{}

		mockRepository.EXPECT().FindByUserId(context.Background(), uid, limit, offset, eventType).Return(records, nil)

		ret, err := usecase.FindByUserId(context.Background(), uid, limit, offset, eventType)

		require.NoError(t, err)
		require.Equal(t, len(records), len(ret))
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		limit := 10
		offset := 0
		eventType := ""

		mockRepository.EXPECT().FindByUserId(context.Background(), uid, limit, offset, eventType).Return(nil, errors.New(""))

		ret, err := usecase.FindByUserId(context.Background(), uid, limit, offset, eventType)

		require.Error(t, err)
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_FindByUserIdOnCursor(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_指定ユーザのカーソル以降の記録一覧を返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		limit := 10
		cursorEventDate := time.Now().Local()
		cursorCreatedAt := time.Now().Local()
		eventType := ""

		record := &entity.Record{
			ID:     id,
			UserId: uid,
		}

		records := []*entity.Record{
			record,
		}

		mockRepository.EXPECT().FindByUserIdOnCursor(context.Background(), uid, limit, cursorEventDate, cursorCreatedAt, eventType).Return(records, nil)

		ret, err := usecase.FindByUserIdOnCursor(context.Background(), uid, limit, cursorEventDate, cursorCreatedAt, eventType)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, uid, ret[0].UserId)
	})

	t.Run("正常系_該当なしの場合は空スライスを返す", func(t *testing.T) {
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		limit := 10
		cursorEventDate := time.Now().Local()
		cursorCreatedAt := time.Now().Local()
		eventType := ""

		records := []*entity.Record{}

		mockRepository.EXPECT().FindByUserIdOnCursor(context.Background(), uid, limit, cursorEventDate, cursorCreatedAt, eventType).Return(records, nil)

		ret, err := usecase.FindByUserIdOnCursor(context.Background(), uid, limit, cursorEventDate, cursorCreatedAt, eventType)

		require.NoError(t, err)
		require.Equal(t, len(records), len(ret))
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
		limit := 10
		cursorEventDate := time.Now().Local()
		cursorCreatedAt := time.Now().Local()
		eventType := ""

		mockRepository.EXPECT().FindByUserIdOnCursor(context.Background(), uid, limit, cursorEventDate, cursorCreatedAt, eventType).Return(nil, errors.New(""))

		ret, err := usecase.FindByUserIdOnCursor(context.Background(), uid, limit, cursorEventDate, cursorCreatedAt, eventType)

		require.Error(t, err)
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_FindByOfficialEventId(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_指定公式イベントの記録一覧を返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		officialEventId := uint(10000)
		limit := 10
		offset := 0

		record := &entity.Record{
			ID:              id,
			OfficialEventId: officialEventId,
		}

		records := []*entity.Record{
			record,
		}

		mockRepository.EXPECT().FindByOfficialEventId(context.Background(), officialEventId, limit, offset).Return(records, nil)

		ret, err := usecase.FindByOfficialEventId(context.Background(), officialEventId, limit, offset)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, officialEventId, ret[0].OfficialEventId)
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		officialEventId := uint(10000)
		limit := 10
		offset := 0

		mockRepository.EXPECT().FindByOfficialEventId(context.Background(), officialEventId, limit, offset).Return(nil, errors.New(""))

		ret, err := usecase.FindByOfficialEventId(context.Background(), officialEventId, limit, offset)

		require.Error(t, err)
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_FindByTonamelEventId(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_指定Tonamelイベントの記録一覧を返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		tonamelEventId := "61ozP"
		limit := 10
		offset := 0

		record := &entity.Record{
			ID:             id,
			TonamelEventId: tonamelEventId,
		}

		records := []*entity.Record{
			record,
		}

		mockRepository.EXPECT().FindByTonamelEventId(context.Background(), tonamelEventId, limit, offset).Return(records, nil)

		ret, err := usecase.FindByTonamelEventId(context.Background(), tonamelEventId, limit, offset)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, tonamelEventId, ret[0].TonamelEventId)
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		tonamelEventId := "61ozP"
		limit := 10
		offset := 0

		mockRepository.EXPECT().FindByTonamelEventId(context.Background(), tonamelEventId, limit, offset).Return(nil, errors.New(""))

		ret, err := usecase.FindByTonamelEventId(context.Background(), tonamelEventId, limit, offset)

		require.Error(t, err)
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_FindByDeckId(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_指定デッキの記録一覧を返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		deckId, err := generateId()
		require.NoError(t, err)

		limit := 10
		offset := 0
		eventType := ""

		record := &entity.Record{
			ID:     id,
			DeckId: deckId,
		}

		records := []*entity.Record{
			record,
		}

		mockRepository.EXPECT().FindByDeckId(context.Background(), deckId, limit, offset, eventType).Return(records, nil)

		ret, err := usecase.FindByDeckId(context.Background(), deckId, limit, offset, eventType)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, deckId, ret[0].DeckId)
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		deckId, err := generateId()
		require.NoError(t, err)

		limit := 10
		offset := 0
		eventType := ""

		mockRepository.EXPECT().FindByDeckId(context.Background(), deckId, limit, offset, eventType).Return(nil, errors.New(""))

		ret, err := usecase.FindByDeckId(context.Background(), deckId, limit, offset, eventType)

		require.Error(t, err)
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_FindByDeckIdOnCursor(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_指定デッキのカーソル以降の記録一覧を返す", func(t *testing.T) {
		id, err := generateId()
		require.NoError(t, err)

		deckId, err := generateId()
		require.NoError(t, err)

		limit := 10
		cursorEventDate := time.Now().Local()
		cursorCreatedAt := time.Now().Local()
		eventType := ""

		record := &entity.Record{
			ID:     id,
			DeckId: deckId,
		}

		records := []*entity.Record{
			record,
		}

		mockRepository.EXPECT().FindByDeckIdOnCursor(context.Background(), deckId, limit, cursorEventDate, cursorCreatedAt, eventType).Return(records, nil)

		ret, err := usecase.FindByDeckIdOnCursor(context.Background(), deckId, limit, cursorEventDate, cursorCreatedAt, eventType)

		require.NoError(t, err)
		require.Equal(t, id, ret[0].ID)
		require.Equal(t, deckId, ret[0].DeckId)
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		deckId, err := generateId()
		require.NoError(t, err)

		limit := 10
		cursorEventDate := time.Now().Local()
		cursorCreatedAt := time.Now().Local()
		eventType := ""

		mockRepository.EXPECT().FindByDeckIdOnCursor(context.Background(), deckId, limit, cursorEventDate, cursorCreatedAt, eventType).Return(nil, errors.New(""))

		ret, err := usecase.FindByDeckIdOnCursor(context.Background(), deckId, limit, cursorEventDate, cursorCreatedAt, eventType)

		require.Error(t, err)
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_Create(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_IDと作成日時を採番して保存する", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()

		record := entity.NewRecord(
			id,
			createdAt,
			0,
			"",
			"",
			"",
			"",
			"",
			"",
			time.Time{},
			false,
			false,
			"",
			"",
		)

		param := NewRecordParam(
			0,
			"",
			"",
			"",
			"",
			"",
			"",
			time.Time{},
			false,
			false,
			"",
			"",
		)

		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		ret, err := usecase.Create(context.Background(), param)

		require.NoError(t, err)
		require.NotEmpty(t, ret.ID)
		require.NotEmpty(t, ret.CreatedAt)
		require.Equal(t, record.OfficialEventId, ret.OfficialEventId)
		require.Equal(t, record.PrivateFlg, ret.PrivateFlg)
	})

	t.Run("異常系_保存失敗時はエラーを返す", func(t *testing.T) {
		param := NewRecordParam(
			0,
			"",
			"",
			"",
			"",
			"",
			"",
			time.Time{},
			false,
			false,
			"",
			"",
		)

		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(errors.New(""))

		ret, err := usecase.Create(context.Background(), param)

		require.Error(t, err)
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_Update(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_取得した記録にパラメータを反映して保存する", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()

		record := entity.NewRecord(
			id,
			createdAt,
			0,
			"",
			"",
			"",
			"",
			"",
			"",
			time.Time{},
			false,
			false,
			"",
			"",
		)

		param := NewRecordParam(
			0,
			"",
			"",
			"",
			"",
			"",
			"",
			time.Time{},
			false,
			false,
			"",
			"",
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(record, nil)
		mockRepository.EXPECT().Save(context.Background(), record).Return(nil)

		ret, err := usecase.Update(context.Background(), id, param)

		require.NoError(t, err)
		require.Equal(t, ret, record)
	})

	t.Run("異常系_存在しないIDはErrRecordNotFoundを返す", func(t *testing.T) {
		id, _ := generateId()

		param := NewRecordParam(
			0,
			"",
			"",
			"",
			"",
			"",
			"",
			time.Time{},
			false,
			false,
			"",
			"",
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		ret, err := usecase.Update(context.Background(), id, param)

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		require.Empty(t, ret)
	})

	t.Run("異常系_保存失敗時はエラーを返す", func(t *testing.T) {
		id, _ := generateId()
		createdAt := time.Now().Local()

		record := entity.NewRecord(
			id,
			createdAt,
			0,
			"",
			"",
			"",
			"",
			"",
			"",
			time.Time{},
			false,
			false,
			"",
			"",
		)

		param := NewRecordParam(
			0,
			"",
			"",
			"",
			"",
			"",
			"",
			time.Time{},
			false,
			false,
			"",
			"",
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(record, nil)
		mockRepository.EXPECT().Save(context.Background(), record).Return(errors.New(""))

		ret, err := usecase.Update(context.Background(), id, param)

		require.Error(t, err)
		require.Empty(t, ret)
	})
}

func test_RecordUsecase_Delete(t *testing.T, mockRepository *mock_repository.MockRecordInterface, usecase RecordInterface) {
	t.Run("正常系_記録を取得してから削除する", func(t *testing.T) {
		id, _ := generateId()

		record := entity.NewRecord(
			id,
			time.Now().Local(),
			0,
			"",
			"",
			"",
			"user-1",
			"",
			"",
			time.Time{},
			false,
			false,
			"",
			"",
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(record, nil)
		mockRepository.EXPECT().Delete(context.Background(), id).Return(nil)

		err := usecase.Delete(context.Background(), id)

		require.NoError(t, err)
	})

	t.Run("異常系_FindByIdが失敗する場合", func(t *testing.T) {
		id, _ := generateId()

		mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

		err := usecase.Delete(context.Background(), id)

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
	})

	t.Run("異常系_Deleteが失敗する場合", func(t *testing.T) {
		id, _ := generateId()

		record := entity.NewRecord(
			id,
			time.Now().Local(),
			0,
			"",
			"",
			"",
			"user-1",
			"",
			"",
			time.Time{},
			false,
			false,
			"",
			"",
		)

		mockRepository.EXPECT().FindById(context.Background(), id).Return(record, nil)
		mockRepository.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

		err := usecase.Delete(context.Background(), id)

		require.Error(t, err)
	})
}
