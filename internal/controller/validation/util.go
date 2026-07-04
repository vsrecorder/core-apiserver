package validation

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
)

const (
	DateLayout = time.DateOnly
)

type DeckIDCheckResponse struct {
	Result    int    `json:"result"`
	ErrMsg    string `json:"errMsg"`
	DeckID    string `json:"deckID"`
	Existence int    `json:"existence"`
}

func checkDeckCode(ctx *gin.Context, deckCode string) {
	data := url.Values{}
	data.Add("deckID", deckCode)

	resp, err := http.PostForm("https://www.pokemon-card.com/deck/deckIDCheck.php", data)

	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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

type PlayerAccountOtherResponse struct {
	Code   int `json:"code"`
	Player *struct {
		PlayerId string `json:"player_id"`
		Nickname string `json:"nickname"`
	} `json:"player"`
}

func checkPlayerId(ctx *gin.Context, playerId string) {
	data := url.Values{}
	data.Add("player_id", playerId)

	resp, err := http.PostForm("https://players.pokemon-card.com/get_player_account_other", data)

	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	defer resp.Body.Close()

	// このAPIは player_id が存在しない/マイページが非公開の場合、200以外の
	// ステータス(404など)とともに {"code":404,"message":"..."} 形式のJSONを返す。
	// そのため一律にステータスコードだけでサーバエラー扱いにはせず、まずボディを
	// 読んでJSONの code / player フィールドで存在確認を行う。
	switch resp.StatusCode {
	case http.StatusServiceUnavailable:
		apierror.ErrServiceUnavailable.JSON(ctx)
		return
	case http.StatusGatewayTimeout:
		apierror.ErrGatewayTimeout.JSON(ctx)
		return
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	var res PlayerAccountOtherResponse
	if err := json.Unmarshal(body, &res); err != nil {
		apierror.ErrInternalServerError.JSON(ctx)
		return
	}

	if res.Code != http.StatusOK || res.Player == nil {
		apierror.ErrBadRequest.JSON(ctx)
		return
	}
}
