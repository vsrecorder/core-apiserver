package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func TestOpponentDeckUsecase_Replace(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	t.Run("正常系_positionの省略は並び順で採番して置き換える", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockRepository := mock_repository.NewMockOpponentDeckInterface(mockCtrl)
		u := NewOpponentDeck(mockRepository)

		mockRepository.EXPECT().Replace(gomock.Any(), uid, gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, _ string, from *entity.OpponentDeckSpec, to *entity.OpponentDeckSpec) (int, error) {
				require.Equal(t, "ドラパ", from.OpponentsDeckInfo)
				require.Equal(t, "0887", from.SpriteIdAt(2))
				require.Equal(t, "", from.SpriteIdAt(1))
				require.Equal(t, "ドラパルトex", to.OpponentsDeckInfo)
				require.Equal(t, "0887", to.SpriteIdAt(1))
				require.Equal(t, "0006", to.SpriteIdAt(2))
				return 4, nil
			})

		updated, err := u.Replace(context.Background(), uid, NewOpponentDeckReplaceParam(
			// 2体目だけのスプライト(position 指定あり)
			NewOpponentDeckSpecParam("ドラパ", []*PokemonSpriteParam{NewPokemonSpriteParamWithPosition("0887", 2)}),
			// position の省略
			NewOpponentDeckSpecParam("ドラパルトex", []*PokemonSpriteParam{NewPokemonSpriteParam("0887"), NewPokemonSpriteParam("0006")}),
		))

		require.NoError(t, err)
		require.Equal(t, 4, updated)
	})

	t.Run("異常系_リポジトリのエラーを返す", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockRepository := mock_repository.NewMockOpponentDeckInterface(mockCtrl)
		u := NewOpponentDeck(mockRepository)

		mockRepository.EXPECT().Replace(gomock.Any(), uid, gomock.Any(), gomock.Any()).Return(0, errors.New(""))

		_, err := u.Replace(context.Background(), uid, NewOpponentDeckReplaceParam(
			NewOpponentDeckSpecParam("ドラパ", nil),
			NewOpponentDeckSpecParam("ドラパルトex", nil),
		))

		require.Error(t, err)
	})
}
