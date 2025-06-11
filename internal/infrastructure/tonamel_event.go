package infrastructure

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type TonamelEvent struct {
}

func NewTonamelEvent() repository.TonamelEventInterface {
	return &TonamelEvent{}
}

func (i *TonamelEvent) FindById(
	ctx context.Context,
	id string,
) (*entity.TonamelEvent, error) {
	m := &model.TonamelEvent{}

	// OGPチェッカーから指定されたIDに紐ずくTonamelのOGP情報を取得
	res, err := http.Get("https://web-toolbox.dev/api/ogtag?url=https://tonamel.com/competition/" + id)

	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, err
	}

	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	if err := json.Unmarshal(body, m); err != nil {
		return nil, err
	}

	e := entity.NewTonamelEvent(
		id,
		m.Result.Metadata.Title,
		m.Result.Metadata.Description,
		m.Result.Metadata.Image,
	)

	return e, nil
}
