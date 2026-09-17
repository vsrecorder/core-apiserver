package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

// 参照先(デッキ・デッキコード・記録)の所有者検証。
// 他人のものと存在しないものは、IDの存在を教えないためどちらも ErrRecordNotFound にする。
func TestVerifyOwnership(t *testing.T) {
	uid := "owner"
	other := "someone-else"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	t.Run("verifyDeckOwnership", func(t *testing.T) {
		newMock := func(t *testing.T) *mock_repository.MockDeckInterface {
			return mock_repository.NewMockDeckInterface(gomock.NewController(t))
		}

		t.Run("正常系_未指定なら参照なしとして通す", func(t *testing.T) {
			// 参照が無いのでリポジトリは呼ばれない(EXPECT を書かない)
			require.NoError(t, verifyDeckOwnership(context.Background(), newMock(t), uid, ""))
		})

		t.Run("正常系_本人のデッキは通す", func(t *testing.T) {
			repo := newMock(t)
			repo.EXPECT().FindById(context.Background(), id).Return(&entity.Deck{ID: id, UserId: uid}, nil)

			require.NoError(t, verifyDeckOwnership(context.Background(), repo, uid, id))
		})

		t.Run("異常系_他人のデッキはErrRecordNotFound", func(t *testing.T) {
			repo := newMock(t)
			repo.EXPECT().FindById(context.Background(), id).Return(&entity.Deck{ID: id, UserId: other}, nil)

			require.ErrorIs(t, verifyDeckOwnership(context.Background(), repo, uid, id), apperror.ErrRecordNotFound)
		})

		t.Run("異常系_存在しないデッキはErrRecordNotFound", func(t *testing.T) {
			repo := newMock(t)
			repo.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

			require.ErrorIs(t, verifyDeckOwnership(context.Background(), repo, uid, id), apperror.ErrRecordNotFound)
		})

		t.Run("異常系_リポジトリのエラーはそのまま返す", func(t *testing.T) {
			repo := newMock(t)
			repoErr := errors.New("db down")
			repo.EXPECT().FindById(context.Background(), id).Return(nil, repoErr)

			require.ErrorIs(t, verifyDeckOwnership(context.Background(), repo, uid, id), repoErr)
		})
	})

	t.Run("verifyDeckCodeOwnership", func(t *testing.T) {
		newMock := func(t *testing.T) *mock_repository.MockDeckCodeInterface {
			return mock_repository.NewMockDeckCodeInterface(gomock.NewController(t))
		}

		t.Run("正常系_未指定なら参照なしとして通す", func(t *testing.T) {
			require.NoError(t, verifyDeckCodeOwnership(context.Background(), newMock(t), uid, ""))
		})

		t.Run("正常系_本人のデッキコードは通す", func(t *testing.T) {
			repo := newMock(t)
			repo.EXPECT().FindById(context.Background(), id).Return(&entity.DeckCode{ID: id, UserId: uid}, nil)

			require.NoError(t, verifyDeckCodeOwnership(context.Background(), repo, uid, id))
		})

		t.Run("異常系_他人のデッキコードはErrRecordNotFound", func(t *testing.T) {
			repo := newMock(t)
			repo.EXPECT().FindById(context.Background(), id).Return(&entity.DeckCode{ID: id, UserId: other}, nil)

			require.ErrorIs(t, verifyDeckCodeOwnership(context.Background(), repo, uid, id), apperror.ErrRecordNotFound)
		})

		t.Run("異常系_存在しないデッキコードはErrRecordNotFound", func(t *testing.T) {
			repo := newMock(t)
			repo.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

			require.ErrorIs(t, verifyDeckCodeOwnership(context.Background(), repo, uid, id), apperror.ErrRecordNotFound)
		})
	})

	t.Run("verifyRecordOwnership", func(t *testing.T) {
		newMock := func(t *testing.T) *mock_repository.MockRecordInterface {
			return mock_repository.NewMockRecordInterface(gomock.NewController(t))
		}

		t.Run("正常系_本人の記録は通す", func(t *testing.T) {
			repo := newMock(t)
			repo.EXPECT().FindById(context.Background(), id).Return(&entity.Record{ID: id, UserId: uid}, nil)

			require.NoError(t, verifyRecordOwnership(context.Background(), repo, uid, id))
		})

		t.Run("異常系_他人の記録はErrRecordNotFound", func(t *testing.T) {
			repo := newMock(t)
			repo.EXPECT().FindById(context.Background(), id).Return(&entity.Record{ID: id, UserId: other}, nil)

			require.ErrorIs(t, verifyRecordOwnership(context.Background(), repo, uid, id), apperror.ErrRecordNotFound)
		})

		t.Run("異常系_存在しない記録はErrRecordNotFound", func(t *testing.T) {
			repo := newMock(t)
			repo.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

			require.ErrorIs(t, verifyRecordOwnership(context.Background(), repo, uid, id), apperror.ErrRecordNotFound)
		})
	})
}

