package infrastructure

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/httpclient"
)

// tonamelEventBaseURL は取得先のベースURL。外部サイトへ実通信せずにテストできるよう、
// httptestサーバへ差し替え可能な変数にしている。
var tonamelEventBaseURL = "https://tonamel.com/competition/"

const (
	// tonamelEventMaxBodyBytes は大会ページの読み込み上限。必要な OGP の meta は <head> に
	// あるため先頭だけで足り、上限を置かないと外部サイトの応答の大きさがそのままメモリになる。
	tonamelEventMaxBodyBytes = 2 << 20

	// tonamelEventMaxConcurrentFetches は tonamel.com への同時取得数の上限(プロセス全体)。
	// 取得は未認証の GET /tonamel_events/:id からも走るため、上限が無いと大量に叩かれるだけで
	// 外向きの接続と goroutine が最大でタイムアウト(10秒)ずつ滞留する。超えたぶんは待たせずに
	// 断る(記録作成側は大会情報の取得失敗を許容しており、後から埋め直せる)。
	tonamelEventMaxConcurrentFetches = 4
)

// tonamelEventFetchSlots は同時取得数を数えるセマフォ。
var tonamelEventFetchSlots = make(chan struct{}, tonamelEventMaxConcurrentFetches)

type TonamelEvent struct {
	logger *slog.Logger
}

func NewTonamelEvent(logger *slog.Logger) repository.TonamelEventInterface {
	return &TonamelEvent{logger}
}

func (i *TonamelEvent) FindById(
	ctx context.Context,
	id string,
) (*entity.TonamelEvent, error) {
	// IDは形式を検証済みだが、URLのパスに埋め込む値は必ずエスケープして
	// 万一の混入でもパスが変わらないようにする(検証済みの値では変化しない)。
	requestURL := tonamelEventBaseURL + url.PathEscape(id)

	select {
	case tonamelEventFetchSlots <- struct{}{}:
		defer func() { <-tonamelEventFetchSlots }()
	default:
		i.logger.WarnContext(
			ctx,
			"Tonamel fetch rejected: too many concurrent fetches",
			slog.String("tonamel_id", id),
			slog.Int("max_concurrent_fetches", tonamelEventMaxConcurrentFetches),
		)

		return nil, apperror.ErrExternalFetchBusy
	}

	res, err := httpclient.Get(requestURL)
	if err != nil {
		i.logger.ErrorContext(
			ctx,
			"failed to fetch Tonamel event page",
			slog.String("tonamel_id", id),
			slog.String("request_url", requestURL),
			slog.String("error_message", err.Error()),
		)

		return nil, err
	}
	defer res.Body.Close()

	// 404 は「そのIDのイベントが無い」だけで、障害ではない。
	//
	// 記録作成の入力欄は打っている途中の文字列でもここを引くため、打ち間違いや入力途中で
	// 普通に起きる(2026-09-15 の本番ログでは、1件のIDを1文字ずつ消していった20回ぶんが
	// すべてここを通っていた)。ERROR で残すと、本当に直すべき失敗がこれに埋もれる
	// (実際、アクセスログ調査でエラーを拾ったとき、この行が大半を占めていた)。
	//
	// どのIDが引かれたかは調査に使うので記録自体は残す。応答としての404は
	// controller が WARN で記録している。
	if res.StatusCode == http.StatusNotFound {
		i.logger.InfoContext(
			ctx,
			"Tonamel event not found",
			slog.String("tonamel_id", id),
			slog.String("request_url", requestURL),
			slog.Int("status_code", res.StatusCode),
		)

		return nil, apperror.ErrRecordNotFound
	}

	if res.StatusCode != http.StatusOK {
		i.logger.ErrorContext(
			ctx,
			"Tonamel event page returned non-200 status",
			slog.String("tonamel_id", id),
			slog.String("request_url", requestURL),
			slog.Int("status_code", res.StatusCode),
		)

		return nil, fmt.Errorf("tonamel event page status: %d", res.StatusCode)
	}

	ogpTitle, ogpDescription, ogpImage, err := extractOGP(io.LimitReader(res.Body, tonamelEventMaxBodyBytes))
	if err != nil {
		i.logger.ErrorContext(
			ctx,
			"failed to parse Tonamel event page HTML",
			slog.String("tonamel_id", id),
			slog.String("request_url", requestURL),
			slog.String("error_message", err.Error()),
		)

		return nil, err
	}

	if ogpTitle == "" {
		i.logger.ErrorContext(
			ctx,
			"Tonamel OGP title not found",
			slog.String("tonamel_id", id),
			slog.String("request_url", requestURL),
		)

		return nil, apperror.ErrRecordNotFound
	}

	ret := entity.NewTonamelEvent(id, ogpTitle, ogpDescription, ogpImage)

	return ret, nil
}

// extractOGP はHTMLからog:title、og:description、og:imageを抽出する。
// og:titleが存在しない場合はtwitter:titleまたは<title>タグへフォールバックする。
// og:descriptionが存在しない場合はtwitter:descriptionまたはname=descriptionへフォールバックする。
func extractOGP(r io.Reader) (title, description, image string, err error) {
	doc, err := html.Parse(r)
	if err != nil {
		return "", "", "", err
	}

	// フォールバック用の値
	var twitterTitle, twitterDescription, metaDescription, titleTagText string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var property, name, content string
			for _, a := range n.Attr {
				switch strings.ToLower(a.Key) {
				case "property":
					property = a.Val
				case "name":
					name = a.Val
				case "content":
					content = a.Val
				}
			}
			switch property {
			case "og:title":
				title = content
			case "og:description":
				description = content
			case "og:image":
				image = content
			}
			switch name {
			case "twitter:title":
				twitterTitle = content
			case "twitter:description":
				twitterDescription = content
			case "description":
				metaDescription = content
			}
		}
		// <title>タグのテキストを取得
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			titleTagText = strings.TrimSpace(n.FirstChild.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	// フォールバック: og:title → twitter:title → <title>タグ
	if title == "" {
		if twitterTitle != "" {
			title = twitterTitle
		} else {
			title = titleTagText
		}
	}

	// タイトル末尾の " - Tonamel" を除去する
	title = strings.TrimSuffix(title, " - Tonamel")

	// フォールバック: og:description → twitter:description → name=description
	if description == "" {
		if twitterDescription != "" {
			description = twitterDescription
		} else {
			description = metaDescription
		}
	}

	return title, description, image, nil
}
