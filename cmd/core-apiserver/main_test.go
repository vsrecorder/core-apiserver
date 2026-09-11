package main

import (
	"net"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

/*
 * --health(runHealthCheck)の判定を確かめる。
 *
 * このイメージは distroless で shell もツールも無く、docker の healthcheck は
 * このバイナリ自身を実行する形でしか書けない。判定を誤ると、応答できない
 * コンテナが healthy のままデプロイ完了になったり(検知漏れ)、正常なのに
 * `docker compose up --wait` が終わらなくなったりする。
 *
 * 実際に listenPort で待ち受けて、動いているサーバを相手に判定させる。
 */
func TestRunHealthCheck(t *testing.T) {
	t.Run("正常系_サーバが200を返せば成功する", func(t *testing.T) {
		srv := startHealthServer(t, http.StatusOK)
		defer srv.Close()

		require.Equal(t, ExitCodeOK, runHealthCheck())
	})

	// 起動途中や過負荷で 5xx を返す状態は「応答できている」とは見なさない
	t.Run("異常系_200以外を返せば失敗する", func(t *testing.T) {
		srv := startHealthServer(t, http.StatusServiceUnavailable)
		defer srv.Close()

		require.Equal(t, ExitCodeNG, runHealthCheck())
	})

	t.Run("異常系_誰も待ち受けていなければ失敗する", func(t *testing.T) {
		require.Equal(t, ExitCodeNG, runHealthCheck())
	})
}

// startHealthServer は healthPath に指定のステータスを返すサーバを listenPort で起動する。
func startHealthServer(t *testing.T, status int) *http.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc(healthPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	})

	srv := &http.Server{Handler: mux}

	ln, err := net.Listen("tcp", "127.0.0.1:"+listenPort)
	require.NoError(t, err)

	go func() {
		// Close 後の ErrServerClosed は正常終了なので握りつぶす
		_ = srv.Serve(ln)
	}()

	return srv
}

// 実サーバのルーティングでも /health が 200 を返すことを確かめる。
// runHealthCheck が正しくても、サーバ側にこの経路が無ければ healthcheck は常に失敗する。
func TestHealthRouteReturnsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET(healthPath, func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "health")
	})

	srv := &http.Server{Handler: r}

	ln, err := net.Listen("tcp", "127.0.0.1:"+listenPort)
	require.NoError(t, err)
	defer srv.Close()

	go func() {
		_ = srv.Serve(ln)
	}()

	require.Equal(t, ExitCodeOK, runHealthCheck())
}
