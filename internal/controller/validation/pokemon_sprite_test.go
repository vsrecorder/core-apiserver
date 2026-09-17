package validation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
)

// 保存側は position が主キー、pokemon_sprite_id が外部キーなので、形式外の入力は
// DB エラー(500)になる前に入口で弾く。
func TestValidatePokemonSprites(t *testing.T) {
	sprite := func(id string, position uint) *dto.PokemonSpriteRequest {
		return &dto.PokemonSpriteRequest{ID: id, Position: position}
	}

	tests := []struct {
		name    string
		sprites []*dto.PokemonSpriteRequest
		want    bool
	}{
		{"正常系_未指定", nil, true},
		{"正常系_positionを指定した2体", []*dto.PokemonSpriteRequest{sprite("0887", 1), sprite("0006_mega_x", 2)}, true},
		{"正常系_positionを省略した2体(並び順で採番)", []*dto.PokemonSpriteRequest{sprite("0887", 0), sprite("0006", 0)}, true},
		{"正常系_2体目の枠だけ指定", []*dto.PokemonSpriteRequest{sprite("0887", 2)}, true},

		{"異常系_3体以上", []*dto.PokemonSpriteRequest{sprite("0001", 0), sprite("0002", 0), sprite("0003", 0)}, false},
		{"異常系_nil要素", []*dto.PokemonSpriteRequest{nil}, false},
		{"異常系_IDが空", []*dto.PokemonSpriteRequest{sprite("", 1)}, false},
		{"異常系_IDに形式外の文字", []*dto.PokemonSpriteRequest{sprite("0887/../x", 1)}, false},
		{"異常系_IDが128文字超", []*dto.PokemonSpriteRequest{sprite(string(make([]byte, 129)), 1)}, false},
		{"異常系_positionが枠数を超える", []*dto.PokemonSpriteRequest{sprite("0887", 3)}, false},
		{"異常系_positionの重複(主キー違反になる)", []*dto.PokemonSpriteRequest{sprite("0887", 1), sprite("0006", 1)}, false},
		{"異常系_指定と省略の混在(採番と衝突しうる)", []*dto.PokemonSpriteRequest{sprite("0887", 0), sprite("0006", 1)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, validatePokemonSprites(tt.sprites))
		})
	}
}
