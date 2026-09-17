package authorization

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

// デッキコードの参照は親デッキの公開範囲に従う(非公開デッキのコードは他人に見せない)。
func TestDeckCodeGetByIdAuthorizationMiddleware(t *testing.T) {
	owner := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	other := "KBp7roRDZobZg1t0OPzFR1kvLeO2"
	deckId := "01HD7Y3K8D6FDHMHTZ2GT41TD1"

	setup := func(t *testing.T) (*mock_repository.MockDeckCodeInterface, *mock_repository.MockDeckInterface) {
		mockCtrl := gomock.NewController(t)
		return mock_repository.NewMockDeckCodeInterface(mockCtrl), mock_repository.NewMockDeckInterface(mockCtrl)
	}

	t.Run("正常系_公開デッキのコードは未認証でも通過する", func(t *testing.T) {
		deckCodeRepo, deckRepo := setup(t)
		id, _ := generateId()
		ctx, w := newDeckCodeAuthContext(t, id, "")

		deckCodeRepo.EXPECT().FindById(gomock.Any(), id).Return(&entity.DeckCode{ID: id, UserId: owner, DeckId: deckId}, nil)
		deckRepo.EXPECT().FindById(gomock.Any(), deckId).Return(&entity.Deck{ID: deckId, UserId: owner, PrivateFlg: false}, nil)

		DeckCodeGetByIdAuthorizationMiddleware(deckCodeRepo, deckRepo)(ctx)

		require.Equal(t, http.StatusOK, w.Code)
		require.False(t, ctx.IsAborted())
	})

	t.Run("正常系_非公開デッキのコードでも所有者なら通過する", func(t *testing.T) {
		deckCodeRepo, deckRepo := setup(t)
		id, _ := generateId()
		ctx, w := newDeckCodeAuthContext(t, id, owner)

		deckCodeRepo.EXPECT().FindById(gomock.Any(), id).Return(&entity.DeckCode{ID: id, UserId: owner, DeckId: deckId}, nil)
		deckRepo.EXPECT().FindById(gomock.Any(), deckId).Return(&entity.Deck{ID: deckId, UserId: owner, PrivateFlg: true}, nil)

		DeckCodeGetByIdAuthorizationMiddleware(deckCodeRepo, deckRepo)(ctx)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("異常系_非公開デッキのコードは他人には403を返す", func(t *testing.T) {
		deckCodeRepo, deckRepo := setup(t)
		id, _ := generateId()
		ctx, w := newDeckCodeAuthContext(t, id, other)

		deckCodeRepo.EXPECT().FindById(gomock.Any(), id).Return(&entity.DeckCode{ID: id, UserId: owner, DeckId: deckId}, nil)
		deckRepo.EXPECT().FindById(gomock.Any(), deckId).Return(&entity.Deck{ID: deckId, UserId: owner, PrivateFlg: true}, nil)

		DeckCodeGetByIdAuthorizationMiddleware(deckCodeRepo, deckRepo)(ctx)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_非公開デッキのコードは未認証にも403を返す", func(t *testing.T) {
		deckCodeRepo, deckRepo := setup(t)
		id, _ := generateId()
		ctx, w := newDeckCodeAuthContext(t, id, "")

		deckCodeRepo.EXPECT().FindById(gomock.Any(), id).Return(&entity.DeckCode{ID: id, UserId: owner, DeckId: deckId}, nil)
		deckRepo.EXPECT().FindById(gomock.Any(), deckId).Return(&entity.Deck{ID: deckId, UserId: owner, PrivateFlg: true}, nil)

		DeckCodeGetByIdAuthorizationMiddleware(deckCodeRepo, deckRepo)(ctx)

		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("異常系_デッキコードが無ければ404を返す", func(t *testing.T) {
		deckCodeRepo, deckRepo := setup(t)
		id, _ := generateId()
		ctx, w := newDeckCodeAuthContext(t, id, owner)

		deckCodeRepo.EXPECT().FindById(gomock.Any(), id).Return(nil, apperror.ErrRecordNotFound)

		DeckCodeGetByIdAuthorizationMiddleware(deckCodeRepo, deckRepo)(ctx)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_親デッキが無ければ404を返す", func(t *testing.T) {
		deckCodeRepo, deckRepo := setup(t)
		id, _ := generateId()
		ctx, w := newDeckCodeAuthContext(t, id, owner)

		deckCodeRepo.EXPECT().FindById(gomock.Any(), id).Return(&entity.DeckCode{ID: id, UserId: owner, DeckId: deckId}, nil)
		deckRepo.EXPECT().FindById(gomock.Any(), deckId).Return(nil, apperror.ErrRecordNotFound)

		DeckCodeGetByIdAuthorizationMiddleware(deckCodeRepo, deckRepo)(ctx)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("異常系_取得エラーなら500を返す", func(t *testing.T) {
		deckCodeRepo, deckRepo := setup(t)
		id, _ := generateId()
		ctx, w := newDeckCodeAuthContext(t, id, owner)

		deckCodeRepo.EXPECT().FindById(gomock.Any(), id).Return(nil, errors.New(""))

		DeckCodeGetByIdAuthorizationMiddleware(deckCodeRepo, deckRepo)(ctx)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
