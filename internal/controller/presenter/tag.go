package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// newTagResponses は付与タグを TagResponse 配列に変換する。
// デッキ/デッキコードのレスポンスに埋め込むタグ表現もこれを使う。
func newTagResponses(tags []*entity.Tag) []*dto.TagResponse {
	ret := []*dto.TagResponse{}
	for _, tag := range tags {
		ret = append(ret, &dto.TagResponse{
			ID:        tag.ID,
			CreatedAt: tag.CreatedAt,
			Name:      tag.Name,
			Color:     tag.Color,
			PresetFlg: tag.PresetFlg,
			TextColor: tag.TextColor,
		})
	}

	return ret
}

func NewTagGetResponse(
	tags []*entity.Tag,
) *dto.TagGetResponse {
	ret := dto.TagGetResponse{}
	for _, tag := range tags {
		ret = append(ret, dto.TagResponse{
			ID:        tag.ID,
			CreatedAt: tag.CreatedAt,
			Name:      tag.Name,
			Color:     tag.Color,
			PresetFlg: tag.PresetFlg,
			TextColor: tag.TextColor,
		})
	}

	return &ret
}

func NewTagCreateResponse(
	tag *entity.Tag,
) *dto.TagCreateResponse {
	return &dto.TagCreateResponse{
		TagResponse: dto.TagResponse{
			ID:        tag.ID,
			CreatedAt: tag.CreatedAt,
			Name:      tag.Name,
			Color:     tag.Color,
			PresetFlg: tag.PresetFlg,
			TextColor: tag.TextColor,
		},
	}
}

func NewTagUpdateResponse(
	tag *entity.Tag,
) *dto.TagUpdateResponse {
	return &dto.TagUpdateResponse{
		TagResponse: dto.TagResponse{
			ID:        tag.ID,
			CreatedAt: tag.CreatedAt,
			Name:      tag.Name,
			Color:     tag.Color,
			PresetFlg: tag.PresetFlg,
			TextColor: tag.TextColor,
		},
	}
}
