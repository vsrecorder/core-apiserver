package validation

import (
	"regexp"

	"github.com/gin-gonic/gin"

	"github.com/vsrecorder/core-apiserver/internal/controller/apierror"
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/controller/helper"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	// MaxDeckCodePostLimit はみんなの公開デッキ関連の一覧で1回に返す最大件数。
	// 未ログインでも叩ける公開APIのため、limit の指定に上限を設ける。
	MaxDeckCodePostLimit = 50

	// ulidLength は ULID の文字数(deck_code_id / 投稿ID の検証に使う)。
	ulidLength = 26
)

// aceSpecCardIdPattern はカードID(公式サイトの数値ID)の形。
var aceSpecCardIdPattern = regexp.MustCompile(`^[0-9]{1,16}$`)

// environmentIdPattern は environments.id(英数字8文字以内)の形。
var environmentIdPattern = regexp.MustCompile(`^[0-9A-Za-z_-]{1,8}$`)

// pokemonSpriteIdPattern は pokemon_sprites.id("0887" / "0006_mega_x" など。128文字以内)の形。
var pokemonSpriteIdPattern = regexp.MustCompile(`^[0-9A-Za-z_.-]{1,128}$`)

// MaxDeckCodePostSpriteFilters はスプライトでの絞り込みに指定できる数。デッキのスプライトと同じく2体まで。
const MaxDeckCodePostSpriteFilters = 2

// parseEnvironmentId は environment_id クエリを検証して返す(空は「現在の環境」)。
func parseEnvironmentId(ctx *gin.Context) (string, bool) {
	environmentId := helper.GetQueryEnvironmentId(ctx)
	if environmentId != "" && !environmentIdPattern.MatchString(environmentId) {
		apierror.ErrBadRequest.JSON(ctx)
		return "", false
	}

	return environmentId, true
}

// parsePokemonSpriteIds は pokemon_sprite_id クエリ(繰り返し可)を検証し、重複と空を除いて返す。
// 3件以上や形式外の値は 400。
func parsePokemonSpriteIds(ctx *gin.Context) ([]string, bool) {
	raw := ctx.QueryArray("pokemon_sprite_id")

	seen := make(map[string]struct{}, len(raw))
	ids := make([]string, 0, len(raw))
	for _, id := range raw {
		if id == "" {
			continue
		}
		if !pokemonSpriteIdPattern.MatchString(id) {
			apierror.ErrBadRequest.JSON(ctx)
			return nil, false
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) > MaxDeckCodePostSpriteFilters {
		apierror.ErrBadRequest.JSON(ctx)
		return nil, false
	}

	return ids, true
}

// capDeckCodePostLimit は公開APIの limit を上限で切り詰める。
func capDeckCodePostLimit(limit int) int {
	if limit > MaxDeckCodePostLimit {
		return MaxDeckCodePostLimit
	}

	return limit
}

func parseDeckCodePostPaging(ctx *gin.Context) bool {
	limit, err := helper.ParseQueryLimit(ctx)
	if err != nil {
		apierror.ErrBadRequest.JSON(ctx, err)
		return false
	}

	offset, err := helper.ParseQueryOffset(ctx)
	if err != nil {
		apierror.ErrBadRequest.JSON(ctx, err)
		return false
	}

	helper.SetLimit(ctx, capDeckCodePostLimit(limit))
	helper.SetOffset(ctx, offset)

	return true
}

// DeckCodePostGetMiddleware は GET /deck_code_posts のクエリを検証する。
//   - sort: "" / new / popular
//   - environment_id: 環境ID(空なら現在の環境)
//   - acespec_card_id: ACE SPEC カードID(空なら絞り込みなし)
//   - pokemon_sprite_id: スプライトID(繰り返し指定で最大2体。すべてを持つデッキに絞る)
func DeckCodePostGetMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !parseDeckCodePostPaging(ctx) {
			return
		}

		sort := ctx.Query("sort")
		switch sort {
		case "":
			sort = repository.DeckCodePostSortNew
		case repository.DeckCodePostSortNew, repository.DeckCodePostSortPopular:
		default:
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		environmentId, ok := parseEnvironmentId(ctx)
		if !ok {
			return
		}

		aceSpecCardId := ctx.Query("acespec_card_id")
		if aceSpecCardId != "" && !aceSpecCardIdPattern.MatchString(aceSpecCardId) {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		pokemonSpriteIds, ok := parsePokemonSpriteIds(ctx)
		if !ok {
			return
		}

		helper.SetSort(ctx, sort)
		helper.SetEnvironmentId(ctx, environmentId)
		helper.SetAceSpecCardId(ctx, aceSpecCardId)
		helper.SetPokemonSpriteIds(ctx, pokemonSpriteIds)
	}
}

// DeckCodePostPagingMiddleware はいいねした人一覧・投稿者ページの limit / offset を検証する。
func DeckCodePostPagingMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		parseDeckCodePostPaging(ctx)
	}
}

func DeckCodePostCreateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := dto.DeckCodePostCreateRequest{}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			apierror.ErrBadRequest.JSON(ctx, err)
			return
		}

		if len(req.DeckCodeId) != ulidLength {
			apierror.ErrBadRequest.JSON(ctx)
			return
		}

		helper.SetDeckCodePostCreateRequest(ctx, req)
	}
}
