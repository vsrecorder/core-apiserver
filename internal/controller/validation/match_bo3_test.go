package validation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
)

// BO3(2本先取)を中心に、Match作成/更新リクエストの整合性検証を網羅する。
//
// 検証内容は作成/更新で同一であるべきなので、全ケースを
// MatchCreateMiddleware と MatchUpdateMiddleware の両方に流して
// 同じ結果になることを確認する。

// game は勝敗のみを指定してGameRequestを組み立てる(先攻/後攻は検証に影響しない)。
func game(winningFlg bool) *dto.GameRequest {
	return &dto.GameRequest{
		GoFirst:             true,
		WinningFlg:          winningFlg,
		YourPrizeCards:      0,
		OpponentsPrizeCards: 0,
		Memo:                "",
	}
}

// runMatchMiddleware はMiddlewareにリクエストを流し、HTTPステータスを返す。
// バリデーションを通過した場合は何も書き込まれないため200になる。
func runMatchMiddleware(t *testing.T, middleware gin.HandlerFunc, req dto.MatchRequest) int {
	t.Helper()

	w := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(w)

	dataBytes, err := json.Marshal(req)
	require.NoError(t, err)

	// Middlewareのテストのためpathは何でもよい
	httpReq, err := http.NewRequest("POST", "/", strings.NewReader(string(dataBytes)))
	require.NoError(t, err)

	ginContext.Request = httpReq

	middleware(ginContext)

	return w.Code
}

