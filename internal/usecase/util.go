package usecase

import (
	"time"

	ulid "github.com/oklog/ulid/v2"
)

// entropy はULID生成用の乱数源。DefaultEntropyはプロセス全体で単調増加する
// (=生成順に文字列としても昇順になる)スレッドセーフな実装のため、同一ミリ秒内で
// generateId()が連続で呼ばれても(例: 称号獲得→ランクアップ通知を同時刻で連続作成する
// notifyRankUp等)IDの前後関係が生成順と一致し、created_at が同値の通知が並ぶ際の
// ソートの安定した第2キーとして使える(notification.goのOrder("created_at DESC, id DESC")
// 参照。idも新しい順にすることで、後発のランクアップ通知が称号獲得通知より上に表示される)。
var entropy = ulid.DefaultEntropy()

// timeNow は現在時刻の取得関数。「現在時刻によって結果が変わる」ロジック
// (当月・当シーズン・当週の判定やロック期限の比較)はこれを経由することで、
// テストから固定時刻に差し替えて月末・シーズン境界でも決定的に検証できる。
var timeNow = time.Now

type PokemonSpriteParam struct {
	ID string
	// 表示枠の位置(1 or 2)。省略時(0)は保存側で配列インデックスから採番する。
	Position uint
}

func NewPokemonSpriteParam(
	id string,
) *PokemonSpriteParam {
	return &PokemonSpriteParam{
		ID: id,
	}
}

// NewPokemonSpriteParamWithPosition は表示枠の位置(position)付きで生成する。
func NewPokemonSpriteParamWithPosition(
	id string,
	position uint,
) *PokemonSpriteParam {
	return &PokemonSpriteParam{
		ID:       id,
		Position: position,
	}
}

func generateId() (string, error) {
	ms := ulid.Timestamp(time.Now())
	id, err := ulid.New(ms, entropy)

	return id.String(), err
}
