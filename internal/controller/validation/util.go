package validation

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
)

const (
	DateLayout = time.DateOnly

	DeckIDCheckURL = "https://www.pokemon-card.com/deck/deckIDCheck.php"
)

type DeckIDCheckResponse struct {
	Result    int    `json:"result"`
	ErrMsg    string `json:"errMsg"`
	DeckID    string `json:"deckID"`
	Existence int    `json:"existence"`
}

// deckCodeCheckLogAttrs はデッキコード確認APIのログに共通で付与する属性を返す。
// request_idはアクセスログと突き合わせるためにRequestIDMiddlewareが設定した値を利用する。
func deckCodeCheckLogAttrs(ctx *gin.Context, deckCode string) []any {
	requestID, _ := ctx.Get("request_id")
	requestIDStr, _ := requestID.(string)

	return []any{
		slog.String("request_id", requestIDStr),
		slog.String("deck_code", deckCode),
		slog.String("request_url", DeckIDCheckURL),
	}
}

func checkDeckCode(ctx *gin.Context, logger *slog.Logger, deckCode string) {
	data := url.Values{}
	data.Add("deckID", deckCode)

	resp, err := http.PostForm(DeckIDCheckURL, data)

	if err != nil {
		logger.ErrorContext(
			ctx.Request.Context(),
			"failed to request deck code check API",
			append(
				deckCodeCheckLogAttrs(ctx, deckCode),
				slog.String("error_message", err.Error()),
			)...,
		)

		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.ErrorContext(
			ctx.Request.Context(),
			"deck code check API returned non-200 status",
			append(
				deckCodeCheckLogAttrs(ctx, deckCode),
				slog.Int("status_code", resp.StatusCode),
			)...,
		)

		switch resp.StatusCode {
		case http.StatusServiceUnavailable:
			apierror.ErrServiceUnavailable.JSON(ctx)
			return
		case http.StatusGatewayTimeout:
			apierror.ErrGatewayTimeout.JSON(ctx)
			return
		default:
			apierror.ErrBadGateway.JSON(ctx)
			return
		}
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	var res DeckIDCheckResponse
	if err := json.Unmarshal(body, &res); err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	if res.Existence == 0 {
		apierror.ErrBadRequest.JSON(ctx)
		return
	}
}
