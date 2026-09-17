package entity

import (
	"regexp"
	"time"
)

// deckCodePattern はデッキコードとして受け付ける文字種。
// 公式サイトのデッキコードは英数字6文字をハイフンで3つ繋いだ形("5dbFbk-uBwjqP-VVk5Vv")
// だが、ここでは文字種だけを見る(桁の内訳まで固定すると、形式が変わったときに
// 正しいコードまで弾いてしまう)。長さの上限は controller 層(MaxDeckCodeLength)が見る。
var deckCodePattern = regexp.MustCompile(`^[0-9A-Za-z-]+$`)

// IsValidDeckCodeFormat はデッキコードが英数字とハイフンだけで構成されているかを返す。
//
// デッキコードは公式サイトのURLのパスと、画像・HTMLを保存するストレージのキーに
// そのまま埋め込まれる。"/" や "?" を通すと、公式サイトの別のページを取得させたり
// 意図しないキーで保存させたりできてしまうため、文字種を限定する。
func IsValidDeckCodeFormat(code string) bool {
	return deckCodePattern.MatchString(code)
}

type DeckCode struct {
	ID             string
	CreatedAt      time.Time
	UserId         string
	DeckId         string
	Code           string
	PrivateCodeFlg bool
	Memo           string
	// Tags は付与されたタグ。読み込み時にインフラ層が詰める。詳細は Deck.Tags と同様。
	Tags []*Tag
}

func NewDeckCode(
	id string,
	createdAt time.Time,
	userId string,
	deckId string,
	code string,
	privateCodeFlg bool,
	memo string,
) *DeckCode {
	return &DeckCode{
		ID:             id,
		CreatedAt:      createdAt,
		UserId:         userId,
		DeckId:         deckId,
		Code:           code,
		PrivateCodeFlg: privateCodeFlg,
		Memo:           memo,
	}
}
