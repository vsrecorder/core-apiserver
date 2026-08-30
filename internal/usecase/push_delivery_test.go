package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func TestPushDeliveryUsecase(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.Local)

	t.Run("MarkDelivered_正常系_本人のidを現在時刻で記録する", func(t *testing.T) {
		overrideTimeNow(t, now)
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushDeliveryInterface(mockCtrl)
		u := NewPushDelivery(repo)

		repo.EXPECT().MarkDelivered(gomock.Any(), "d-1", "user-1", now).Return(nil)

		require.NoError(t, u.MarkDelivered(context.Background(), "user-1", "d-1"))
	})

	t.Run("MarkClicked_異常系_他人のidはErrRecordNotFoundをそのまま返す", func(t *testing.T) {
		overrideTimeNow(t, now)
		mockCtrl := gomock.NewController(t)
		repo := mock_repository.NewMockPushDeliveryInterface(mockCtrl)
		u := NewPushDelivery(repo)

		repo.EXPECT().MarkClicked(gomock.Any(), "d-1", "user-2", now).Return(apperror.ErrRecordNotFound)

		require.ErrorIs(t, u.MarkClicked(context.Background(), "user-2", "d-1"), apperror.ErrRecordNotFound)
	})
}
