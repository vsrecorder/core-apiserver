package entity

// Regulation は対戦記録のレギュレーション(使用可能なカードの範囲)。
// スタンダード / エクストラ / 殿堂 の3種類で、値は regulations テーブル(db/schema.sql)が正。
//
// 「スタンダードで使えるレギュレーションマークの組み合わせと適用期間」を表す
// StandardRegulation とは別物なので混同しないこと。
type Regulation struct {
	ID   uint
	Name string
}

func NewRegulation(
	id uint,
	name string,
) *Regulation {
	return &Regulation{
		ID:   id,
		Name: name,
	}
}

// 既知のレギュレーションID。regulations テーブルの初期データと対応する。
// 記録の検証はマスタを都度引かずこの定数で行うため、テーブルへ行を足すときは
// ここと IsValidRegulationId も併せて更新する。
const (
	// RegulationIdStandard は記録作成時の既定値。
	RegulationIdStandard   uint = 1
	RegulationIdExtra      uint = 2
	RegulationIdHallOfFame uint = 3
	// RegulationIdOther は上のいずれにも当てはまらない対戦(独自ルールの自主大会など)。
	RegulationIdOther uint = 4
)

// IsValidRegulationId は regulations に存在するIDかを判定する。
func IsValidRegulationId(id uint) bool {
	switch id {
	case RegulationIdStandard, RegulationIdExtra, RegulationIdHallOfFame, RegulationIdOther:
		return true
	default:
		return false
	}
}

// NormalizeRegulationId は未指定(0)をスタンダードへ寄せる。
// regulation_id を送らない旧クライアントからのリクエストを、DB側の
// DEFAULT 1 と同じ扱いにしてFK違反を防ぐ。
func NormalizeRegulationId(id uint) uint {
	if id == 0 {
		return RegulationIdStandard
	}

	return id
}
