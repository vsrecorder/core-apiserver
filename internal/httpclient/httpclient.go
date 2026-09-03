// Package httpclient は外部サービスへのリクエストで共用するHTTPクライアントを提供する。
//
// http.Get や http.PostForm が使う http.DefaultClient にはタイムアウトが無く、
// 接続先が応答を返さないまま保持し続けるとgoroutineとコネクションが滞留する。
// 外部サービス(ポケモンカード公式・Tonamel)の遅延がAPIサーバ自体の停止に
// 波及しないよう、必ずタイムアウト付きのクライアントを経由させる。
package httpclient

import (
	"bytes"
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

// PostJSON はタイムアウト付きで application/json のPOSTリクエストを行う。
// Slackのincoming webhookのように、JSONボディを送る通知先で使う。
func PostJSON(url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return client.Do(req)
}
