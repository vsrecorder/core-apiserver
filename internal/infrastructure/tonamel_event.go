package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"gorm.io/gorm"
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
	tonamelEvent := &model.TonamelEvent{}

	url := "https://tonamel.com/competition/" + id
	apiURL := "https://web-toolbox.dev/api/ogtag?url=" + url

	// OGPチェッカーから指定されたIDに紐ずくTonamelのOGP情報を取得
	res, err := http.Get(apiURL)
	if err != nil {
		i.logger.Error(
			"failed to fetch Tonamel OGP via OGP checker",
			slog.String("tonamel_id", id),
			slog.String("request_url", apiURL),
			slog.Int("status_code", res.StatusCode),
			slog.String("error_message", err.Error()),
		)

		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		i.logger.Error(
			"OGP checker returned non-200 status",
			slog.String("tonamel_id", id),
			slog.String("request_url", apiURL),
			slog.Int("status_code", res.StatusCode),
		)

		return nil, fmt.Errorf("ogp checker status: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		i.logger.Error(
			"failed to read OGP checker response body",
			slog.String("tonamel_id", id),
			slog.String("request_url", apiURL),
			slog.Int("status_code", res.StatusCode),
			slog.String("error_message", err.Error()),
		)

		return nil, err
	}

	if err := json.Unmarshal(body, tonamelEvent); err != nil {
		i.logger.Error(
			"failed to unmarshal TonamelEvent model",
			slog.String("tonamel_id", id),
			slog.String("request_url", apiURL),
			slog.Int("status_code", res.StatusCode),
			slog.String("error_message", err.Error()),
		)

		return nil, err
	}

	if !tonamelEvent.Success {
		i.logger.Error(
			"Tonamel OGP not found or OGP checker not available",
			slog.String("tonamel_id", id),
			slog.String("request_url", apiURL),
			slog.Int("status_code", res.StatusCode),
			slog.String("error_message", tonamelEvent.Result.Error),
		)

		return nil, gorm.ErrRecordNotFound
	}

	ret := entity.NewTonamelEvent(
		id,
		tonamelEvent.Result.Metadata.Title,
		tonamelEvent.Result.Metadata.Description,
		tonamelEvent.Result.Metadata.Image,
	)

	return ret, nil
}
