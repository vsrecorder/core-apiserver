package entity

import "time"

// タグのプリセット群(Tag.PresetCategory)。付与先ごとに出し分けるために使う。
// ユーザー個別タグ(PresetFlg=false)は常に空文字。
const (
	// TagPresetCategoryAceSpec は ACE SPECカードのプリセット。
	// デッキ・デッキコード・対戦結果の付与UIで見せる。
	TagPresetCategoryAceSpec = "acespec"
	// TagPresetCategoryPlacement は大会順位(優勝・ベスト4 など)のプリセット。
	// 記録の付与UIで見せる。
	TagPresetCategoryPlacement = "placement"
)

// IsValidTagPresetCategory はプリセット群の絞り込みに使える値かを返す。
// 空文字は「群で絞らない(全プリセット)」を表すため許容する。
func IsValidTagPresetCategory(category string) bool {
	switch category {
	case "", TagPresetCategoryAceSpec, TagPresetCategoryPlacement:
		return true
	default:
		return false
	}
}

// Tag はユーザーが任意に付けられるラベル。ユーザーごとに名前空間を持ち、
// デッキ・デッキコード・記録・対戦結果に付与できる。
// 付与先との関連はエンティティごとの中間テーブル(deck_tags など)で表す。
type Tag struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	UserId    string
	Name      string
	Color     string // '#RRGGBB' 形式。未設定は空文字。
	// PresetFlg=true は運営が用意する全ユーザー共通のプリセットタグ(例: ACE SPECカード)。
	// プリセットは特定ユーザーに属さない(UserId は空文字)。誰でも付与できるが編集・削除は不可。
	PresetFlg bool
	// PresetCategory はプリセットの群(TagPresetCategoryAceSpec など)。
	// 付与先によって見せるプリセットを切り替えるために使う。
	// ユーザー個別タグは空文字。
	PresetCategory string
	// TextColor は Color の上に乗せる文字色('#RRGGBB' 形式)。配色まで指定したい
	// プリセット用で、空なら表示側が Color の明るさから白/黒を選ぶ。
	// APIからは設定できず、投入時(schema.sql / backfill)にだけ決まる。
	TextColor string
}

func NewTag(
	id string,
	createdAt time.Time,
	updatedAt time.Time,
	userId string,
	name string,
	color string,
	presetFlg bool,
	presetCategory string,
	textColor string,
) *Tag {
	return &Tag{
		ID:             id,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		UserId:         userId,
		Name:           name,
		Color:          color,
		PresetFlg:      presetFlg,
		PresetCategory: presetCategory,
		TextColor:      textColor,
	}
}
