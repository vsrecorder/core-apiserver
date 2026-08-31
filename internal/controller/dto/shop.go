package dto

type ShopResponse struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	ZipCode        string `json:"zip_code"`
	PrefectureId   uint   `json:"prefecture_id"`
	PrefectureName string `json:"prefecture_name"`
	Address        string `json:"address"`
	Tel            string `json:"tel"`
	BusinessHours  string `json:"business_hours"`
	URL            string `json:"url"`
}

type ShopGetResponse struct {
	Keyword string          `json:"keyword"`
	Count   int             `json:"count"`
	Shops   []*ShopResponse `json:"shops"`
}
