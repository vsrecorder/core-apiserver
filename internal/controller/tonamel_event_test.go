package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/mock/mock_usecase"
)

func setupMock4TestTonamelEventController(t *testing.T) *mock_usecase.MockTonamelEventInterface {
	mockCtrl := gomock.NewController(t)
	mockUsecase := mock_usecase.NewMockTonamelEventInterface(mockCtrl)

	return mockUsecase
}

func setup4TestTonamelEventController(t *testing.T, r *gin.Engine) (
	*TonamelEvent,
	*mock_usecase.MockTonamelEventInterface,
) {
	mockUsecase := setupMock4TestTonamelEventController(t)

	c := NewTonamelEvent(r, mockUsecase)
	c.RegisterRoute("")

	return c, mockUsecase
}

func TestTonamelEventController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for scenario, fn := range map[string]func(t *testing.T){
		"GetById": test_TonamelEventController_GetById,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_TonamelEventController_GetById(t *testing.T) {
	r := gin.Default()
	c, mockUsecase := setup4TestTonamelEventController(t, r)

	t.Run("正常系_指定IDのTonamelイベントを返す", func(t *testing.T) {
		id := "61ozP"

		tonamelEvent := &entity.TonamelEvent{
			ID: id,
		}

		mockUsecase.EXPECT().FindById(gomock.Any(), id).Return(tonamelEvent, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", TonamelEventsPath+"/"+id, nil)
		c.router.ServeHTTP(w, req)

		var res dto.TonamelEventGetByIdResponse
		json.Unmarshal(w.Body.Bytes(), &res)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, id, res.ID)
	})

	t.Run("異常系_ユースケースのエラーで500を返す", func(t *testing.T) {
		id := "61ozP"

		mockUsecase.EXPECT().FindById(gomock.Any(), id).Return(nil, errors.New(""))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", TonamelEventsPath+"/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	// 同時取得数の上限に当たった(一時的な状態)ときは 503 にして、クライアントに再試行を促す。
	t.Run("異常系_外部取得が混み合っていれば503を返す", func(t *testing.T) {
		id := "61ozP"

		mockUsecase.EXPECT().FindById(gomock.Any(), id).Return(nil, apperror.ErrExternalFetchBusy)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", TonamelEventsPath+"/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	// IDはそのまま tonamel.com のURLに連結するため、形式外の値は外部サイトへ問い合わせる前に弾く。
	t.Run("異常系_形式外のIDは400を返しユースケースを呼ばない", func(t *testing.T) {
		id := "123456789" // 9文字(上限8文字を超える)

		// FindById は EXPECT しない(バリデーションで弾かれ、呼ばれない)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", TonamelEventsPath+"/"+id, nil)
		c.router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}
