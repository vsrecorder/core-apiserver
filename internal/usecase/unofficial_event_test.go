package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// stubUnofficialEventRepository は自由形式イベントリポジトリのスタブ。
// mock_repositoryにUnofficialEvent用のモックが存在しないため手書きする。
type stubUnofficialEventRepository struct {
	findResult *entity.UnofficialEvent
	findErr    error
	saveErr    error
	saved      *entity.UnofficialEvent
}

func (s *stubUnofficialEventRepository) FindById(ctx context.Context, id string) (*entity.UnofficialEvent, error) {
	return s.findResult, s.findErr
}

func (s *stubUnofficialEventRepository) Save(ctx context.Context, e *entity.UnofficialEvent) error {
	s.saved = e
	return s.saveErr
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
}