// 他人のデッキにデッキコードを足せると、そのデッキの公開一覧の「最新デッキコード」を
// 差し替えられてしまう。外部サイトへの取得も保存も始める前に弾くこと。
func TestDeckCodeUsecase_Create_RejectsOthersDeck(t *testing.T) {
	uid := "owner"
	deckId := "01HD7Y3K8D6FDHMHTZ2GT41TN2"
	code := "5dbFbk-uBwjqP-VVk5Vv"

	newUsecase := func(t *testing.T, deck *entity.Deck, deckErr error) DeckCodeInterface {
		mockCtrl := gomock.NewController(t)
		mockRepository := mock_repository.NewMockDeckCodeInterface(mockCtrl)
		mockDeckRepository := mock_repository.NewMockDeckInterface(mockCtrl)
		mockDeckAsset := mock_repository.NewMockDeckAssetInterface(mockCtrl)
		mockTagRepository := mock_repository.NewMockTagInterface(mockCtrl)

		mockDeckRepository.EXPECT().FindById(context.Background(), deckId).Return(deck, deckErr)
		// 所有者検証で落ちるため、アップロード(deckAsset)・保存(repository)・タグ(tag)は呼ばれない

		return NewDeckCode(mockRepository, mockDeckRepository, mockDeckAsset, mockTagRepository, spyDeckCodeBadgeEvaluation{called: new(bool)}, stubTransactionManager{}, nil)
	}

	t.Run("異常系_他人のデッキへの作成はErrRecordNotFoundでアップロードも保存もしない", func(t *testing.T) {
		usecase := newUsecase(t, &entity.Deck{ID: deckId, UserId: "someone-else"}, nil)

		ret, err := usecase.Create(context.Background(), NewDeckCodeCreateParam(uid, deckId, code, false, "", nil))

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		require.Nil(t, ret)
	})

	t.Run("異常系_存在しないデッキへの作成はErrRecordNotFound", func(t *testing.T) {
		usecase := newUsecase(t, nil, apperror.ErrRecordNotFound)

		ret, err := usecase.Create(context.Background(), NewDeckCodeCreateParam(uid, deckId, code, false, "", nil))

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		require.Nil(t, ret)
	})
}

