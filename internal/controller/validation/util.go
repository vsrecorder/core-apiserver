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

	if resp.StatusCode != http.StatusOK {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		ctx.Abort()
		return
	}

	defer resp.Body.Close()

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
