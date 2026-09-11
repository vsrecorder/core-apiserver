/*
 * Slack の incoming webhook へ通知する。
 *
 * 監視系のバッチが「壊れていることに気付ける」ための共通の出口。
 * cmd 配下の各バッチが同じ送信処理を持つと、タイムアウトの有無や失敗時の扱いが
 * ばらけるため、ここに集約する。
 */
package slack

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/vsrecorder/core-apiserver/internal/httpclient"
)

// ErrWebhookURLNotSet は webhook URL が空のまま通知しようとしたことを表す。
var ErrWebhookURLNotSet = errors.New("slack webhook url is not set")

// responseBodyLimit は失敗理由として読み取るレスポンス本文の上限。
const responseBodyLimit = 1024

// Notify は incoming webhook へメッセージを投稿する。
//
// タイムアウトの無い http.DefaultClient を使うと、Slack が応答しないときにバッチが
// 終わらず次回起動と重なるため、必ず internal/httpclient を経由する。
func Notify(webhookURL string, message string) error {
	if webhookURL == "" {
		return ErrWebhookURLNotSet
	}

	payload, err := json.Marshal(struct {
		Text string `json:"text"`
	}{Text: message})
	if err != nil {
		return err
	}

	resp, err := httpclient.PostJSON(webhookURL, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Slack は失敗理由を "invalid_payload" のようにボディへ入れて返すため、原因調査用に読み取る
		body, _ := io.ReadAll(io.LimitReader(resp.Body, responseBodyLimit))
		return fmt.Errorf("slack webhook returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}
