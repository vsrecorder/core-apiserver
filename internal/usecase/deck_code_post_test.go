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
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
)

// stubDeckCard は deckcard-api への問い合わせを差し替えるスタブ。
type stubDeckCard struct {
	card *entity.AceSpecCard
	err  error
}

func (s stubDeckCard) FindAceSpec(ctx context.Context, deckCode string) (*entity.AceSpecCard, error) {
	return s.card, s.err
}

type deckCodePostUsecaseMocks struct {
	post        *mock_repository.MockDeckCodePostInterface
	deck        *mock_repository.MockDeckInterface
	deckCode    *mock_repository.MockDeckCodeInterface
	user        *mock_repository.MockUserInterface
	environment *mock_repository.MockEnvironmentInterface
}

func setup4TestDeckCodePostUsecase(t *testing.T, deckCard repository.DeckCardInterface) (*deckCodePostUsecaseMocks, DeckCodePostInterface) {
	t.Helper()

	ctrl := gomock.NewController(t)
	m := &deckCodePostUsecaseMocks{
		post:        mock_repository.NewMockDeckCodePostInterface(ctrl),
		deck:        mock_repository.NewMockDeckInterface(ctrl),
		deckCode:    mock_repository.NewMockDeckCodeInterface(ctrl),
		user:        mock_repository.NewMockUserInterface(ctrl),
		environment: mock_repository.NewMockEnvironmentInterface(ctrl),
	}

	// 称号(designation)は nil にして「称号なし(0)」で通す。称号の判定は designation のテストが受け持つ。
	usecase := NewDeckCodePost(m.post, m.deck, m.deckCode, m.user, m.environment, nil, nil, deckCard)

	return m, usecase
}

func newTestDeckCodePostEntity(id string, uid string, deckCodeId string, publishedAt time.Time) *entity.DeckCodePost {
	post := entity.NewDeckCodePost(
		id, publishedAt, publishedAt, uid, "01HD7Y3K8D6FDHMHTZ2GT41TD1", deckCodeId, publishedAt,
		time.Time{}, time.Time{}, "", "", 0,
	)
	post.User = entity.NewUser(uid, publishedAt, "投稿者", "")
	post.DeckName = "テストデッキ"
	post.Code = "5dbFbk-uBwjqP-VVk5Vv"

	return post
}

