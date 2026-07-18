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

// spyDeckCodeBadgeEvaluation はEvaluateOnDeckCodeCreatedの呼び出し有無だけを
// 記録する手書きスタブ(stubBadgeEvaluationと同じ理由でgomockを使わない)。
type spyDeckCodeBadgeEvaluation struct {
	stubBadgeEvaluation
	called *bool
}

func (s spyDeckCodeBadgeEvaluation) EvaluateOnDeckCodeCreated(
	ctx context.Context,
	userId string,
	deckCode *entity.DeckCode,
) {
	*s.called = true
}

func setup4DeckCodeUsecase(t *testing.T) (
	*mock_repository.MockDeckCodeInterface,
	*mock_repository.MockDeckAssetInterface,
	*bool,
	DeckCodeInterface,
) {
	mockCtrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockDeckCodeInterface(mockCtrl)
	mockDeckAsset := mock_repository.NewMockDeckAssetInterface(mockCtrl)

	badgeEvaluationCalled := false
	usecase := NewDeckCode(mockRepository, mockDeckAsset, spyDeckCodeBadgeEvaluation{called: &badgeEvaluationCalled})

	return mockRepository, mockDeckAsset, &badgeEvaluationCalled, usecase
}

func TestDeckCodeUsecase(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	deckId := "01HD7Y3K8D6FDHMHTZ2GT41TN2"
	code := "5dbFbk-uBwjqP-VVk5Vv"

	t.Run("FindById", func(t *testing.T) {
		t.Run("正常系_指定IDのデッキコードを返す", func(t *testing.T) {
			mockRepository, _, _, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			deckCode := &entity.DeckCode{ID: id}

			mockRepository.EXPECT().FindById(context.Background(), id).Return(deckCode, nil)

			ret, err := usecase.FindById(context.Background(), id)

			require.NoError(t, err)
			require.Equal(t, id, ret.ID)
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			mockRepository, _, _, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, errors.New(""))

			ret, err := usecase.FindById(context.Background(), id)

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("FindByDeckId", func(t *testing.T) {
		t.Run("正常系_指定デッキのデッキコード一覧を返す", func(t *testing.T) {
			mockRepository, _, _, usecase := setup4DeckCodeUsecase(t)

			deckCodes := []*entity.DeckCode{{ID: "01HD7Y3K8D6FDHMHTZ2GT41TC1", DeckId: deckId}}

			mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return(deckCodes, nil)

			ret, err := usecase.FindByDeckId(context.Background(), deckId)

			require.NoError(t, err)
			require.Len(t, ret, 1)
			require.Equal(t, deckId, ret[0].DeckId)
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			mockRepository, _, _, usecase := setup4DeckCodeUsecase(t)

			mockRepository.EXPECT().FindByDeckId(context.Background(), deckId).Return(nil, errors.New(""))

			ret, err := usecase.FindByDeckId(context.Background(), deckId)

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("Create", func(t *testing.T) {
		t.Run("正常系_コード未指定なら外部アップロードと称号評価なしで保存する", func(t *testing.T) {
			mockRepository, _, badgeEvaluationCalled, usecase := setup4DeckCodeUsecase(t)

			param := NewDeckCodeCreateParam(uid, deckId, "", false, "")

			mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

			ret, err := usecase.Create(context.Background(), param)

			require.NoError(t, err)
			require.NotEmpty(t, ret.ID)
			require.NotEmpty(t, ret.CreatedAt)
			require.Equal(t, uid, ret.UserId)
			require.Equal(t, deckId, ret.DeckId)
			require.Empty(t, ret.Code)
			require.False(t, *badgeEvaluationCalled)
		})

		t.Run("正常系_コード指定時はHTMLと画像をアップロードして保存し称号評価する", func(t *testing.T) {
			mockRepository, mockDeckAsset, badgeEvaluationCalled, usecase := setup4DeckCodeUsecase(t)

			param := NewDeckCodeCreateParam(uid, deckId, code, true, "メモ")

			gomock.InOrder(
				mockDeckAsset.EXPECT().UploadDeckResultHTML(context.Background(), code).Return(nil),
				mockDeckAsset.EXPECT().UploadDeckImage(context.Background(), code).Return(nil),
				mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil),
			)

			ret, err := usecase.Create(context.Background(), param)

			require.NoError(t, err)
			require.Equal(t, code, ret.Code)
			require.True(t, ret.PrivateCodeFlg)
			require.Equal(t, "メモ", ret.Memo)
			require.True(t, *badgeEvaluationCalled)
		})

		t.Run("異常系_HTMLアップロード失敗時は画像アップロードも保存も行わない", func(t *testing.T) {
			_, mockDeckAsset, badgeEvaluationCalled, usecase := setup4DeckCodeUsecase(t)

			param := NewDeckCodeCreateParam(uid, deckId, code, false, "")

			mockDeckAsset.EXPECT().UploadDeckResultHTML(context.Background(), code).Return(errors.New(""))

			ret, err := usecase.Create(context.Background(), param)

			require.Error(t, err)
			require.Nil(t, ret)
			require.False(t, *badgeEvaluationCalled)
		})

		t.Run("異常系_画像アップロード失敗時は保存を行わない", func(t *testing.T) {
			_, mockDeckAsset, badgeEvaluationCalled, usecase := setup4DeckCodeUsecase(t)

			param := NewDeckCodeCreateParam(uid, deckId, code, false, "")

			gomock.InOrder(
				mockDeckAsset.EXPECT().UploadDeckResultHTML(context.Background(), code).Return(nil),
				mockDeckAsset.EXPECT().UploadDeckImage(context.Background(), code).Return(errors.New("")),
			)

			ret, err := usecase.Create(context.Background(), param)

			require.Error(t, err)
			require.Nil(t, ret)
			require.False(t, *badgeEvaluationCalled)
		})

		t.Run("異常系_保存失敗時はエラーを返し称号評価しない", func(t *testing.T) {
			mockRepository, _, badgeEvaluationCalled, usecase := setup4DeckCodeUsecase(t)

			param := NewDeckCodeCreateParam(uid, deckId, "", false, "")

			mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(errors.New(""))

			ret, err := usecase.Create(context.Background(), param)

			require.Error(t, err)
			require.Nil(t, ret)
			require.False(t, *badgeEvaluationCalled)
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("正常系_公開設定とメモのみ更新され他は維持される", func(t *testing.T) {
			mockRepository, _, _, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			createdAt := time.Now().Local()
			existing := entity.NewDeckCode(id, createdAt, uid, deckId, code, false, "更新前のメモ")

			mockRepository.EXPECT().FindById(context.Background(), id).Return(existing, nil)
			mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(nil)

			param := NewDeckCodeUpdateParam(true, "更新後のメモ")

			ret, err := usecase.Update(context.Background(), id, param)

			require.NoError(t, err)
			require.Equal(t, id, ret.ID)
			require.Equal(t, createdAt, ret.CreatedAt)
			require.Equal(t, uid, ret.UserId)
			require.Equal(t, deckId, ret.DeckId)
			require.Equal(t, code, ret.Code)
			require.True(t, ret.PrivateCodeFlg)
			require.Equal(t, "更新後のメモ", ret.Memo)
		})

		t.Run("異常系_存在しないIDはErrRecordNotFoundを返す", func(t *testing.T) {
			mockRepository, _, _, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			mockRepository.EXPECT().FindById(context.Background(), id).Return(nil, apperror.ErrRecordNotFound)

			ret, err := usecase.Update(context.Background(), id, NewDeckCodeUpdateParam(false, ""))

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.Nil(t, ret)
		})

		t.Run("異常系_保存失敗時はエラーを返す", func(t *testing.T) {
			mockRepository, _, _, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			existing := entity.NewDeckCode(id, time.Now().Local(), uid, deckId, code, false, "")

			mockRepository.EXPECT().FindById(context.Background(), id).Return(existing, nil)
			mockRepository.EXPECT().Save(context.Background(), gomock.Any()).Return(errors.New(""))

			ret, err := usecase.Update(context.Background(), id, NewDeckCodeUpdateParam(false, ""))

			require.Error(t, err)
			require.Nil(t, ret)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("正常系_リポジトリのDeleteを呼び出す", func(t *testing.T) {
			mockRepository, _, _, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			mockRepository.EXPECT().Delete(context.Background(), id).Return(nil)

			require.NoError(t, usecase.Delete(context.Background(), id))
		})

		t.Run("異常系_リポジトリのエラーをそのまま返す", func(t *testing.T) {
			mockRepository, _, _, usecase := setup4DeckCodeUsecase(t)

			id, err := generateId()
			require.NoError(t, err)

			mockRepository.EXPECT().Delete(context.Background(), id).Return(errors.New(""))

			require.Error(t, usecase.Delete(context.Background(), id))
		})
	})
}
