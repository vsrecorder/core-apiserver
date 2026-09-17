package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func TestOpponentDeckCandidateUsecase(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.Local)

	candidates := []*entity.OpponentDeckCandidate{
		entity.NewOpponentDeckCandidate("ロストバレット", []*entity.PokemonSprite{entity.NewPokemonSpriteWithPosition("0887", 1)}, 12),
		entity.NewOpponentDeckCandidate("サーナイトex", nil, 7),
		entity.NewOpponentDeckCandidate("リザードンex", nil, 5),
	}

	t.Run("正常系_直近の期間を対象に集計しlimit件を返す", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockOpponentDeckCandidateInterface(gomock.NewController(t))
		// 集計の対象期間は now から OpponentDeckCandidateWindow だけ遡った時点以降。
		// 取得件数はリクエストの limit ではなく保持する上限(100)で、そこから切り出す。
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), now.Add(-OpponentDeckCandidateWindow), opponentDeckCandidateCacheSize).Return(candidates, nil)

		ret, err := NewOpponentDeckCandidate(repo).FindOpponentDeckCandidates(context.Background(), 2)

		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, "ロストバレット", ret[0].OpponentsDeckInfo)
		require.Equal(t, "サーナイトex", ret[1].OpponentsDeckInfo)
	})

	t.Run("正常系_保持している件数を超えるlimitは全件を返す", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockOpponentDeckCandidateInterface(gomock.NewController(t))
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), gomock.Any(), opponentDeckCandidateCacheSize).Return(candidates, nil)

		ret, err := NewOpponentDeckCandidate(repo).FindOpponentDeckCandidates(context.Background(), 100)

		require.NoError(t, err)
		require.Len(t, ret, 3)
	})

	// 全ユーザーの対戦結果を GROUP BY する集計なので、フォームを開くたびには走らせない
	t.Run("正常系_TTLのあいだは集計を繰り返さない", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockOpponentDeckCandidateInterface(gomock.NewController(t))
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), gomock.Any(), opponentDeckCandidateCacheSize).Return(candidates, nil).Times(1)
		usecase := NewOpponentDeckCandidate(repo)

		for i := 0; i < 3; i++ {
			ret, err := usecase.FindOpponentDeckCandidates(context.Background(), 10)
			require.NoError(t, err)
			require.Len(t, ret, 3)
		}
	})

	t.Run("正常系_TTLが過ぎたら集計し直す", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockOpponentDeckCandidateInterface(gomock.NewController(t))
		usecase := NewOpponentDeckCandidate(repo)

		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), gomock.Any(), opponentDeckCandidateCacheSize).Return(candidates, nil)
		_, err := usecase.FindOpponentDeckCandidates(context.Background(), 10)
		require.NoError(t, err)

		later := now.Add(opponentDeckCandidateCacheTTL + time.Second)
		overrideTimeNow(t, later)
		refreshed := []*entity.OpponentDeckCandidate{entity.NewOpponentDeckCandidate("新しい環境のデッキ", nil, 3)}
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), later.Add(-OpponentDeckCandidateWindow), opponentDeckCandidateCacheSize).Return(refreshed, nil)

		ret, err := usecase.FindOpponentDeckCandidates(context.Background(), 10)

		require.NoError(t, err)
		require.Len(t, ret, 1)
		require.Equal(t, "新しい環境のデッキ", ret[0].OpponentsDeckInfo)
	})

	// 返した結果を呼び出し側が並び替えても、保持している結果は変わらない
	t.Run("正常系_返す結果はコピーで保持している結果に影響しない", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockOpponentDeckCandidateInterface(gomock.NewController(t))
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), gomock.Any(), opponentDeckCandidateCacheSize).Return(candidates, nil).Times(1)
		usecase := NewOpponentDeckCandidate(repo)

		first, err := usecase.FindOpponentDeckCandidates(context.Background(), 3)
		require.NoError(t, err)
		first[0], first[2] = first[2], first[0]

		second, err := usecase.FindOpponentDeckCandidates(context.Background(), 3)
		require.NoError(t, err)
		require.Equal(t, "ロストバレット", second[0].OpponentsDeckInfo)
	})

	t.Run("異常系_集計の失敗はそのまま返し保持しない", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockOpponentDeckCandidateInterface(gomock.NewController(t))
		usecase := NewOpponentDeckCandidate(repo)

		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), gomock.Any(), opponentDeckCandidateCacheSize).Return(nil, errors.New("db down"))
		ret, err := usecase.FindOpponentDeckCandidates(context.Background(), 10)
		require.Error(t, err)
		require.Nil(t, ret)

		// 失敗は保持されず、次の呼び出しで集計し直す
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), gomock.Any(), opponentDeckCandidateCacheSize).Return(candidates, nil)
		ret, err = usecase.FindOpponentDeckCandidates(context.Background(), 10)
		require.NoError(t, err)
		require.Len(t, ret, 3)
	})
}
