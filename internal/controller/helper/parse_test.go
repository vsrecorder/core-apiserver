package helper

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// newTestContext は指定したクエリ文字列を持つGETリクエストのgin.Contextを返す。
func newTestContext(t *testing.T, rawQuery string) *gin.Context {
	t.Helper()

	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	url := "/"
	if rawQuery != "" {
		url = "/?" + rawQuery
	}

	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)

	ctx.Request = req

	return ctx
}

func TestParseQueryLimit(t *testing.T) {
	t.Parallel()

	t.Run("正常系_未指定ならデフォルト値を返す", func(t *testing.T) {
		limit, err := ParseQueryLimit(newTestContext(t, ""))
		require.NoError(t, err)
		require.Equal(t, DefaultLimit, limit)
	})

	t.Run("正常系_正の数値ならその値を返す", func(t *testing.T) {
		limit, err := ParseQueryLimit(newTestContext(t, "limit=25"))
		require.NoError(t, err)
		require.Equal(t, 25, limit)
	})

	t.Run("正常系_0以下ならデフォルト値へ丸める", func(t *testing.T) {
		limit, err := ParseQueryLimit(newTestContext(t, "limit=0"))
		require.NoError(t, err)
		require.Equal(t, DefaultLimit, limit)

		limit, err = ParseQueryLimit(newTestContext(t, "limit=-5"))
		require.NoError(t, err)
		require.Equal(t, DefaultLimit, limit)
	})

	t.Run("異常系_数値でなければエラーを返す", func(t *testing.T) {
		_, err := ParseQueryLimit(newTestContext(t, "limit=abc"))
		require.Error(t, err)
	})
}

func TestParseQueryOffset(t *testing.T) {
	t.Parallel()

	t.Run("正常系_未指定ならデフォルト値を返す", func(t *testing.T) {
		offset, err := ParseQueryOffset(newTestContext(t, ""))
		require.NoError(t, err)
		require.Equal(t, DefaultOffset, offset)
	})

	t.Run("正常系_正の数値ならその値を返す", func(t *testing.T) {
		offset, err := ParseQueryOffset(newTestContext(t, "offset=30"))
		require.NoError(t, err)
		require.Equal(t, 30, offset)
	})

	t.Run("正常系_0以下ならデフォルト値へ丸める", func(t *testing.T) {
		offset, err := ParseQueryOffset(newTestContext(t, "offset=-1"))
		require.NoError(t, err)
		require.Equal(t, DefaultOffset, offset)
	})

	t.Run("異常系_数値でなければエラーを返す", func(t *testing.T) {
		_, err := ParseQueryOffset(newTestContext(t, "offset=abc"))
		require.Error(t, err)
	})
}

func TestParseQuerySingleCursor(t *testing.T) {
	t.Parallel()

	t.Run("正常系_未指定ならゼロ値を返す", func(t *testing.T) {
		cursor, err := ParseQuerySingleCursor(newTestContext(t, ""))
		require.NoError(t, err)
		require.True(t, cursor.IsZero())
	})

	t.Run("正常系_base64エンコードされたRFC3339の時刻を復元する", func(t *testing.T) {
		want := time.Date(2026, 7, 1, 12, 34, 56, 0, time.UTC)
		encoded := base64.StdEncoding.EncodeToString([]byte(want.Format(time.RFC3339)))

		cursor, err := ParseQuerySingleCursor(newTestContext(t, "cursor="+encoded))
		require.NoError(t, err)
		require.True(t, cursor.Equal(want))
	})

	t.Run("異常系_base64として不正ならエラーを返す", func(t *testing.T) {
		_, err := ParseQuerySingleCursor(newTestContext(t, "cursor=%21invalid"))
		require.Error(t, err)
	})

	t.Run("異常系_復号結果がRFC3339でなければエラーを返す", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("not-a-time"))
		_, err := ParseQuerySingleCursor(newTestContext(t, "cursor="+encoded))
		require.Error(t, err)
	})
}