// 記録の deck_id / deck_code_id に他人のものを指定できると、相手が「記録に使われている」
// としてデッキを削除できなくなる。作成・更新のどちらでも保存前に弾くこと。
func TestRecordUsecase_RejectsOthersDeckReferences(t *testing.T) {
	uid := "owner"
	other := "someone-else"
	deckId := "01HD7Y3K8D6FDHMHTZ2GT41TN2"
	deckCodeId := "01HD7Y3K8D6FDHMHTZ2GT41TC2"

	newUsecase := func(mockRepository *mock_repository.MockRecordInterface, deckOwner string, deckCodeOwner string) RecordInterface {
		return NewRecord(
			testLogger(),
			mockRepository,
			stubDeckRepository{owner: deckOwner},
			stubDeckCodeRepository{owner: deckCodeOwner},
			stubTagRepository{},
			stubBadgeEvaluation{},
			stubDesignationEvaluation{},
			&stubTonamelEventFetcher{},
			&stubTonamelEventStore{},
			stubTransactionManager{},
		)
	}

	t.Run("異常系_他人のデッキを参照する記録の作成はErrRecordNotFoundで保存しない", func(t *testing.T) {
		mockRepository := mock_repository.NewMockRecordInterface(gomock.NewController(t))
		usecase := newUsecase(mockRepository, other, uid)

		param := NewRecordParam(1, "", "", "", uid, deckId, "", testRecordEventDate, false, false, entity.RegulationIdStandard, "", "")

		ret, err := usecase.Create(context.Background(), param)

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		require.Nil(t, ret)
	})

	t.Run("異常系_他人のデッキコードを参照する記録の作成はErrRecordNotFoundで保存しない", func(t *testing.T) {
		mockRepository := mock_repository.NewMockRecordInterface(gomock.NewController(t))
		usecase := newUsecase(mockRepository, uid, other)

		param := NewRecordParam(1, "", "", "", uid, deckId, deckCodeId, testRecordEventDate, false, false, entity.RegulationIdStandard, "", "")

		ret, err := usecase.Create(context.Background(), param)

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		require.Nil(t, ret)
	})

	t.Run("異常系_更新で他人のデッキへ付け替えるとErrRecordNotFoundで保存しない", func(t *testing.T) {
		mockRepository := mock_repository.NewMockRecordInterface(gomock.NewController(t))
		usecase := newUsecase(mockRepository, other, uid)

		id, _ := generateId()
		existing := entity.NewRecord(id, time.Now().Local(), 1, "", "", "", uid, "", "", testRecordEventDate, false, false, entity.RegulationIdStandard, "", "")
		mockRepository.EXPECT().FindById(context.Background(), id).Return(existing, nil)
		// Save は EXPECT しない(所有者検証で弾かれ、呼ばれない)

		param := NewRecordParam(1, "", "", "", uid, deckId, "", testRecordEventDate, false, false, entity.RegulationIdStandard, "", "")

		ret, err := usecase.Update(context.Background(), id, param)

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		require.Nil(t, ret)
	})

	t.Run("正常系_本人のデッキとデッキコードの参照は保存される", func(t *testing.T) {
		mockRepository := mock_repository.NewMockRecordInterface(gomock.NewController(t))
		usecase := newUsecase(mockRepository, uid, uid)

		mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

		param := NewRecordParam(1, "", "", "", uid, deckId, deckCodeId, testRecordEventDate, false, false, entity.RegulationIdStandard, "", "")

		ret, err := usecase.Create(context.Background(), param)

		require.NoError(t, err)
		require.Equal(t, deckId, ret.DeckId)
		require.Equal(t, deckCodeId, ret.DeckCodeId)
	})
}

// 他人の記録に対戦結果を混ぜられると、その人の記録の対戦一覧・集計に他人の対戦が現れる。
// 作成でも、更新での record_id の付け替えでも保存前に弾くこと。
func TestMatchUsecase_RejectsOthersReferences(t *testing.T) {
	uid := "owner"
	other := "someone-else"
	recordId := "01JMPK4VF04QX714CG4PHYJ88K"
	deckId := "01JMKRNBW5TVN902YAE8GYZ367"

	newUsecase := func(t *testing.T, deckOwner string) (*mock_repository.MockMatchInterface, *mock_repository.MockRecordInterface, MatchInterface) {
		mockCtrl := gomock.NewController(t)
		mockRepository := mock_repository.NewMockMatchInterface(mockCtrl)
		mockRecordRepository := mock_repository.NewMockRecordInterface(mockCtrl)

		usecase := NewMatch(
			mockRepository,
			mockRecordRepository,
			stubDeckRepository{owner: deckOwner},
			stubDeckCodeRepository{owner: deckOwner},
			stubTagRepository{},
			stubBadgeEvaluation{},
			stubDesignationEvaluation{},
			stubEnvironmentBadgeEvaluation{},
			stubTransactionManager{},
		)

		return mockRepository, mockRecordRepository, usecase
	}

	// 検証を通す最小構成のパラメータ(既存の「正常系_BO1のマッチをゲーム込みで作成する」と同じ)
	newParam := func(deckId string) *MatchParam {
		return NewMatchParam(
			recordId, deckId, "", uid, "",
			false, false, false, false, false, false, false, false, false,
			"", "", []*GameParam{NewGameParam(true, false, 0, 0, "")}, nil,
		)
	}

	t.Run("異常系_他人の記録への作成はErrRecordNotFoundで保存しない", func(t *testing.T) {
		_, mockRecordRepository, usecase := newUsecase(t, uid)
		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(&entity.Record{ID: recordId, UserId: other}, nil)
		// Create は EXPECT しない(所有者検証で弾かれ、呼ばれない)

		ret, err := usecase.Create(context.Background(), newParam(""))

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		require.Nil(t, ret)
	})

	t.Run("異常系_存在しない記録への作成はErrRecordNotFound", func(t *testing.T) {
		_, mockRecordRepository, usecase := newUsecase(t, uid)
		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(nil, apperror.ErrRecordNotFound)

		ret, err := usecase.Create(context.Background(), newParam(""))

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		require.Nil(t, ret)
	})

	t.Run("異常系_他人のデッキを参照する作成はErrRecordNotFoundで保存しない", func(t *testing.T) {
		_, mockRecordRepository, usecase := newUsecase(t, other)
		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(&entity.Record{ID: recordId, UserId: uid}, nil)

		ret, err := usecase.Create(context.Background(), newParam(deckId))

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		require.Nil(t, ret)
	})

	t.Run("異常系_更新で他人の記録へ付け替えるとErrRecordNotFoundで保存しない", func(t *testing.T) {
		mockRepository, mockRecordRepository, usecase := newUsecase(t, uid)

		matchId, _ := generateId()
		mockRepository.EXPECT().FindById(context.Background(), matchId).Return(&entity.Match{ID: matchId, UserId: uid, RecordId: "01JMPK4VF04QX714CG4PHYJ000"}, nil)
		mockRecordRepository.EXPECT().FindById(context.Background(), recordId).Return(&entity.Record{ID: recordId, UserId: other}, nil)
		// Update は EXPECT しない(所有者検証で弾かれ、呼ばれない)

		ret, err := usecase.Update(context.Background(), matchId, newParam(""))

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		require.Nil(t, ret)
	})
}

