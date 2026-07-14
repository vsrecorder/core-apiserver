package infrastructure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	deckAssetBaseEndpoint = "https://s3.isk01.sakurastorage.jp"
	deckAssetBucket       = "vsrecorder"
)

type DeckAsset struct {
	logger *slog.Logger
}

func NewDeckAsset(logger *slog.Logger) repository.DeckAssetInterface {
	return &DeckAsset{logger}
}

func (i *DeckAsset) s3Client(ctx context.Context) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	baseEndpoint := deckAssetBaseEndpoint

	return s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = &baseEndpoint
	}), nil
}

// isNotFound は指定したキーのオブジェクトが存在するかどうかを返す。
// オブジェクトが存在する場合(=アップロード済み)はfalseを返す。
func isNotFound(ctx context.Context, s3client *s3.Client, key string) (bool, error) {
	if _, err := s3client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(deckAssetBucket),
		Key:    aws.String(key),
	}); err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return true, nil
		}

		return false, err
	}

	return false, nil
}

func putObject(ctx context.Context, s3client *s3.Client, key string, body []byte) error {
	_, err := s3client.PutObject(ctx, &s3.PutObjectInput{
		ACL:    "public-read",
		Bucket: aws.String(deckAssetBucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})

	return err
}

func (i *DeckAsset) UploadDeckResultHTML(
	ctx context.Context,
	deckCode string,
) error {
	s3client, err := i.s3Client(ctx)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("deck-result_html/%s", deckCode)

	// すでにアップロードされている場合はスキップする
	notFound, err := isNotFound(ctx, s3client, key)
	if err != nil {
		return err
	}
	if !notFound {
		return nil
	}

	url := fmt.Sprintf("https://www.pokemon-card.com/deck/result.html/deckID/%s", deckCode)

	resp, err := http.Get(url)
	if err != nil {
		i.logger.Error(
			"failed to fetch deck result HTML page",
			slog.String("deck_code", deckCode),
			slog.String("request_url", url),
			slog.String("error_message", err.Error()),
		)

		return err
	}
	defer resp.Body.Close()

	// 異常なレスポンスのHTMLをS3にアップロードしてしまうと、
	// HeadObjectで存在確認をしているため以降ずっと壊れたページを配信し続けることになる。
	// そのためステータスが200以外のときはアップロードせずにエラーを返す。
	if resp.StatusCode != http.StatusOK {
		i.logger.Error(
			"deck result HTML page returned non-200 status",
			slog.String("deck_code", deckCode),
			slog.String("request_url", url),
			slog.Int("status_code", resp.StatusCode),
		)

		return fmt.Errorf("deck result html page status: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		i.logger.Error(
			"failed to read deck result HTML page body",
			slog.String("deck_code", deckCode),
			slog.String("request_url", url),
			slog.String("error_message", err.Error()),
		)

		return err
	}

	// メンテナンス中のときはアップロードしない
	if bytes.Contains(bodyBytes, []byte("現在メンテナンスをしております")) {
		return apperror.ErrUnderMaintenance
	}

	// デッキコードエラーのときはアップロードしない
	if bytes.Contains(bodyBytes, []byte("デッキコードが正しくありません")) {
		return apperror.ErrDeckCodeInvalid
	}

	return putObject(ctx, s3client, key, bodyBytes)
}

func (i *DeckAsset) UploadDeckImage(
	ctx context.Context,
	deckCode string,
) error {
	s3client, err := i.s3Client(ctx)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("images/decks/%s.jpg", deckCode)

	// すでにアップロードされている場合はスキップする
	notFound, err := isNotFound(ctx, s3client, key)
	if err != nil {
		return err
	}
	if !notFound {
		return nil
	}

	url := fmt.Sprintf("https://www.pokemon-card.com/deck/deckView.php/deckID/%s.png", deckCode)

	resp, err := http.Get(url)
	if err != nil {
		i.logger.Error(
			"failed to fetch deck image",
			slog.String("deck_code", deckCode),
			slog.String("request_url", url),
			slog.String("error_message", err.Error()),
		)

		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		i.logger.Error(
			"deck image returned non-200 status",
			slog.String("deck_code", deckCode),
			slog.String("request_url", url),
			slog.Int("status_code", resp.StatusCode),
		)

		return fmt.Errorf("deck image status: %d", resp.StatusCode)
	}

	srcImg, _, err := image.Decode(resp.Body)
	if err != nil {
		i.logger.Error(
			"failed to decode deck image",
			slog.String("deck_code", deckCode),
			slog.String("request_url", url),
			slog.String("error_message", err.Error()),
		)

		return err
	}

	var w bytes.Buffer
	if err := png.Encode(&w, srcImg); err != nil {
		return err
	}

	imageBytes, err := convertPNG2JPG(w.Bytes())
	if err != nil {
		return err
	}

	return putObject(ctx, s3client, key, imageBytes)
}

func convertPNG2JPG(imageBytes []byte) ([]byte, error) {
	contentType := http.DetectContentType(imageBytes)

	switch contentType {
	case "image/png":
		img, err := png.Decode(bytes.NewReader(imageBytes))
		if err != nil {
			return nil, err
		}

		buf := new(bytes.Buffer)
		if err := jpeg.Encode(buf, img, nil); err != nil {
			return nil, err
		}

		return buf.Bytes(), nil
	}

	return nil, fmt.Errorf("unable to convert %#v to jpeg", contentType)
}
