package infrastructure

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
)

func TestTonamelEventInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"FindById": test_TonamelEventInfrastructure_FindById,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_TonamelEventInfrastructure_FindById(t *testing.T) {
	logger := slog.Default()
	r := NewTonamelEvent(logger)

	t.Run("正常系_実在イベントIDでタイトルと画像を取得できる", func(t *testing.T) {

		id := "OakZc"
		title := "第23回 ACEカップ～FINAL～"

		res, err := r.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, res.ID, id)
		require.Equal(t, res.Title, title)
		require.NotEmpty(t, res.Image)
	})

	t.Run("異常系_存在しないIDはErrRecordNotFoundを返す", func(t *testing.T) {
		id := "ERROR"

		_, err := r.FindById(context.Background(), id)

		require.Equal(t, err, apperror.ErrRecordNotFound)
	})
}
