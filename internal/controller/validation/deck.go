package validation

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
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

func DeckGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		limit, err := helper.ParseQueryLimit(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		offset, err := helper.ParseQueryOffset(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		cursor, err := helper.ParseQueryCursor(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		archived, err := helper.ParseQueryArchive(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		helper.SetLimit(ctx, limit)
		helper.SetOffset(ctx, offset)
		helper.SetCursor(ctx, cursor)
		helper.SetArchived(ctx, archived)
	}
}

func DeckCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.DeckCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		if req.Name == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		if req.Code != "" {
			checkDeckCode(ctx, req.Code)
		}

		helper.SetDeckCreateRequest(ctx, req)
	}
}

func DeckUpdateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.DeckUpdateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		if req.Name == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "bad request"})
			ctx.Abort()
			return
		}

		if req.Code != "" {
			checkDeckCode(ctx, req.Code)
		}

		helper.SetDeckUpdateRequest(ctx, req)
	}
}