func encodeCompositeCursor(t *testing.T, eventDate string, createdAt string) string {
	t.Helper()

	b, err := json.Marshal(cursorJSON{EventDate: eventDate, CreatedAt: createdAt})
	require.NoError(t, err)

	return base64.StdEncoding.EncodeToString(b)
}

func TestParseQueryCursor(t *testing.T) {
	t.Parallel()

	t.Run("正常系_未指定なら両方ゼロ値を返す", func(t *testing.T) {
		eventDate, createdAt, err := ParseQueryCursor(newTestContext(t, ""))
		require.NoError(t, err)
		require.True(t, eventDate.IsZero())
		require.True(t, createdAt.IsZero())
	})

	t.Run("正常系_event_dateとcreated_atを復元する", func(t *testing.T) {
		wantEventDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		wantCreatedAt := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
		encoded := encodeCompositeCursor(t, wantEventDate.Format(time.RFC3339), wantCreatedAt.Format(time.RFC3339))

		eventDate, createdAt, err := ParseQueryCursor(newTestContext(t, "cursor="+encoded))
		require.NoError(t, err)
		require.True(t, eventDate.Equal(wantEventDate))
		require.True(t, createdAt.Equal(wantCreatedAt))
	})

	t.Run("異常系_base64として不正ならエラーを返す", func(t *testing.T) {
		_, _, err := ParseQueryCursor(newTestContext(t, "cursor=%21invalid"))
		require.Error(t, err)
	})

	t.Run("異常系_JSONとして不正ならエラーを返す", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("not-json"))
		_, _, err := ParseQueryCursor(newTestContext(t, "cursor="+encoded))
		require.Error(t, err)
	})

	t.Run("異常系_event_dateが欠けていればエラーを返す", func(t *testing.T) {
		encoded := encodeCompositeCursor(t, "", time.Now().Format(time.RFC3339))
		_, _, err := ParseQueryCursor(newTestContext(t, "cursor="+encoded))
		require.Error(t, err)
	})

	t.Run("異常系_created_atが欠けていればエラーを返す", func(t *testing.T) {
		encoded := encodeCompositeCursor(t, time.Now().Format(time.RFC3339), "")
		_, _, err := ParseQueryCursor(newTestContext(t, "cursor="+encoded))
		require.Error(t, err)
	})

	t.Run("異常系_時刻の形式が不正ならエラーを返す", func(t *testing.T) {
		encoded := encodeCompositeCursor(t, "2026/07/01", time.Now().Format(time.RFC3339))
		_, _, err := ParseQueryCursor(newTestContext(t, "cursor="+encoded))
		require.Error(t, err)

		encoded = encodeCompositeCursor(t, time.Now().Format(time.RFC3339), "2026/07/02")
		_, _, err = ParseQueryCursor(newTestContext(t, "cursor="+encoded))
		require.Error(t, err)
	})
}

func TestParseQueryDate(t *testing.T) {
	t.Parallel()

	t.Run("正常系_未指定ならゼロ値を返す", func(t *testing.T) {
		date, err := ParseQueryDate(newTestContext(t, ""))
		require.NoError(t, err)
		require.True(t, date.IsZero())
	})

	t.Run("正常系_指定日をローカル時刻の0時として返す", func(t *testing.T) {
		date, err := ParseQueryDate(newTestContext(t, "date=2026-07-18"))
		require.NoError(t, err)
		require.Equal(t, time.Date(2026, 7, 18, 0, 0, 0, 0, time.Local), date)
	})

	t.Run("異常系_形式が不正ならエラーを返す", func(t *testing.T) {
		_, err := ParseQueryDate(newTestContext(t, "date=20260718"))
		require.Error(t, err)
	})
}

