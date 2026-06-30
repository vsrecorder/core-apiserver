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