func TestMatchValidationBO3(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const recordId = "01JMPK4VF04QX714CG4PHYJ88K"

	tests := []struct {
		name  string
		req   dto.MatchRequest
		valid bool
	}{
		// --- BO3: 2ゲームで決着(2-0 / 0-2) ---
		{
			name: "正常系_BO3_2ゲーム_2連勝で勝利",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: true,
				Games: []*dto.GameRequest{game(true), game(true)},
			},
			valid: true,
		},
		{
			name: "正常系_BO3_2ゲーム_2連敗で敗北",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: false,
				Games: []*dto.GameRequest{game(false), game(false)},
			},
			valid: true,
		},
		{
			name: "異常系_BO3_2ゲーム_2連勝なのに敗北になっている",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: false,
				Games: []*dto.GameRequest{game(true), game(true)},
			},
			valid: false,
		},
		{
			name: "異常系_BO3_2ゲーム_1勝1敗なのに2ゲームで決着している",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: true,
				Games: []*dto.GameRequest{game(true), game(false)},
			},
			valid: false,
		},

		// --- BO3: 3ゲームで決着(2-1 / 1-2) ---
		{
			name: "正常系_BO3_3ゲーム_勝敗勝で勝利",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: true,
				Games: []*dto.GameRequest{game(true), game(false), game(true)},
			},
			valid: true,
		},
		{
			name: "正常系_BO3_3ゲーム_敗勝勝で勝利",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: true,
				Games: []*dto.GameRequest{game(false), game(true), game(true)},
			},
			valid: true,
		},
		{
			name: "正常系_BO3_3ゲーム_勝敗敗で敗北",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: false,
				Games: []*dto.GameRequest{game(true), game(false), game(false)},
			},
			valid: true,
		},
		{
			name: "正常系_BO3_3ゲーム_敗勝敗で敗北",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: false,
				Games: []*dto.GameRequest{game(false), game(true), game(false)},
			},
			valid: true,
		},
		{
			name: "異常系_BO3_3ゲーム_2連勝で決着済みなのに3ゲーム目がある",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: true,
				Games: []*dto.GameRequest{game(true), game(true), game(true)},
			},
			valid: false,
		},
		{
			name: "異常系_BO3_3ゲーム_2連敗で決着済みなのに3ゲーム目がある",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: false,
				Games: []*dto.GameRequest{game(false), game(false), game(false)},
			},
			valid: false,
		},
		{
			name: "異常系_BO3_3ゲーム_3ゲーム目に勝ったのに敗北になっている",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: false,
				Games: []*dto.GameRequest{game(true), game(false), game(true)},
			},
			valid: false,
		},
		{
			name: "異常系_BO3_3ゲーム_3ゲーム目に負けたのに勝利になっている",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: true,
				Games: []*dto.GameRequest{game(true), game(false), game(false)},
			},
			valid: false,
		},

		// --- BO3: ゲーム数が不正 ---
		{
			name: "異常系_BO3_0ゲーム(不戦勝でもないのにゲームが無い)",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: true,
				Games: []*dto.GameRequest{},
			},
			valid: false,
		},
		{
			name: "異常系_BO3_1ゲーム(2本先取に達していない)",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: true,
				Games: []*dto.GameRequest{game(true)},
			},
			valid: false,
		},
		{
			name: "異常系_BO3_4ゲーム(3ゲームを超えている)",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: true,
				Games: []*dto.GameRequest{game(true), game(false), game(true), game(true)},
			},
			valid: false,
		},

		// --- BO3: 不戦勝/不戦敗 ---
		{
			name: "正常系_BO3_不戦勝はゲーム0件でよい",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: true,
				DefaultVictoryFlg: true,
				Games:             []*dto.GameRequest{},
			},
			valid: true,
		},
		{
			name: "正常系_BO3_不戦敗はゲーム0件でよい",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: false,
				DefaultDefeatFlg: true,
				Games:            []*dto.GameRequest{},
			},
			valid: true,
		},
		{
			name: "異常系_BO3_不戦勝なのにゲームが存在する",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: true,
				DefaultVictoryFlg: true,
				Games:             []*dto.GameRequest{game(true), game(true)},
			},
			valid: false,
		},

		// --- BO1 ---
		{
			name: "正常系_BO1_1ゲームで勝利",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: false, VictoryFlg: true,
				Games: []*dto.GameRequest{game(true)},
			},
			valid: true,
		},
		{
			name: "異常系_BO1_0ゲーム(不戦勝でもないのにゲームが無い)",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: false, VictoryFlg: true,
				Games: []*dto.GameRequest{},
			},
			valid: false,
		},
		{
			name: "異常系_BO1_2ゲーム(1本勝負なのに複数ゲームある)",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: false, VictoryFlg: true,
				Games: []*dto.GameRequest{game(true), game(true)},
			},
			valid: false,
		},
		{
			name: "異常系_BO1_ゲームの勝敗と対戦の勝敗が食い違う",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: false, VictoryFlg: true,
				Games: []*dto.GameRequest{game(false)},
			},
			valid: false,
		},

		// --- チーム戦(GroupMatch)との組み合わせ ---
		{
			name: "正常系_チーム戦_BO1ならチームの勝敗を持てる",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: false, VictoryFlg: true,
				GroupMatchFlg: true, GroupMatchVictoryFlg: true,
				Games: []*dto.GameRequest{game(true)},
			},
			valid: true,
		},
		{
			name: "異常系_チーム戦_BO3はチームの勝敗を持てない",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: true, VictoryFlg: true,
				GroupMatchFlg: true, GroupMatchVictoryFlg: true,
				Games: []*dto.GameRequest{game(true), game(true)},
			},
			valid: false,
		},
		{
			name: "異常系_チーム戦でないのにチームの勝敗が立っている",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: false, VictoryFlg: true,
				GroupMatchFlg: false, GroupMatchVictoryFlg: true,
				Games: []*dto.GameRequest{game(true)},
			},
			valid: false,
		},

		// --- その他 ---
		{
			name: "異常系_RecordIdが空",
			req: dto.MatchRequest{
				RecordId: "", BO3Flg: true, VictoryFlg: true,
				Games: []*dto.GameRequest{game(true), game(true)},
			},
			valid: false,
		},
		{
			name: "異常系_不戦勝と不戦敗の両方が立っている",
			req: dto.MatchRequest{
				RecordId: recordId, BO3Flg: false, VictoryFlg: true,
				DefaultVictoryFlg: true, DefaultDefeatFlg: true,
				Games: []*dto.GameRequest{},
			},
			valid: false,
		},
	}

	// 検証内容は作成/更新で同一であるべきなので、両Middlewareに同じケースを流す
	middlewares := map[string]gin.HandlerFunc{
		"Create": MatchCreateMiddleware(),
		"Update": MatchUpdateMiddleware(),
	}

	for name, middleware := range middlewares {
		t.Run(name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					code := runMatchMiddleware(t, middleware, tt.req)

					if tt.valid {
						require.NotEqual(t, http.StatusBadRequest, code, "受理されるべきリクエストが400になっている")
					} else {
						require.Equal(t, http.StatusBadRequest, code, "拒否されるべきリクエストが400になっていない")
					}
				})
			}
		})
	}
}
