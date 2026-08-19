package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// stubUnofficialEventRepository は自由形式イベントリポジトリのスタブ。
// mock_repositoryにUnofficialEvent用のモックが存在しないため手書きする。
type stubUnofficialEventRepository struct {
	findResult *entity.UnofficialEvent
	findErr    error
	saveErr    error
	deleteErr  error
	saved      *entity.UnofficialEvent
	deletedId  string
}

func (s *stubUnofficialEventRepository) FindById(ctx context.Context, id string) (*entity.UnofficialEvent, error) {
	return s.findResult, s.findErr
}

func (s *stubUnofficialEventRepository) Save(ctx context.Context, e *entity.UnofficialEvent) error {
	s.saved = e
	return s.saveErr
}

func (s *stubUnofficialEventRepository) Delete(ctx context.Context, id string) error {
	s.deletedId = id
	return s.deleteErr
}

func TestUnofficialEventUsecase(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("FindById", func(t *testing.T) {
		t.Run("正常系_指定IDの自由形式イベントを返す", func(t *testing.T) {
			event := &entity.UnofficialEvent{ID: "01HD7Y3K8D6FDHMHTZ2GT41TN2"}
			usecase := NewUnofficialEvent(&stubUnofficialEventRepository{findResult: event})

			ret, err := usecase.FindById(context.Background(), event.ID)

			require.NoError(t, err)
			require.Equal(t, event, ret)
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			usecase := NewUnofficialEvent(&stubUnofficialEventRepository{findErr: errors.New("")})

			ret, err := usecase.FindById(context.Background(), "01HD7Y3K8D6FDHMHTZ2GT41TN2")

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("Create", func(t *testing.T) {
		t.Run("正常系_IDを採番してイベントを保存する", func(t *testing.T) {
			repo := &stubUnofficialEventRepository{}
			usecase := NewUnofficialEvent(repo)

			date := time.Date(2026, 7, 18, 0, 0, 0, 0, time.Local)
			param := NewUnofficialEventParam(uid, "自主大会", date)

			ret, err := usecase.Create(context.Background(), param)

			require.NoError(t, err)
			require.NotEmpty(t, ret.ID)
			require.Equal(t, uid, ret.UserId)
			require.Equal(t, "自主大会", ret.Title)
			require.Equal(t, date, ret.Date)
			require.Equal(t, ret, repo.saved)
		})

		t.Run("異常系_保存失敗時はエラーを返す", func(t *testing.T) {
			usecase := NewUnofficialEvent(&stubUnofficialEventRepository{saveErr: errors.New("")})

			param := NewUnofficialEventParam(uid, "自主大会", time.Now().Local())

			ret, err := usecase.Create(context.Background(), param)

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("Update", func(t *testing.T) {
		id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"
		createdAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.Local)

		t.Run("正常系_イベント名と開催日を更新する", func(t *testing.T) {
			repo := &stubUnofficialEventRepository{
				findResult: &entity.UnofficialEvent{
					ID:        id,
					UserId:    uid,
					Title:     "自主大会",
					Date:      time.Date(2026, 7, 18, 0, 0, 0, 0, time.Local),
					CreatedAt: createdAt,
				},
			}
			usecase := NewUnofficialEvent(repo)

			date := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)
			param := NewUnofficialEventParam(uid, "身内対戦会", date)

			ret, err := usecase.Update(context.Background(), id, param)

			require.NoError(t, err)
			require.Equal(t, id, ret.ID)
			require.Equal(t, uid, ret.UserId)
			require.Equal(t, "身内対戦会", ret.Title)
			require.Equal(t, date, ret.Date)
			require.Equal(t, ret, repo.saved)
		})

		t.Run("正常系_作成日時は更新せず元の値を引き継ぐ", func(t *testing.T) {
			repo := &stubUnofficialEventRepository{
				findResult: &entity.UnofficialEvent{ID: id, UserId: uid, CreatedAt: createdAt},
			}
			usecase := NewUnofficialEvent(repo)

			param := NewUnofficialEventParam(uid, "身内対戦会", time.Now().Local())

			ret, err := usecase.Update(context.Background(), id, param)

			require.NoError(t, err)
			require.Equal(t, createdAt, ret.CreatedAt)
			require.Equal(t, createdAt, repo.saved.CreatedAt)
		})

		t.Run("異常系_存在しないIDはErrRecordNotFoundを返す", func(t *testing.T) {
			repo := &stubUnofficialEventRepository{findErr: apperror.ErrRecordNotFound}
			usecase := NewUnofficialEvent(repo)

			param := NewUnofficialEventParam(uid, "身内対戦会", time.Now().Local())

			ret, err := usecase.Update(context.Background(), id, param)

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Nil(t, ret)
			require.Nil(t, repo.saved)
		})

		t.Run("異常系_保存失敗時はエラーを返す", func(t *testing.T) {
			repo := &stubUnofficialEventRepository{
				findResult: &entity.UnofficialEvent{ID: id, UserId: uid},
				saveErr:    errors.New(""),
			}
			usecase := NewUnofficialEvent(repo)

			param := NewUnofficialEventParam(uid, "身内対戦会", time.Now().Local())

			ret, err := usecase.Update(context.Background(), id, param)

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

		t.Run("正常系_指定IDのイベントを削除する", func(t *testing.T) {
			repo := &stubUnofficialEventRepository{
				findResult: &entity.UnofficialEvent{ID: id, UserId: uid},
			}
			usecase := NewUnofficialEvent(repo)

			require.NoError(t, usecase.Delete(context.Background(), id))
			require.Equal(t, id, repo.deletedId)
		})

		t.Run("異常系_存在しないIDはErrRecordNotFoundを返し削除しない", func(t *testing.T) {
			repo := &stubUnofficialEventRepository{findErr: apperror.ErrRecordNotFound}
			usecase := NewUnofficialEvent(repo)

			err := usecase.Delete(context.Background(), id)

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Empty(t, repo.deletedId)
		})

		t.Run("異常系_削除失敗時はエラーを返す", func(t *testing.T) {
			repo := &stubUnofficialEventRepository{
				findResult: &entity.UnofficialEvent{ID: id, UserId: uid},
				deleteErr:  errors.New(""),
			}
			usecase := NewUnofficialEvent(repo)

			require.Error(t, usecase.Delete(context.Background(), id))
		})
	})
}
