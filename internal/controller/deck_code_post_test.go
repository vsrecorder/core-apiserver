package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/controller/auth/authentication"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_repository"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
	"github.com/vsrecorder/core-apiserver/internal/testutil"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func setup4TestDeckCodePostController(t *testing.T) (
	*DeckCodePost,
	*mock_repository.MockDeckCodePostInterface,
	*mock_repository.MockDeckInterface,
	*mock_usecase.MockDeckCodePostInterface,
	string,
) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	secretKey, err := testutil.GenerateJWTSecret()
	require.NoError(t, err)
	t.Setenv("VSRECORDER_JWT_SECRET", secretKey)

	ctrl := gomock.NewController(t)
	mockRepository := mock_repository.NewMockDeckCodePostInterface(ctrl)
	mockDeckRepository := mock_repository.NewMockDeckInterface(ctrl)
	mockUsecase := mock_usecase.NewMockDeckCodePostInterface(ctrl)

	r := gin.Default()
	c := NewDeckCodePost(r, mockRepository, mockDeckRepository, mockUsecase)
	c.RegisterRoute("")

	return c, mockRepository, mockDeckRepository, mockUsecase, secretKey
}

func newTestDeckCodePost(id string, uid string) *entity.DeckCodePost {
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	post := entity.NewDeckCodePost(
		id, now, now, uid, "01HD7Y3K8D6FDHMHTZ2GT41TD1", "01HD7Y3K8D6FDHMHTZ2GT41TC1", now,
		time.Time{}, time.Time{}, "46232", "プライムキャッチャー", 3,
	)
	post.User = entity.NewUser(uid, now, "投稿者", "https://example.com/icon.png")
	post.DeckName = "テストデッキ"
	post.Code = "5dbFbk-uBwjqP-VVk5Vv"
	post.PokemonSprites = []*entity.PokemonSprite{entity.NewPokemonSpriteWithPosition("0887", 1)}
	post.RecentLikers = []*entity.User{entity.NewUser("liker", now, "いいねした人", "")}
	post.DesignationTier = 7

	return post
}

