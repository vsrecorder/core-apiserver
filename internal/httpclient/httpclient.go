// Package httpclient は外部サービスへのリクエストで共用するHTTPクライアントを提供する。
//
// http.Get や http.PostForm が使う http.DefaultClient にはタイムアウトが無く、
// 接続先が応答を返さないまま保持し続けるとgoroutineとコネクションが滞留する。
// 外部サービス(ポケモンカード公式・Tonamel)の遅延がAPIサーバ自体の停止に
// 波及しないよう、必ずタイムアウト付きのクライアントを経由させる。
package httpclient

import (
	"net/http"
	"net/url"
	"time"
)

// Timeout は接続からレスポンスボディの読み切りまでを含めた上限。
const Timeout = 10 * time.Second

var client = &http.Client{
	Timeout: Timeout,
}

// Get はタイムアウト付きで http.Get 相当のリクエストを行う。
func Get(url string) (*http.Response, error) {
	return client.Get(url)
}

// PostForm はタイムアウト付きで http.PostForm 相当のリクエストを行う。
func PostForm(url string, data url.Values) (*http.Response, error) {
	return client.PostForm(url, data)
}
