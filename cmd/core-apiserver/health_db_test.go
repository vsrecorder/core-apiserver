package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// fakePinger は ping の結果を差し替えるためのもの
type fakePinger struct {
	err error
	// block が true なら、ctx が打ち切られるまで返さない(DB が応答しない状態)
	block bool
}

func (p fakePinger) PingContext(ctx context.Context) error {
	if p.block {
		<-ctx.Done()
		return ctx.Err()
	}
	return p.err
}

func serveDBHealth(t *testing.T, p pinger) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET(dbHealthPath, dbHealthHandler(p))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, dbHealthPath, nil))
	return w
}

/*
 * 外形監視(webapp の /health)が DB の状態を状態コードで受け取れることを確かめる。
 * webapp は 200 を「使える」、503 を「DB に届かない」として読むので、この 2 つを取り違えると
 * 障害を見落とすか、正常なのに監視が鳴り続ける。
 */
func TestDBHealthHandler(t *testing.T) {
	t.Run("正常系_pingが通れば200とokを返す", func(t *testing.T) {
		w := serveDBHealth(t, fakePinger{})

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "ok", w.Body.String())
		require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	})

	// エラーの文言(接続先などを含みうる)は外へ出さない
	t.Run("異常系_pingが失敗すれば503を返しエラーの中身は出さない", func(t *testing.T) {
		w := serveDBHealth(t, fakePinger{err: errors.New("dial tcp 10.0.0.1:5432: connection refused")})

		require.Equal(t, http.StatusServiceUnavailable, w.Code)
		require.Equal(t, "db unreachable", w.Body.String())
		require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	})

	// DB が応答しないときに待ち続けず、webapp の待ち時間(3 秒)より前に 503 を返す
	t.Run("異常系_pingが返らなければ時間切れで503を返す", func(t *testing.T) {
		started := time.Now()
		w := serveDBHealth(t, fakePinger{block: true})

		require.Equal(t, http.StatusServiceUnavailable, w.Code)
		require.Equal(t, "db unreachable", w.Body.String())
		require.Less(t, time.Since(started), 3*time.Second)
	})
}
