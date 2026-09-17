package validation

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
)

// 参照先IDが列幅(VARCHAR)を超える値は、保存時のDBエラー(500)ではなく入口で 400 にする。
func TestReferenceIdLengthValidation(t *testing.T) {
	tooLongEntityId := strings.Repeat("A", MaxEntityIdLength+1)
	tooLongUserId := strings.Repeat("A", MaxUserIdLength+1)

	t.Run("RecordCreateMiddleware", func(t *testing.T) {
		cases := map[string]dto.RecordRequest{
			"friend_id":           {EventDate: testValidationEventDate, FriendId: tooLongUserId},
			"unofficial_event_id": {EventDate: testValidationEventDate, UnofficialEventId: tooLongEntityId},
			"deck_id":             {EventDate: testValidationEventDate, OfficialEventId: 1, DeckId: tooLongEntityId},
			"deck_code_id":        {EventDate: testValidationEventDate, OfficialEventId: 1, DeckCodeId: tooLongEntityId},
		}

		for field, req := range cases {
			t.Run("異常系_"+field+"が列幅を超えると400を返す", func(t *testing.T) {
				b, err := json.Marshal(dto.RecordCreateRequest{RecordRequest: req})
				require.NoError(t, err)

				ctx, w := newValidationJSONContext(t, string(b))

				RecordCreateMiddleware()(ctx)

				require.Equal(t, http.StatusBadRequest, w.Code)
			})
		}

		t.Run("正常系_列幅内のIDは受理する", func(t *testing.T) {
			b, err := json.Marshal(dto.RecordCreateRequest{RecordRequest: dto.RecordRequest{
				EventDate: testValidationEventDate,
				FriendId:  strings.Repeat("A", MaxUserIdLength),
				DeckId:    strings.Repeat("A", MaxEntityIdLength),
			}})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			RecordCreateMiddleware()(ctx)

			require.Equal(t, http.StatusOK, w.Code)
		})
	})

	t.Run("MatchCreateMiddleware", func(t *testing.T) {
		games := []*dto.GameRequest{{GoFirst: false, WinningFlg: false}}
		cases := map[string]dto.MatchRequest{
			"record_id":         {RecordId: tooLongEntityId, Games: games},
			"deck_id":           {RecordId: "01HD7Y3K8D6FDHMHTZ2GT41TR1", DeckId: tooLongEntityId, Games: games},
			"deck_code_id":      {RecordId: "01HD7Y3K8D6FDHMHTZ2GT41TR1", DeckCodeId: tooLongEntityId, Games: games},
			"opponents_user_id": {RecordId: "01HD7Y3K8D6FDHMHTZ2GT41TR1", OpponentsUserId: tooLongUserId, Games: games},
		}

		for field, req := range cases {
			t.Run("異常系_"+field+"が列幅を超えると400を返す", func(t *testing.T) {
				b, err := json.Marshal(dto.MatchCreateRequest{MatchRequest: req})
				require.NoError(t, err)

				ctx, w := newValidationJSONContext(t, string(b))

				MatchCreateMiddleware()(ctx)

				require.Equal(t, http.StatusBadRequest, w.Code)
			})
		}

		t.Run("異常系_スプライトのpositionが重複していると400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.MatchCreateRequest{MatchRequest: dto.MatchRequest{
				RecordId:       "01HD7Y3K8D6FDHMHTZ2GT41TR1",
				Games:          games,
				PokemonSprites: []*dto.PokemonSpriteRequest{{ID: "0887", Position: 1}, {ID: "0006", Position: 1}},
			}})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			MatchCreateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("DeckCreateMiddleware", func(t *testing.T) {
		t.Run("異常系_スプライトが3体以上なら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.DeckCreateRequest{
				Name:           "test",
				PokemonSprites: []*dto.PokemonSpriteRequest{{ID: "0001"}, {ID: "0002"}, {ID: "0003"}},
			})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCreateMiddleware(slog.Default())(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("正常系_2体までのスプライトは受理する", func(t *testing.T) {
			b, err := json.Marshal(dto.DeckCreateRequest{
				Name:           "test",
				PokemonSprites: []*dto.PokemonSpriteRequest{{ID: "0887", Position: 1}, {ID: "0006_mega_x", Position: 2}},
			})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckCreateMiddleware(slog.Default())(ctx)

			require.Equal(t, http.StatusOK, w.Code)
		})
	})

	t.Run("DeckUpdateMiddleware", func(t *testing.T) {
		t.Run("異常系_スプライトIDが形式外なら400を返す", func(t *testing.T) {
			b, err := json.Marshal(dto.DeckUpdateRequest{
				Name:           "test",
				PokemonSprites: []*dto.PokemonSpriteRequest{{ID: "0887 x", Position: 1}},
			})
			require.NoError(t, err)

			ctx, w := newValidationJSONContext(t, string(b))

			DeckUpdateMiddleware()(ctx)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})
}