func TestParseQueryFromDateAndToDate(t *testing.T) {
	t.Parallel()

	t.Run("正常系_未指定ならゼロ値を返す", func(t *testing.T) {
		fromDate, err := ParseQueryFromDate(newTestContext(t, ""))
		require.NoError(t, err)
		require.True(t, fromDate.IsZero())

		toDate, err := ParseQueryToDate(newTestContext(t, ""))
		require.NoError(t, err)
		require.True(t, toDate.IsZero())
	})

	t.Run("正常系_指定日をローカル時刻の0時として返す", func(t *testing.T) {
		fromDate, err := ParseQueryFromDate(newTestContext(t, "from_date=2026-07-01"))
		require.NoError(t, err)
		require.Equal(t, time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local), fromDate)

		toDate, err := ParseQueryToDate(newTestContext(t, "to_date=2026-07-31"))
		require.NoError(t, err)
		require.Equal(t, time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local), toDate)
	})

	t.Run("異常系_形式が不正ならエラーを返す", func(t *testing.T) {
		_, err := ParseQueryFromDate(newTestContext(t, "from_date=2026/07/01"))
		require.Error(t, err)

		_, err = ParseQueryToDate(newTestContext(t, "to_date=2026/07/31"))
		require.Error(t, err)
	})
}

func TestParseQueryStartDateAndEndDate(t *testing.T) {
	t.Parallel()

	t.Run("正常系_未指定なら当日の0時を返す", func(t *testing.T) {
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

		startDate, err := ParseQueryStartDate(newTestContext(t, ""))
		require.NoError(t, err)
		require.Equal(t, today, startDate)

		endDate, err := ParseQueryEndDate(newTestContext(t, ""))
		require.NoError(t, err)
		require.Equal(t, today, endDate)
	})

	t.Run("正常系_指定日をローカル時刻の0時として返す", func(t *testing.T) {
		startDate, err := ParseQueryStartDate(newTestContext(t, "start_date=2026-07-01"))
		require.NoError(t, err)
		require.Equal(t, time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local), startDate)

		endDate, err := ParseQueryEndDate(newTestContext(t, "end_date=2026-07-31"))
		require.NoError(t, err)
		require.Equal(t, time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local), endDate)
	})

	t.Run("異常系_形式が不正ならエラーを返す", func(t *testing.T) {
		_, err := ParseQueryStartDate(newTestContext(t, "start_date=bad"))
		require.Error(t, err)

		_, err = ParseQueryEndDate(newTestContext(t, "end_date=bad"))
		require.Error(t, err)
	})
}

func TestParseQueryOfficialEventId(t *testing.T) {
	t.Parallel()

	t.Run("正常系_未指定ならデフォルト値を返す", func(t *testing.T) {
		id, err := ParseQueryOfficialEventId(newTestContext(t, ""))
		require.NoError(t, err)
		require.Equal(t, uint(DefaultOfficialEventId), id)
	})

	t.Run("正常系_正の数値ならその値を返す", func(t *testing.T) {
		id, err := ParseQueryOfficialEventId(newTestContext(t, "official_event_id=10000"))
		require.NoError(t, err)
		require.Equal(t, uint(10000), id)
	})

	t.Run("異常系_数値でなければエラーを返す", func(t *testing.T) {
		_, err := ParseQueryOfficialEventId(newTestContext(t, "official_event_id=abc"))
		require.Error(t, err)
	})

	t.Run("異常系_負数ならエラーを返す", func(t *testing.T) {
		_, err := ParseQueryOfficialEventId(newTestContext(t, "official_event_id=-1"))
		require.Error(t, err)
	})
}

