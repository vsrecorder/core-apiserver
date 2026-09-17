package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

// 大会情報は保存済み(tonamel_events)を先に引き、無いときだけ tonamel.com へ取りに行って保存する。
// 未認証で叩ける経路なので、既知のIDで毎回外部サイトへ取りに行かないことを保証する。
func TestTonamelEventUsecase(t *testing.T) {
	id := "61ozP"

	t.Run("FindById", func(t *testing.T) {
		t.Run("正常系_保存済みなら外部サイトへ取りに行かずそれを返す", func(t *testing.T) {
			mockRepository := mock_repository.NewMockTonamelEventInterface(gomock.NewController(t))
			store := &stubTonamelEventStore{events: map[string]*entity.TonamelEvent{
				id: entity.NewTonamelEvent(id, "保存済みの大会", "説明", "https://example.com/i.png"),
			}}
			// repository.FindById は EXPECT しない(呼ばれない)
			usecase := NewTonamelEvent(mockRepository, store)

			ret, err := usecase.FindById(context.Background(), id)

			require.NoError(t, err)
			require.Equal(t, "保存済みの大会", ret.Title)
			require.Empty(t, store.savedByCalls)
		})

		t.Run("正常系_未保存なら外部サイトから取得して保存する", func(t *testing.T) {
			mockRepository := mock_repository.NewMockTonamelEventInterface(gomock.NewController(t))
			store := &stubTonamelEventStore{}
			usecase := NewTonamelEvent(mockRepository, store)

			fetched := entity.NewTonamelEvent(id, "取得した大会", "", "")
			mockRepository.EXPECT().FindById(context.Background(), id).Return(fetched, nil)

			ret, err := usecase.FindById(context.Background(), id)

			require.NoError(t, err)
			require.Equal(t, "取得した大会", ret.Title)
			require.Len(t, store.savedByCalls, 1)
			require.Equal(t, id, store.savedByCalls[0].ID)
		})

		t.Run("正常系_保存に失敗しても取得結果は返す", func(t *testing.T) {
			mockRepository := mock_repository.NewMockTonamelEventInterface(gomock.NewController(t))
			store := &stubTonamelEventStore{saveErr: errors.New("db down")}
			usecase := NewTonamelEvent(mockRepository, store)

			mockRepository.EXPECT().FindById(context.Background(), id).Return(entity.NewTonamelEvent(id, "取得した大会", "", ""), nil)

			ret, err := usecase.FindById(context.Background(), id)

			require.NoError(t, err)
			require.Equal(t, "取得した大会", ret.Title)
		})

		t.Run("正常系_保存済みの確認に失敗しても取得へ進む", func(t *testing.T) {
			mockRepository := mock_repository.NewMockTonamelEventInterface(gomock.NewController(t))
			store := &stubTonamelEventStore{err: errors.New("db down")}
			usecase := NewTonamelEvent(mockRepository, store)

			mockRepository.EXPECT().FindById(context.Background(), id).Return(entity.NewTonamelEvent(id, "取得した大会", "", ""), nil)

			ret, err := usecase.FindById(context.Background(), id)

			require.NoError(t, err)
			require.Equal(t, id, ret.ID)
		})

		t.Run("異常系_リポジトリのエラーをそのまま返し保存しない", func(t *testing.T) {
			mockRepository := mock_repository.NewMockTonamelEventInterface(gomock.NewController(t))
			store := &stubTonamelEventStore{}
			usecase := NewTonamelEvent(mockRepository, store)

			mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

			ret, err := usecase.FindById(context.Background(), id)

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Nil(t, ret)
			require.Empty(t, store.savedByCalls)
		})
	})
}
