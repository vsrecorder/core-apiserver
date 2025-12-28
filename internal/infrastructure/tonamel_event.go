package infrastructure

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"gorm.io/gorm"
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
	tonamelEvent := &model.TonamelEvent{}

	// OGPチェッカーから指定されたIDに紐ずくTonamelのOGP情報を取得
	res, err := http.Get("https://web-toolbox.dev/api/ogtag?url=https://tonamel.com/competition/" + id)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	if err := json.Unmarshal(body, tonamelEvent); err != nil {
		return nil, err
	}

	if !tonamelEvent.Success {
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