func TestParseQueryTypeId(t *testing.T) {
	t.Parallel()

	t.Run("正常系_未指定ならデフォルト値を返す", func(t *testing.T) {
		typeId, err := ParseQueryTypeId(newTestContext(t, ""))
		require.NoError(t, err)
		require.Equal(t, uint(DefaultTypeId), typeId)
	})

	// 大型大会(1)/シティ(2)/トレリ(3)/ジムイベント(4)/オーガナイザー(6)/その他(7)が有効
	t.Run("正常系_有効な種類IDはそのまま返す", func(t *testing.T) {
		for _, valid := range []int{1, 2, 3, 4, 6, 7} {
			typeId, err := ParseQueryTypeId(newTestContext(t, fmt.Sprintf("type_id=%d", valid)))
			require.NoError(t, err)
			require.Equal(t, uint(valid), typeId)
		}
	})

	t.Run("正常系_0以下はデフォルト値へ丸める", func(t *testing.T) {
		typeId, err := ParseQueryTypeId(newTestContext(t, "type_id=0"))
		require.NoError(t, err)
		require.Equal(t, uint(DefaultTypeId), typeId)
	})

	t.Run("異常系_数値でなければエラーを返す", func(t *testing.T) {
		_, err := ParseQueryTypeId(newTestContext(t, "type_id=abc"))
		require.Error(t, err)
	})

	t.Run("異常系_未定義の種類IDはエラーを返す", func(t *testing.T) {
		for _, invalid := range []int{5, 8, 100} {
			_, err := ParseQueryTypeId(newTestContext(t, fmt.Sprintf("type_id=%d", invalid)))
			require.Error(t, err)
		}
	})
}

func TestParseQueryLeagueType(t *testing.T) {
	t.Parallel()

	t.Run("正常系_未指定ならデフォルト値を返す", func(t *testing.T) {
		leagueType, err := ParseQueryLeagueType(newTestContext(t, ""))
		require.NoError(t, err)
		require.Equal(t, uint(DefaultLeagueType), leagueType)
	})

	// オープン(1)/ジュニア(2)/シニア(3)/マスター(4)が有効
	t.Run("正常系_有効なリーグ種別はそのまま返す", func(t *testing.T) {
		for _, valid := range []int{1, 2, 3, 4} {
			leagueType, err := ParseQueryLeagueType(newTestContext(t, fmt.Sprintf("league_type=%d", valid)))
			require.NoError(t, err)
			require.Equal(t, uint(valid), leagueType)
		}
	})

	t.Run("正常系_0以下はデフォルト値へ丸める", func(t *testing.T) {
		leagueType, err := ParseQueryLeagueType(newTestContext(t, "league_type=-1"))
		require.NoError(t, err)
		require.Equal(t, uint(DefaultLeagueType), leagueType)
	})

	t.Run("異常系_数値でなければエラーを返す", func(t *testing.T) {
		_, err := ParseQueryLeagueType(newTestContext(t, "league_type=abc"))
		require.Error(t, err)
	})

	t.Run("異常系_5以上はエラーを返す", func(t *testing.T) {
		_, err := ParseQueryLeagueType(newTestContext(t, "league_type=5"))
		require.Error(t, err)
	})
}

func TestParseQueryEventType(t *testing.T) {
	t.Parallel()

	t.Run("正常系_未指定ならデフォルト値を返す", func(t *testing.T) {
		eventType, err := ParseQueryEventType(newTestContext(t, ""))
		require.NoError(t, err)
		require.Equal(t, DefaultEventType, eventType)
	})

	t.Run("正常系_定義済みのイベント種別はそのまま返す", func(t *testing.T) {
		for _, valid := range []string{"official", "tonamel", "unofficial"} {
			eventType, err := ParseQueryEventType(newTestContext(t, "event_type="+valid))
			require.NoError(t, err)
			require.Equal(t, valid, eventType)
		}
	})

	t.Run("正常系_未定義の値はデフォルト値へ丸める", func(t *testing.T) {
		eventType, err := ParseQueryEventType(newTestContext(t, "event_type=unknown"))
		require.NoError(t, err)
		require.Equal(t, DefaultEventType, eventType)
	})
}