// ユーザー横断の最新対戦(相手デッキの入力候補用)は、他人に見せる前提ではない項目を落として返す。
func TestMatchUsecase_FindLatest_SanitizesForPublicFeed(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockMatchInterface(mockCtrl)
	mockRecordRepository := mock_repository.NewMockRecordInterface(mockCtrl)

	usecase := NewMatch(
		mockRepository,
		mockRecordRepository,
		stubDeckRepository{},
		stubDeckCodeRepository{},
		stubTagRepository{},
		stubBadgeEvaluation{},
		stubDesignationEvaluation{},
		stubEnvironmentBadgeEvaluation{},
		stubTransactionManager{},
	)

	t.Run("正常系_メモ・タグ・参照先を落とし相手デッキ情報とスプライトは残す", func(t *testing.T) {
		limit := 10
		match := &entity.Match{
			ID:                "01HD7Y3K8D6FDHMHTZ2GT41TN1",
			RecordId:          "01JMPK4VF04QX714CG4PHYJ88K",
			DeckId:            "01JMKRNBW5TVN902YAE8GYZ367",
			DeckCodeId:        "01HD7Y3K8D6FDHMHTZ2GT41TC2",
			UserId:            "someone",
			OpponentsUserId:   "opponent",
			DefaultVictoryFlg: true,
			OpponentsDeckInfo: "ロストバレット",
			Memo:              "本人向けの覚え書き",
			Games:             []*entity.Game{{ID: "g1", Memo: "対局メモ", WinningFlg: true}},
			PokemonSprites:    []*entity.PokemonSprite{{ID: "0887", Position: 1}},
			Tags:              []*entity.Tag{{ID: "t1", Name: "他人のタグ"}},
		}
		mockRepository.EXPECT().FindLatest(context.Background(), limit).Return([]*entity.Match{match}, nil)

		ret, err := usecase.FindLatest(context.Background(), limit)

		require.NoError(t, err)
		require.Len(t, ret, 1)

		// 候補の生成に要る項目は残る
		require.Equal(t, "ロストバレット", ret[0].OpponentsDeckInfo)
		require.Equal(t, "0887", ret[0].PokemonSprites[0].ID)
		require.True(t, ret[0].DefaultVictoryFlg)
		require.True(t, ret[0].Games[0].WinningFlg)

		// 本人向けの項目は落ちる
		require.Empty(t, ret[0].Memo)
		require.Empty(t, ret[0].Games[0].Memo)
		require.Empty(t, ret[0].OpponentsUserId)
		require.Empty(t, ret[0].DeckId)
		require.Empty(t, ret[0].DeckCodeId)
		require.Nil(t, ret[0].Tags)
	})

	t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
		mockRepository.EXPECT().FindLatest(context.Background(), 10).Return(nil, errors.New(""))

		ret, err := usecase.FindLatest(context.Background(), 10)

		require.Error(t, err)
		require.Nil(t, ret)
	})
}
