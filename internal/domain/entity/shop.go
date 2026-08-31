package entity

// Shop は公式イベントの開催店舗。公式サイト由来のマスタで、
// import-officialevent-bat がイベントと一緒に取り込む(サービス側では作らない)。
//
// PrefectureName は shops 自身の列ではなく prefectures を結合して埋める。
// 店舗の一覧・検索結果は「どこの店か」が分からないと選べないため、
// 都道府県名まで揃った状態で扱う。
type Shop struct {
	ID             uint
	Name           string
	ZipCode        string
	PrefectureId   uint
	PrefectureName string
	Address        string
	Tel            string
	BusinessHours  string
	URL            string
}

func NewShop(
	id uint,
	name string,
	zipCode string,
	prefectureId uint,
	prefectureName string,
	address string,
	tel string,
	businessHours string,
	url string,
) *Shop {
	return &Shop{
		ID:             id,
		Name:           name,
		ZipCode:        zipCode,
		PrefectureId:   prefectureId,
		PrefectureName: prefectureName,
		Address:        address,
		Tel:            tel,
		BusinessHours:  businessHours,
		URL:            url,
	}
}