func TestParseQueryArchive(t *testing.T) {
	t.Parallel()

	t.Run("正常系_未指定ならデフォルト値を返す", func(t *testing.T) {
		archived, err := ParseQueryArchive(newTestContext(t, ""))
		require.NoError(t, err)
		require.Equal(t, DefaultArchived, archived)
	})

	t.Run("正常系_真偽値を解釈して返す", func(t *testing.T) {
		archived, err := ParseQueryArchive(newTestContext(t, "archived=true"))
		require.NoError(t, err)
		require.True(t, archived)

		archived, err = ParseQueryArchive(newTestContext(t, "archived=false"))
		require.NoError(t, err)
		require.False(t, archived)
	})

	t.Run("異常系_真偽値でなければエラーを返す", func(t *testing.T) {
		_, err := ParseQueryArchive(newTestContext(t, "archived=abc"))
		require.Error(t, err)
	})
}

func TestParseQueryAllTime(t *testing.T) {
	t.Parallel()

	t.Run("正常系_未指定ならデフォルト値を返す", func(t *testing.T) {
		allTime, err := ParseQueryAllTime(newTestContext(t, ""))
		require.NoError(t, err)
		require.Equal(t, DefaultAllTime, allTime)
	})

	t.Run("正常系_真偽値を解釈して返す", func(t *testing.T) {
		allTime, err := ParseQueryAllTime(newTestContext(t, "all_time=true"))
		require.NoError(t, err)
		require.True(t, allTime)
	})

	t.Run("異常系_真偽値でなければエラーを返す", func(t *testing.T) {
		_, err := ParseQueryAllTime(newTestContext(t, "all_time=abc"))
		require.Error(t, err)
	})
}

func TestParseQueryYearMonth(t *testing.T) {
	t.Parallel()

	t.Run("正常系_未指定なら空文字を返す", func(t *testing.T) {
		yearMonth, err := ParseQueryYearMonth(newTestContext(t, ""))
		require.NoError(t, err)
		require.Equal(t, "", yearMonth)
	})

	t.Run("正常系_YYYY-MM形式の値はそのまま返す", func(t *testing.T) {
		yearMonth, err := ParseQueryYearMonth(newTestContext(t, "year_month=2026-07"))
		require.NoError(t, err)
		require.Equal(t, "2026-07", yearMonth)
	})

	t.Run("異常系_形式が不正ならエラーを返す", func(t *testing.T) {
		_, err := ParseQueryYearMonth(newTestContext(t, "year_month=202607"))
		require.Error(t, err)

		_, err = ParseQueryYearMonth(newTestContext(t, "year_month=2026-13"))
		require.Error(t, err)
	})
}

func TestParseQuerySeason(t *testing.T) {
	t.Parallel()

	t.Run("正常系_未指定なら空文字を返す", func(t *testing.T) {
		season, err := ParseQuerySeason(newTestContext(t, ""))
		require.NoError(t, err)
		require.Equal(t, "", season)
	})

	t.Run("正常系_YYYY形式の値はそのまま返す", func(t *testing.T) {
		season, err := ParseQuerySeason(newTestContext(t, "season=2026"))
		require.NoError(t, err)
		require.Equal(t, "2026", season)
	})

	t.Run("異常系_形式が不正ならエラーを返す", func(t *testing.T) {
		_, err := ParseQuerySeason(newTestContext(t, "season=abc"))
		require.Error(t, err)
	})
}

func TestParseQueryWeek(t *testing.T) {
	t.Parallel()

	t.Run("正常系_未指定なら空文字を返す", func(t *testing.T) {
		week, err := ParseQueryWeek(newTestContext(t, ""))
		require.NoError(t, err)
		require.Equal(t, "", week)
	})

	t.Run("正常系_YYYY-MM-DD形式の値はそのまま返す", func(t *testing.T) {
		week, err := ParseQueryWeek(newTestContext(t, "week=2026-07-13"))
		require.NoError(t, err)
		require.Equal(t, "2026-07-13", week)
	})

	t.Run("異常系_形式が不正ならエラーを返す", func(t *testing.T) {
		_, err := ParseQueryWeek(newTestContext(t, "week=2026/07/13"))
		require.Error(t, err)
	})
}
