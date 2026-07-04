package infrastructure

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"golang.org/x/net/html"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

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
	url := "https://tonamel.com/competition/" + id

	res, err := http.Get(url)
	if err != nil {
		i.logger.Error(
			"failed to fetch Tonamel event page",
			slog.String("tonamel_id", id),
			slog.String("request_url", url),
			slog.String("error_message", err.Error()),
		)

		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		i.logger.Error(
			"Tonamel event not found",
			slog.String("tonamel_id", id),
			slog.String("request_url", url),
			slog.Int("status_code", res.StatusCode),
		)

		return nil, apperror.ErrRecordNotFound
	}

	if res.StatusCode != http.StatusOK {
		i.logger.Error(
			"Tonamel event page returned non-200 status",
			slog.String("tonamel_id", id),
			slog.String("request_url", url),
			slog.Int("status_code", res.StatusCode),
		)

		return nil, fmt.Errorf("tonamel event page status: %d", res.StatusCode)
	}

	ogpTitle, ogpDescription, ogpImage, err := extractOGP(res.Body)
	if err != nil {
		i.logger.Error(
			"failed to parse Tonamel event page HTML",
			slog.String("tonamel_id", id),
			slog.String("request_url", url),
			slog.String("error_message", err.Error()),
		)

		return nil, err
	}

	if ogpTitle == "" {
		i.logger.Error(
			"Tonamel OGP title not found",
			slog.String("tonamel_id", id),
			slog.String("request_url", url),
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
