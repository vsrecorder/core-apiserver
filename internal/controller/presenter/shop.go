package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func newShopResponse(shop *entity.Shop) *dto.ShopResponse {
	return &dto.ShopResponse{
		ID:             shop.ID,
		Name:           shop.Name,
		ZipCode:        shop.ZipCode,
		PrefectureId:   shop.PrefectureId,
		PrefectureName: shop.PrefectureName,
		Address:        shop.Address,
		Tel:            shop.Tel,
		BusinessHours:  shop.BusinessHours,
		URL:            shop.URL,
	}
}

func NewShopGetResponse(
	keyword string,
	shops []*entity.Shop,
) *dto.ShopGetResponse {
	ret := []*dto.ShopResponse{}

	for _, shop := range shops {
		ret = append(ret, newShopResponse(shop))
	}

	return &dto.ShopGetResponse{
		Keyword: keyword,
		Count:   len(ret),
		Shops:   ret,
	}
}