func TestDeckCodePostController(t *testing.T) {
	uid := "zor5SLfEfwfZ90yRVXzlxBEFARy2"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TP1"

	t.Run("Get", func(t *testing.T) {
		t.Run("正常系_未認証でも一覧を参照できる", func(t *testing.T) {
			c, _, _, mockUsecase, _ := setup4TestDeckCodePostController(t)

			mockUsecase.EXPECT().Find(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx any, param *usecase.DeckCodePostFindParam) (*usecase.DeckCodePostFindResult, error) {
					require.Equal(t, "popular", param.Sort)
					require.Equal(t, "m6", param.EnvironmentId)
					require.Equal(t, "", param.ViewerUserId)
					require.Equal(t, 50, param.Limit, "上限を超える limit は切り詰める")
					return &usecase.DeckCodePostFindResult{
						Environment: entity.NewEnvironment("m6", "ストームエメラルダ", time.Time{}, time.Time{}),
						Posts:       []*entity.DeckCodePost{newTestDeckCodePost(id, uid)},
					}, nil
				},
			)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DeckCodePostsPath+"?sort=popular&environment_id=m6&limit=100", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)

			var res dto.DeckCodePostGetResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
			require.Equal(t, "m6", res.Environment.ID)
			require.Len(t, res.Posts, 1)
			require.Equal(t, "テストデッキ", res.Posts[0].DeckName)
			require.Equal(t, 7, res.Posts[0].User.DesignationTier)
			require.Len(t, res.Posts[0].RecentLikers, 1)
		})

		t.Run("異常系_並び順が不正なら400", func(t *testing.T) {
			c, _, _, _, _ := setup4TestDeckCodePostController(t)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DeckCodePostsPath+"?sort=oldest", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("GetById", func(t *testing.T) {
		t.Run("正常系_公開中の投稿を返す", func(t *testing.T) {
			c, _, _, mockUsecase, _ := setup4TestDeckCodePostController(t)

			mockUsecase.EXPECT().FindById(gomock.Any(), id, "").Return(newTestDeckCodePost(id, uid), nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DeckCodePostsPath+"/"+id, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("異常系_取り下げ済みの投稿は410", func(t *testing.T) {
			c, _, _, mockUsecase, _ := setup4TestDeckCodePostController(t)

			post := newTestDeckCodePost(id, uid)
			post.UnpublishedAt = time.Now()
			mockUsecase.EXPECT().FindById(gomock.Any(), id, "").Return(post, nil)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DeckCodePostsPath+"/"+id, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusGone, w.Code)
		})

		t.Run("正常系_運営が非表示にした投稿は投稿者本人にはhidden付きで返す", func(t *testing.T) {
			c, _, _, mockUsecase, secretKey := setup4TestDeckCodePostController(t)

			post := newTestDeckCodePost(id, uid)
			post.HiddenAt = time.Now()
			mockUsecase.EXPECT().FindById(gomock.Any(), id, uid).Return(post, nil)

			token, err := testutil.GenerateJWT(uid, secretKey, authentication.ExpectedIssuer)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DeckCodePostsPath+"/"+id, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)

			var res dto.DeckCodePostGetByIdResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
			require.True(t, res.Hidden)
		})

		t.Run("異常系_運営が非表示にした投稿は本人以外には410", func(t *testing.T) {
			c, _, _, mockUsecase, secretKey := setup4TestDeckCodePostController(t)

			post := newTestDeckCodePost(id, uid)
			post.HiddenAt = time.Now()
			mockUsecase.EXPECT().FindById(gomock.Any(), id, "someone-else").Return(post, nil)

			token, err := testutil.GenerateJWT("someone-else", secretKey, authentication.ExpectedIssuer)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DeckCodePostsPath+"/"+id, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusGone, w.Code)
		})

		t.Run("異常系_存在しない投稿は404", func(t *testing.T) {
			c, _, _, mockUsecase, _ := setup4TestDeckCodePostController(t)

			mockUsecase.EXPECT().FindById(gomock.Any(), id, "").Return(nil, apperror.ErrRecordNotFound)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DeckCodePostsPath+"/"+id, nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNotFound, w.Code)
		})
	})

	t.Run("Create", func(t *testing.T) {
		t.Run("異常系_未認証では公開できない", func(t *testing.T) {
			c, _, _, _, _ := setup4TestDeckCodePostController(t)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", DeckCodePostsPath, strings.NewReader(`{"deck_code_id":"01HD7Y3K8D6FDHMHTZ2GT41TC1"}`))
			req.Header.Set("Content-Type", "application/json")
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("正常系_公開すると201で投稿を返す", func(t *testing.T) {
			c, _, _, mockUsecase, secretKey := setup4TestDeckCodePostController(t)

			mockUsecase.EXPECT().Publish(gomock.Any(), uid, "01HD7Y3K8D6FDHMHTZ2GT41TC1").Return(newTestDeckCodePost(id, uid), nil)

			token, err := testutil.GenerateJWT(uid, secretKey, authentication.ExpectedIssuer)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", DeckCodePostsPath, strings.NewReader(`{"deck_code_id":"01HD7Y3K8D6FDHMHTZ2GT41TC1"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusCreated, w.Code)

			var res dto.DeckCodePostCreateResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
			require.Equal(t, id, res.ID)
		})

		t.Run("異常系_24時間以内の公開し直しは429", func(t *testing.T) {
			c, _, _, mockUsecase, secretKey := setup4TestDeckCodePostController(t)

			mockUsecase.EXPECT().Publish(gomock.Any(), uid, "01HD7Y3K8D6FDHMHTZ2GT41TC1").Return(nil, apperror.ErrRepublishTooSoon)

			token, err := testutil.GenerateJWT(uid, secretKey, authentication.ExpectedIssuer)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", DeckCodePostsPath, strings.NewReader(`{"deck_code_id":"01HD7Y3K8D6FDHMHTZ2GT41TC1"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusTooManyRequests, w.Code)
		})

		t.Run("異常系_deck_code_idの形式が不正なら400", func(t *testing.T) {
			c, _, _, _, secretKey := setup4TestDeckCodePostController(t)

			token, err := testutil.GenerateJWT(uid, secretKey, authentication.ExpectedIssuer)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", DeckCodePostsPath, strings.NewReader(`{"deck_code_id":"short"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("異常系_他人の投稿は取り下げられない", func(t *testing.T) {
			c, mockRepository, _, _, secretKey := setup4TestDeckCodePostController(t)

			mockRepository.EXPECT().FindLiteById(gomock.Any(), id).Return(newTestDeckCodePost(id, "someone-else"), nil)

			token, err := testutil.GenerateJWT(uid, secretKey, authentication.ExpectedIssuer)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", DeckCodePostsPath+"/"+id, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusForbidden, w.Code)
		})

		t.Run("正常系_本人なら取り下げて204", func(t *testing.T) {
			c, mockRepository, _, mockUsecase, secretKey := setup4TestDeckCodePostController(t)

			mockRepository.EXPECT().FindLiteById(gomock.Any(), id).Return(newTestDeckCodePost(id, uid), nil)
			mockUsecase.EXPECT().Unpublish(gomock.Any(), id).Return(nil)

			token, err := testutil.GenerateJWT(uid, secretKey, authentication.ExpectedIssuer)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", DeckCodePostsPath+"/"+id, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNoContent, w.Code)
		})
	})

	t.Run("Like", func(t *testing.T) {
		t.Run("異常系_未認証ではいいねできない", func(t *testing.T) {
			c, _, _, _, _ := setup4TestDeckCodePostController(t)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", DeckCodePostsPath+"/"+id+"/like", nil)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("正常系_いいねすると更新後の投稿を返す", func(t *testing.T) {
			c, _, _, mockUsecase, secretKey := setup4TestDeckCodePostController(t)

			liked := newTestDeckCodePost(id, "author")
			liked.LikedByMe = true
			mockUsecase.EXPECT().Like(gomock.Any(), id, uid).Return(liked, nil)

			token, err := testutil.GenerateJWT(uid, secretKey, authentication.ExpectedIssuer)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", DeckCodePostsPath+"/"+id+"/like", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)

			var res dto.DeckCodePostLikeResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
			require.True(t, res.LikedByMe)
		})
	})

	t.Run("GetByDeckId", func(t *testing.T) {
		t.Run("異常系_他人のデッキの投稿一覧は参照できない", func(t *testing.T) {
			c, _, mockDeckRepository, _, secretKey := setup4TestDeckCodePostController(t)

			deckId := "01HD7Y3K8D6FDHMHTZ2GT41TD1"
			mockDeckRepository.EXPECT().FindById(gomock.Any(), deckId).Return(
				entity.NewDeck(deckId, time.Now(), time.Time{}, time.Time{}, "someone-else", "デッキ", false, nil, nil), nil,
			)

			token, err := testutil.GenerateJWT(uid, secretKey, authentication.ExpectedIssuer)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", DecksPath+"/"+deckId+DeckCodePostsPath, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			c.router.ServeHTTP(w, req)

			require.Equal(t, http.StatusForbidden, w.Code)
		})
	})
}