func TestDeckCodePostUsecase(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	deckId := "01HD7Y3K8D6FDHMHTZ2GT41TD1"
	deckCodeId := "01HD7Y3K8D6FDHMHTZ2GT41TC1"
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)

	deckCode := entity.NewDeckCode(deckCodeId, now, uid, deckId, "5dbFbk-uBwjqP-VVk5Vv", false, "")
	deck := entity.NewDeck(deckId, now, time.Time{}, time.Time{}, uid, "テストデッキ", false, deckCode, nil)

	// DATE 列はドライバが UTC の 0 時で返すため、その形で渡す
	currentEnv := entity.NewEnvironment(
		"m6", "ストームエメラルダ",
		time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
	)
	pastEnv := entity.NewEnvironment(
		"m5", "ロケット団の栄光",
		time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC),
	)

	t.Run("Publish", func(t *testing.T) {
		t.Run("正常系_公開して投稿者付きの投稿を返す", func(t *testing.T) {
			overrideTimeNow(t, now)
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{card: &entity.AceSpecCard{
				CardId: "46232", CardName: "プライムキャッチャー", ImageURL: "https://example.com/46232.jpg",
			}})

			m.deckCode.EXPECT().FindById(gomock.Any(), deckCodeId).Return(deckCode, nil)
			m.deck.EXPECT().FindById(gomock.Any(), deckId).Return(deck, nil)
			m.post.EXPECT().FindActiveByDeckCodeId(gomock.Any(), deckCodeId).Return(nil, apperror.ErrRecordNotFound)
			m.post.EXPECT().FindLatestByDeckCodeId(gomock.Any(), deckCodeId).Return(nil, apperror.ErrRecordNotFound)

			var savedId string
			m.post.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, post *entity.DeckCodePost) error {
					savedId = post.ID
					require.Equal(t, uid, post.UserId)
					require.Equal(t, deckId, post.DeckId)
					require.Equal(t, deckCodeId, post.DeckCodeId)
					require.Equal(t, now, post.PublishedAt)
					require.True(t, post.UnpublishedAt.IsZero())
					require.Equal(t, "46232", post.AceSpecCardId)
					require.Equal(t, "プライムキャッチャー", post.AceSpecCardName)
					require.Equal(t, "https://example.com/46232.jpg", post.AceSpecImageURL)
					return nil
				},
			)
			m.post.EXPECT().FindById(gomock.Any(), gomock.Any(), uid).DoAndReturn(
				func(ctx context.Context, id string, viewer string) (*entity.DeckCodePost, error) {
					require.Equal(t, savedId, id)
					return newTestDeckCodePostEntity(id, uid, deckCodeId, now), nil
				},
			)

			post, err := usecase.Publish(context.Background(), uid, deckCodeId)

			require.NoError(t, err)
			require.Equal(t, savedId, post.ID)
			require.Equal(t, "投稿者", post.User.Name)
		})

		t.Run("正常系_deckcard-apiの失敗ではACE SPECなしとして公開する", func(t *testing.T) {
			overrideTimeNow(t, now)
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{err: errors.New("deckcard-api responded with 500")})

			m.deckCode.EXPECT().FindById(gomock.Any(), deckCodeId).Return(deckCode, nil)
			m.deck.EXPECT().FindById(gomock.Any(), deckId).Return(deck, nil)
			m.post.EXPECT().FindActiveByDeckCodeId(gomock.Any(), deckCodeId).Return(nil, apperror.ErrRecordNotFound)
			m.post.EXPECT().FindLatestByDeckCodeId(gomock.Any(), deckCodeId).Return(nil, apperror.ErrRecordNotFound)
			m.post.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, post *entity.DeckCodePost) error {
					require.Empty(t, post.AceSpecCardId)
					require.Empty(t, post.AceSpecCardName)
					return nil
				},
			)
			m.post.EXPECT().FindById(gomock.Any(), gomock.Any(), uid).DoAndReturn(
				func(ctx context.Context, id string, viewer string) (*entity.DeckCodePost, error) {
					return newTestDeckCodePostEntity(id, uid, deckCodeId, now), nil
				},
			)

			_, err := usecase.Publish(context.Background(), uid, deckCodeId)

			require.NoError(t, err)
		})

		t.Run("正常系_公開中なら既存の投稿を返し二重に作らない", func(t *testing.T) {
			overrideTimeNow(t, now)
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			existing := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now.Add(-time.Hour))
			m.deckCode.EXPECT().FindById(gomock.Any(), deckCodeId).Return(deckCode, nil)
			m.deck.EXPECT().FindById(gomock.Any(), deckId).Return(deck, nil)
			m.post.EXPECT().FindActiveByDeckCodeId(gomock.Any(), deckCodeId).Return(existing, nil)

			post, err := usecase.Publish(context.Background(), uid, deckCodeId)

			require.NoError(t, err)
			require.Equal(t, existing.ID, post.ID)
		})

		t.Run("正常系_同時に公開されて一意制約に当たったら先に公開された投稿を返す", func(t *testing.T) {
			overrideTimeNow(t, now)
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			existing := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now)
			m.deckCode.EXPECT().FindById(gomock.Any(), deckCodeId).Return(deckCode, nil)
			m.deck.EXPECT().FindById(gomock.Any(), deckId).Return(deck, nil)
			gomock.InOrder(
				m.post.EXPECT().FindActiveByDeckCodeId(gomock.Any(), deckCodeId).Return(nil, apperror.ErrRecordNotFound),
				m.post.EXPECT().FindLatestByDeckCodeId(gomock.Any(), deckCodeId).Return(nil, apperror.ErrRecordNotFound),
				m.post.EXPECT().Save(gomock.Any(), gomock.Any()).Return(apperror.ErrAlreadyExists),
				m.post.EXPECT().FindActiveByDeckCodeId(gomock.Any(), deckCodeId).Return(existing, nil),
			)

			post, err := usecase.Publish(context.Background(), uid, deckCodeId)

			require.NoError(t, err)
			require.Equal(t, existing.ID, post.ID)
		})

		t.Run("異常系_他人のデッキコードは存在しない扱いにする", func(t *testing.T) {
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			m.deckCode.EXPECT().FindById(gomock.Any(), deckCodeId).Return(deckCode, nil)

			_, err := usecase.Publish(context.Background(), "someone-else", deckCodeId)

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		})

		t.Run("異常系_他人のデッキに作ったコードは存在しない扱いにする", func(t *testing.T) {
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			othersDeck := entity.NewDeck(deckId, now, time.Time{}, time.Time{}, "someone-else", "他人のデッキ", false, deckCode, nil)
			m.deckCode.EXPECT().FindById(gomock.Any(), deckCodeId).Return(deckCode, nil)
			m.deck.EXPECT().FindById(gomock.Any(), deckId).Return(othersDeck, nil)

			_, err := usecase.Publish(context.Background(), uid, deckCodeId)

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		})

		t.Run("異常系_アーカイブ済みのデッキは公開できない", func(t *testing.T) {
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			archived := entity.NewDeck(deckId, now, now.Add(-time.Hour), time.Time{}, uid, "テストデッキ", false, deckCode, nil)
			m.deckCode.EXPECT().FindById(gomock.Any(), deckCodeId).Return(deckCode, nil)
			m.deck.EXPECT().FindById(gomock.Any(), deckId).Return(archived, nil)

			_, err := usecase.Publish(context.Background(), uid, deckCodeId)

			require.ErrorIs(t, err, apperror.ErrDeckArchived)
		})

		t.Run("異常系_24時間以内の公開し直しは拒否する", func(t *testing.T) {
			overrideTimeNow(t, now)
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			latest := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now.Add(-time.Hour))
			latest.UnpublishedAt = now.Add(-30 * time.Minute)
			m.deckCode.EXPECT().FindById(gomock.Any(), deckCodeId).Return(deckCode, nil)
			m.deck.EXPECT().FindById(gomock.Any(), deckId).Return(deck, nil)
			m.post.EXPECT().FindActiveByDeckCodeId(gomock.Any(), deckCodeId).Return(nil, apperror.ErrRecordNotFound)
			m.post.EXPECT().FindLatestByDeckCodeId(gomock.Any(), deckCodeId).Return(latest, nil)

			_, err := usecase.Publish(context.Background(), uid, deckCodeId)

			require.ErrorIs(t, err, apperror.ErrRepublishTooSoon)
		})

		t.Run("異常系_運営が非表示にしたコードは取り下げた後も公開し直せない", func(t *testing.T) {
			overrideTimeNow(t, now)
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			latest := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now.Add(-48*time.Hour))
			latest.HiddenAt = now.Add(-24 * time.Hour)
			latest.UnpublishedAt = now.Add(-2 * time.Hour)
			m.deckCode.EXPECT().FindById(gomock.Any(), deckCodeId).Return(deckCode, nil)
			m.deck.EXPECT().FindById(gomock.Any(), deckId).Return(deck, nil)
			m.post.EXPECT().FindActiveByDeckCodeId(gomock.Any(), deckCodeId).Return(nil, apperror.ErrRecordNotFound)
			m.post.EXPECT().FindLatestByDeckCodeId(gomock.Any(), deckCodeId).Return(latest, nil)

			_, err := usecase.Publish(context.Background(), uid, deckCodeId)

			require.ErrorIs(t, err, apperror.ErrDeckCodePostHidden)
		})

		t.Run("正常系_24時間を過ぎていれば公開し直せる", func(t *testing.T) {
			overrideTimeNow(t, now)
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			latest := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now.Add(-25*time.Hour))
			latest.UnpublishedAt = now.Add(-2 * time.Hour)
			m.deckCode.EXPECT().FindById(gomock.Any(), deckCodeId).Return(deckCode, nil)
			m.deck.EXPECT().FindById(gomock.Any(), deckId).Return(deck, nil)
			m.post.EXPECT().FindActiveByDeckCodeId(gomock.Any(), deckCodeId).Return(nil, apperror.ErrRecordNotFound)
			m.post.EXPECT().FindLatestByDeckCodeId(gomock.Any(), deckCodeId).Return(latest, nil)
			m.post.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
			m.post.EXPECT().FindById(gomock.Any(), gomock.Any(), uid).DoAndReturn(
				func(ctx context.Context, id string, viewer string) (*entity.DeckCodePost, error) {
					return newTestDeckCodePostEntity(id, uid, deckCodeId, now), nil
				},
			)

			_, err := usecase.Publish(context.Background(), uid, deckCodeId)

			require.NoError(t, err)
		})
	})

	t.Run("Unpublish", func(t *testing.T) {
		t.Run("正常系_取り下げ済みなら何もしない", func(t *testing.T) {
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			post := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now)
			post.UnpublishedAt = now
			m.post.EXPECT().FindLiteById(gomock.Any(), post.ID).Return(post, nil)

			require.NoError(t, usecase.Unpublish(context.Background(), post.ID))
		})

		t.Run("正常系_公開中なら取り下げる", func(t *testing.T) {
			overrideTimeNow(t, now)
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			post := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now)
			m.post.EXPECT().FindLiteById(gomock.Any(), post.ID).Return(post, nil)
			m.post.EXPECT().Unpublish(gomock.Any(), post.ID, now).Return(nil)

			require.NoError(t, usecase.Unpublish(context.Background(), post.ID))
		})

		t.Run("正常系_運営が非表示にした投稿も投稿者は取り下げられる", func(t *testing.T) {
			overrideTimeNow(t, now)
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			post := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now)
			post.HiddenAt = now
			m.post.EXPECT().FindLiteById(gomock.Any(), post.ID).Return(post, nil)
			m.post.EXPECT().Unpublish(gomock.Any(), post.ID, now).Return(nil)

			require.NoError(t, usecase.Unpublish(context.Background(), post.ID))
		})
	})

	t.Run("Like", func(t *testing.T) {
		t.Run("異常系_取り下げ済みの投稿にはいいねできない", func(t *testing.T) {
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			post := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now)
			post.UnpublishedAt = now
			m.post.EXPECT().FindLiteById(gomock.Any(), post.ID).Return(post, nil)

			_, err := usecase.Like(context.Background(), post.ID, "viewer")

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		})

		t.Run("異常系_運営が非表示にした投稿にはいいねできない", func(t *testing.T) {
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			post := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now)
			post.HiddenAt = now
			m.post.EXPECT().FindLiteById(gomock.Any(), post.ID).Return(post, nil)

			_, err := usecase.Like(context.Background(), post.ID, "viewer")

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		})

		t.Run("正常系_いいねして最新の投稿を返す", func(t *testing.T) {
			overrideTimeNow(t, now)
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			post := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now)
			liked := newTestDeckCodePostEntity(post.ID, uid, deckCodeId, now)
			liked.LikeCount = 1
			liked.LikedByMe = true
			m.post.EXPECT().FindLiteById(gomock.Any(), post.ID).Return(post, nil)
			m.post.EXPECT().Like(gomock.Any(), post.ID, "viewer", now).Return(nil)
			m.post.EXPECT().FindById(gomock.Any(), post.ID, "viewer").Return(liked, nil)

			ret, err := usecase.Like(context.Background(), post.ID, "viewer")

			require.NoError(t, err)
			require.Equal(t, 1, ret.LikeCount)
			require.True(t, ret.LikedByMe)
		})
	})

	t.Run("Unlike", func(t *testing.T) {
		t.Run("正常系_取り消して最新の投稿を返す", func(t *testing.T) {
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			post := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now)
			m.post.EXPECT().Unlike(gomock.Any(), post.ID, "viewer").Return(nil)
			m.post.EXPECT().FindById(gomock.Any(), post.ID, "viewer").Return(post, nil)

			ret, err := usecase.Unlike(context.Background(), post.ID, "viewer")

			require.NoError(t, err)
			require.False(t, ret.LikedByMe)
		})

		t.Run("異常系_存在しない投稿はNotFound", func(t *testing.T) {
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			m.post.EXPECT().Unlike(gomock.Any(), "missing", "viewer").Return(nil)
			m.post.EXPECT().FindById(gomock.Any(), "missing", "viewer").Return(nil, apperror.ErrRecordNotFound)

			_, err := usecase.Unlike(context.Background(), "missing", "viewer")

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		})
	})

	t.Run("FindByUserId", func(t *testing.T) {
		t.Run("異常系_公開中の投稿が無いユーザは存在しない扱いにする", func(t *testing.T) {
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			m.user.EXPECT().FindById(gomock.Any(), uid).Return(entity.NewUser(uid, now, "投稿者", ""), nil)
			m.post.EXPECT().SummarizeByUserId(gomock.Any(), uid).Return(&entity.DeckCodePostUserSummary{PostCount: 0}, nil)

			_, err := usecase.FindByUserId(context.Background(), uid, "viewer", 20, 0)

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		})

		t.Run("正常系_投稿者の情報と集計と投稿を返す", func(t *testing.T) {
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			post := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now)
			m.user.EXPECT().FindById(gomock.Any(), uid).Return(entity.NewUser(uid, now, "投稿者", ""), nil)
			m.post.EXPECT().SummarizeByUserId(gomock.Any(), uid).Return(&entity.DeckCodePostUserSummary{PostCount: 1, LikeCountTotal: 2}, nil)
			m.post.EXPECT().FindByUserId(gomock.Any(), uid, "viewer", 20, 0).Return([]*entity.DeckCodePost{post}, nil)

			view, err := usecase.FindByUserId(context.Background(), uid, "viewer", 20, 0)

			require.NoError(t, err)
			require.Equal(t, 1, view.Summary.PostCount)
			require.Len(t, view.Posts, 1)
		})
	})

	t.Run("FindLikers", func(t *testing.T) {
		t.Run("異常系_取り下げ済みの投稿のいいねした人は返さない", func(t *testing.T) {
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			post := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now)
			post.UnpublishedAt = now
			m.post.EXPECT().FindLiteById(gomock.Any(), post.ID).Return(post, nil)

			_, err := usecase.FindLikers(context.Background(), post.ID, 20, 0)

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		})

		t.Run("正常系_公開中の投稿のいいねした人を返す", func(t *testing.T) {
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			post := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now)
			m.post.EXPECT().FindLiteById(gomock.Any(), post.ID).Return(post, nil)
			m.post.EXPECT().FindLikers(gomock.Any(), post.ID, 20, 0).Return([]*entity.DeckCodePostLiker{
				{User: entity.NewUser("viewer", now, "閲覧者", ""), CreatedAt: now},
			}, nil)

			likers, err := usecase.FindLikers(context.Background(), post.ID, 20, 0)

			require.NoError(t, err)
			require.Len(t, likers, 1)
		})
	})

	t.Run("RecordImport", func(t *testing.T) {
		t.Run("異常系_取り下げ済みの投稿の取り込みは数えない", func(t *testing.T) {
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			post := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now)
			post.UnpublishedAt = now
			m.post.EXPECT().FindLiteById(gomock.Any(), post.ID).Return(post, nil)

			err := usecase.RecordImport(context.Background(), post.ID, "viewer")

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		})

		t.Run("正常系_公開中なら閲覧者の取り込みを記録する", func(t *testing.T) {
			overrideTimeNow(t, now)
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			post := newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now)
			m.post.EXPECT().FindLiteById(gomock.Any(), post.ID).Return(post, nil)
			m.post.EXPECT().RecordImport(gomock.Any(), post.ID, "viewer", now).Return(nil)

			require.NoError(t, usecase.RecordImport(context.Background(), post.ID, "viewer"))
		})
	})

	t.Run("Find", func(t *testing.T) {
		t.Run("正常系_環境未指定なら今日が属する環境の開始日以降で絞り終了日は区切らない", func(t *testing.T) {
			overrideTimeNow(t, now)
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			m.environment.EXPECT().FindByDate(gomock.Any(), now).Return(currentEnv, nil)
			m.post.EXPECT().Find(gomock.Any(), gomock.Any(), 20, 0).DoAndReturn(
				func(ctx context.Context, filter *repository.DeckCodePostFilter, limit int, offset int) ([]*entity.DeckCodePost, error) {
					require.Equal(t, repository.DeckCodePostSortPopular, filter.Sort)
					require.Equal(t, "viewer", filter.ViewerUserId)
					require.Equal(t, time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local), filter.From)
					require.True(t, filter.To.IsZero(), "現在の環境は次の環境が登録されるまで続くものとして終了日で区切らない")
					require.Equal(t, now.Add(-DeckCodePostPopularWindow), filter.PopularSince)
					return []*entity.DeckCodePost{newTestDeckCodePostEntity("01HD7Y3K8D6FDHMHTZ2GT41TP1", uid, deckCodeId, now)}, nil
				},
			)

			result, err := usecase.Find(context.Background(), &DeckCodePostFindParam{
				Sort: repository.DeckCodePostSortPopular, ViewerUserId: "viewer", Limit: 20, Offset: 0,
			})

			require.NoError(t, err)
			require.Equal(t, "m6", result.Environment.ID)
			require.Len(t, result.Posts, 1)
		})

		t.Run("正常系_過去の環境を指定するとその期間で絞る", func(t *testing.T) {
			overrideTimeNow(t, now)
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			m.environment.EXPECT().FindById(gomock.Any(), "m5").Return(pastEnv, nil)
			m.environment.EXPECT().FindByDate(gomock.Any(), now).Return(currentEnv, nil)
			m.post.EXPECT().Find(gomock.Any(), gomock.Any(), 10, 0).DoAndReturn(
				func(ctx context.Context, filter *repository.DeckCodePostFilter, limit int, offset int) ([]*entity.DeckCodePost, error) {
					require.Equal(t, time.Date(2026, 6, 5, 0, 0, 0, 0, time.Local), filter.From)
					require.Equal(t, time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local), filter.To)
					return []*entity.DeckCodePost{}, nil
				},
			)

			result, err := usecase.Find(context.Background(), &DeckCodePostFindParam{EnvironmentId: "m5", Limit: 10})

			require.NoError(t, err)
			require.Equal(t, "m5", result.Environment.ID)
		})

		t.Run("正常系_現在の環境を明示的に指定しても終了日で区切らない", func(t *testing.T) {
			overrideTimeNow(t, now)
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			m.environment.EXPECT().FindById(gomock.Any(), "m6").Return(currentEnv, nil)
			m.environment.EXPECT().FindByDate(gomock.Any(), now).Return(currentEnv, nil)
			m.post.EXPECT().Find(gomock.Any(), gomock.Any(), 10, 0).DoAndReturn(
				func(ctx context.Context, filter *repository.DeckCodePostFilter, limit int, offset int) ([]*entity.DeckCodePost, error) {
					require.Equal(t, time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local), filter.From)
					require.True(t, filter.To.IsZero())
					return []*entity.DeckCodePost{}, nil
				},
			)

			_, err := usecase.Find(context.Background(), &DeckCodePostFindParam{EnvironmentId: "m6", Limit: 10})

			require.NoError(t, err)
		})

		t.Run("正常系_今日に対応する環境が無ければ期間で絞らない", func(t *testing.T) {
			overrideTimeNow(t, now)
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			m.environment.EXPECT().FindByDate(gomock.Any(), now).Return(nil, apperror.ErrRecordNotFound)
			m.post.EXPECT().Find(gomock.Any(), gomock.Any(), 10, 0).DoAndReturn(
				func(ctx context.Context, filter *repository.DeckCodePostFilter, limit int, offset int) ([]*entity.DeckCodePost, error) {
					require.True(t, filter.From.IsZero())
					require.True(t, filter.To.IsZero())
					return []*entity.DeckCodePost{}, nil
				},
			)

			result, err := usecase.Find(context.Background(), &DeckCodePostFindParam{Limit: 10})

			require.NoError(t, err)
			require.Nil(t, result.Environment)
		})

		t.Run("異常系_指定した環境が無ければNotFound", func(t *testing.T) {
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			m.environment.EXPECT().FindById(gomock.Any(), "zz").Return(nil, apperror.ErrRecordNotFound)

			_, err := usecase.Find(context.Background(), &DeckCodePostFindParam{EnvironmentId: "zz", Limit: 10})

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		})
	})

	t.Run("FindAceSpecCounts", func(t *testing.T) {
		t.Run("正常系_一覧と同じ期間で候補を引く", func(t *testing.T) {
			overrideTimeNow(t, now)
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			m.environment.EXPECT().FindByDate(gomock.Any(), now).Return(currentEnv, nil)
			m.post.EXPECT().FindAceSpecCounts(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, filter *repository.DeckCodePostFilter) ([]*entity.DeckCodePostAceSpecCount, error) {
					require.Equal(t, time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local), filter.From)
					require.True(t, filter.To.IsZero(), "現在の環境は終了日で区切らない")
					return []*entity.DeckCodePostAceSpecCount{
						{CardName: "アンフェアスタンプ", ImageURL: "https://example.com/47870.jpg", Count: 3},
					}, nil
				},
			)

			result, err := usecase.FindAceSpecCounts(context.Background(), "")

			require.NoError(t, err)
			require.Equal(t, "m6", result.Environment.ID)
			require.Len(t, result.AceSpecs, 1)
			require.Equal(t, "アンフェアスタンプ", result.AceSpecs[0].CardName)
		})

		t.Run("異常系_指定した環境が無ければNotFound", func(t *testing.T) {
			m, usecase := setup4TestDeckCodePostUsecase(t, stubDeckCard{})

			m.environment.EXPECT().FindById(gomock.Any(), "zz").Return(nil, apperror.ErrRecordNotFound)

			_, err := usecase.FindAceSpecCounts(context.Background(), "zz")

			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		})
	})

}
