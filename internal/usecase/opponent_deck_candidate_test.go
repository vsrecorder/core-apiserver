package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

func TestOpponentDeckCandidateUsecase(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.Local)
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"

	// 自身の履歴の集計条件(直近6ヶ月・uid で絞る)にだけ一致するマッチャ
	ownFilter := func(limit int) gomock.Matcher {
		return gomock.Cond(func(f *repository.OpponentDeckCandidateFilter) bool {
			return f.UserId == uid &&
				f.Since.Equal(now.AddDate(0, -OwnOpponentDeckCandidateWindowMonths, 0)) &&
				f.Limit == limit
		})
	}

	// 全体候補の集計条件(直近3ヶ月・ユーザー指定なし)にだけ一致するマッチャ
	globalFilter := gomock.Cond(func(f *repository.OpponentDeckCandidateFilter) bool {
		return f.UserId == "" &&
			f.Since.Equal(now.AddDate(0, -OpponentDeckCandidateWindowMonths, 0)) &&
			f.Limit == opponentDeckCandidateCacheSize
	})

	candidate := func(deckInfo string, spriteId string, count int) *entity.OpponentDeckCandidate {
		sprites := []*entity.PokemonSprite{}
		if spriteId != "" {
			sprites = append(sprites, entity.NewPokemonSpriteWithPosition(spriteId, 1))
		}

		return entity.NewOpponentDeckCandidate(deckInfo, sprites, count)
	}

	t.Run("正常系_自身の履歴を先頭に不足分を全体候補で埋める", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockOpponentDeckCandidateInterface(gomock.NewController(t))
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), ownFilter(4)).
			Return([]*entity.OpponentDeckCandidate{candidate("自分のデッキA", "0887", 5)}, nil)
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), globalFilter).
			Return([]*entity.OpponentDeckCandidate{
				candidate("全体のデッキA", "0006", 30),
				candidate("全体のデッキB", "", 20),
				candidate("全体のデッキC", "", 10),
			}, nil)

		ret, err := NewOpponentDeckCandidate(repo).FindOpponentDeckCandidates(context.Background(), uid, 4)

		require.NoError(t, err)
		require.Len(t, ret, 4)
		require.Equal(t, "自分のデッキA", ret[0].OpponentsDeckInfo)
		require.Equal(t, "全体のデッキA", ret[1].OpponentsDeckInfo)
		require.Equal(t, "全体のデッキC", ret[3].OpponentsDeckInfo)
	})

	// 同じ組み合わせが両方に出たら、自身のぶんを残す(先頭側の順位を保つ)
	t.Run("正常系_自身の履歴と重複する全体候補は除く", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockOpponentDeckCandidateInterface(gomock.NewController(t))
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), ownFilter(3)).
			Return([]*entity.OpponentDeckCandidate{candidate("ロストバレット", "0887", 5)}, nil)
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), globalFilter).
			Return([]*entity.OpponentDeckCandidate{
				candidate("ロストバレット", "0887", 99), // 自身のぶんと同じ組み合わせ
				candidate("ロストバレット", "0006", 50), // スプライトが違うので別の候補
				candidate("サーナイトex", "", 40),
			}, nil)

		ret, err := NewOpponentDeckCandidate(repo).FindOpponentDeckCandidates(context.Background(), uid, 3)

		require.NoError(t, err)
		require.Len(t, ret, 3)
		require.Equal(t, 5, ret[0].Count) // 自身のぶん(99 の全体候補で置き換わらない)
		require.Equal(t, "0006", ret[1].PokemonSprites[0].ID)
		require.Equal(t, "サーナイトex", ret[2].OpponentsDeckInfo)
	})

	// 自身の履歴だけで埋まるなら、全体の集計は行わない
	t.Run("正常系_自身の履歴がlimitに達していれば全体候補を引かない", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockOpponentDeckCandidateInterface(gomock.NewController(t))
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), ownFilter(2)).
			Return([]*entity.OpponentDeckCandidate{
				candidate("自分のデッキA", "", 5),
				candidate("自分のデッキB", "", 3),
			}, nil)
		// 全体候補は EXPECT しない(引かれない)

		ret, err := NewOpponentDeckCandidate(repo).FindOpponentDeckCandidates(context.Background(), uid, 2)

		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, "自分のデッキA", ret[0].OpponentsDeckInfo)
	})

	t.Run("正常系_履歴が無いユーザーには全体候補だけを返す", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockOpponentDeckCandidateInterface(gomock.NewController(t))
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), ownFilter(2)).Return(nil, nil)
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), globalFilter).
			Return([]*entity.OpponentDeckCandidate{candidate("全体のデッキA", "", 30)}, nil)

		ret, err := NewOpponentDeckCandidate(repo).FindOpponentDeckCandidates(context.Background(), uid, 2)

		require.NoError(t, err)
		require.Len(t, ret, 1)
		require.Equal(t, "全体のデッキA", ret[0].OpponentsDeckInfo)
	})

	t.Run("正常系_limitが0以下なら空を返し集計しない", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockOpponentDeckCandidateInterface(gomock.NewController(t))

		ret, err := NewOpponentDeckCandidate(repo).FindOpponentDeckCandidates(context.Background(), uid, 0)

		require.NoError(t, err)
		require.Empty(t, ret)
	})

	// 全体候補は重い集計なので保持する。自身の履歴は毎回集計する
	// (対戦を記録した直後に、その相手デッキが候補へ出るようにするため)
	t.Run("正常系_全体候補はTTLのあいだ保持し自身の履歴は毎回集計する", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockOpponentDeckCandidateInterface(gomock.NewController(t))
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), ownFilter(5)).
			Return([]*entity.OpponentDeckCandidate{candidate("自分のデッキA", "", 5)}, nil).Times(3)
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), globalFilter).
			Return([]*entity.OpponentDeckCandidate{candidate("全体のデッキA", "", 30)}, nil).Times(1)
		usecase := NewOpponentDeckCandidate(repo)

		for i := 0; i < 3; i++ {
			ret, err := usecase.FindOpponentDeckCandidates(context.Background(), uid, 5)
			require.NoError(t, err)
			require.Len(t, ret, 2)
		}
	})

	t.Run("正常系_TTLが過ぎたら全体候補を集計し直す", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockOpponentDeckCandidateInterface(gomock.NewController(t))
		usecase := NewOpponentDeckCandidate(repo)

		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), gomock.Any()).Return(nil, nil)
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), globalFilter).
			Return([]*entity.OpponentDeckCandidate{candidate("古い環境のデッキ", "", 30)}, nil)
		ret, err := usecase.FindOpponentDeckCandidates(context.Background(), uid, 5)
		require.NoError(t, err)
		require.Equal(t, "古い環境のデッキ", ret[0].OpponentsDeckInfo)

		later := now.Add(opponentDeckCandidateCacheTTL + time.Second)
		overrideTimeNow(t, later)
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), gomock.Any()).Return(nil, nil)
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), gomock.Cond(func(f *repository.OpponentDeckCandidateFilter) bool {
			return f.UserId == "" && f.Since.Equal(later.AddDate(0, -OpponentDeckCandidateWindowMonths, 0))
		})).Return([]*entity.OpponentDeckCandidate{candidate("新しい環境のデッキ", "", 30)}, nil)

		ret, err = usecase.FindOpponentDeckCandidates(context.Background(), uid, 5)
		require.NoError(t, err)
		require.Equal(t, "新しい環境のデッキ", ret[0].OpponentsDeckInfo)
	})

	t.Run("異常系_自身の履歴の集計に失敗したらエラーを返す", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockOpponentDeckCandidateInterface(gomock.NewController(t))
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), ownFilter(5)).Return(nil, errors.New("db down"))

		ret, err := NewOpponentDeckCandidate(repo).FindOpponentDeckCandidates(context.Background(), uid, 5)

		require.Error(t, err)
		require.Nil(t, ret)
	})

	t.Run("異常系_全体候補の集計に失敗したらエラーを返し保持しない", func(t *testing.T) {
		overrideTimeNow(t, now)
		repo := mock_repository.NewMockOpponentDeckCandidateInterface(gomock.NewController(t))
		usecase := NewOpponentDeckCandidate(repo)

		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), ownFilter(5)).Return(nil, nil)
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), globalFilter).Return(nil, errors.New("db down"))
		ret, err := usecase.FindOpponentDeckCandidates(context.Background(), uid, 5)
		require.Error(t, err)
		require.Nil(t, ret)

		// 失敗は保持されず、次の呼び出しで集計し直す
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), ownFilter(5)).Return(nil, nil)
		repo.EXPECT().FindOpponentDeckCandidates(gomock.Any(), globalFilter).
			Return([]*entity.OpponentDeckCandidate{candidate("全体のデッキA", "", 30)}, nil)
		ret, err = usecase.FindOpponentDeckCandidates(context.Background(), uid, 5)
		require.NoError(t, err)
		require.Len(t, ret, 1)
	})
}
