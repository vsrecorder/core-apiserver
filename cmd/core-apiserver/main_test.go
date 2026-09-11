package main

import (
	"net"
	"net/http"
	"strconv"
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
 * 待ち受けは空きポートで行う。listenPort に固定すると、同じ機械で本体が
 * 動いているだけでテストが落ちる。
 */
func TestRunHealthCheckAt(t *testing.T) {
	t.Run("正常系_サーバが200を返せば成功する", func(t *testing.T) {
		port, srv := startHealthServer(t, http.StatusOK)
		defer srv.Close()

		require.Equal(t, ExitCodeOK, runHealthCheckAt(port))
	})

	// 起動途中や過負荷で 5xx を返す状態は「応答できている」とは見なさない
	t.Run("異常系_200以外を返せば失敗する", func(t *testing.T) {
		port, srv := startHealthServer(t, http.StatusServiceUnavailable)
		defer srv.Close()

		require.Equal(t, ExitCodeNG, runHealthCheckAt(port))
	})

	t.Run("異常系_誰も待ち受けていなければ失敗する", func(t *testing.T) {
		// 一度確保してすぐ閉じたポートなら、誰も待ち受けていない
		port, srv := startHealthServer(t, http.StatusOK)
		srv.Close()

		require.Equal(t, ExitCodeNG, runHealthCheckAt(port))
	})
}

// 実サーバのルーティングでも /health が 200 を返すことを確かめる。
// runHealthCheckAt が正しくても、サーバ側にこの経路が無ければ healthcheck は常に失敗する。
func TestHealthRouteReturnsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET(healthPath, func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "health")
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := &http.Server{Handler: r}
	defer srv.Close()

	go func() {
		// Close 後の ErrServerClosed は正常終了なので握りつぶす
		_ = srv.Serve(ln)
	}()

	require.Equal(t, ExitCodeOK, runHealthCheckAt(portOf(t, ln)))
}

// startHealthServer は healthPath に指定のステータスを返すサーバを空きポートで起動し、
// そのポート番号を返す。
func startHealthServer(t *testing.T, status int) (string, *http.Server) {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc(healthPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := &http.Server{Handler: mux}

	go func() {
		_ = srv.Serve(ln)
	}()

	return portOf(t, ln), srv
}

func portOf(t *testing.T, ln net.Listener) string {
	t.Helper()

	addr, ok := ln.Addr().(*net.TCPAddr)
	require.True(t, ok, "unexpected listener address: %v", ln.Addr())

	return strconv.Itoa(addr.Port)
}
