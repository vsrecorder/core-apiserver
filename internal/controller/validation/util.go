package validation

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
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
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusServiceUnavailable:
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"message": "service unavailable"})
			ctx.Abort()
			return
		case http.StatusGatewayTimeout:
			ctx.JSON(http.StatusGatewayTimeout, gin.H{"message": "gateway timeout"})
			ctx.Abort()
			return
		default:
			ctx.JSON(http.StatusBadGateway, gin.H{"message": "bad gateway", "status": resp.Status})
			ctx.Abort()
			return
		}
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	var res DeckIDCheckResponse
	if err := json.Unmarshal(body, &res); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	if res.Existence == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
		ctx.Abort()
		return
	}
}
