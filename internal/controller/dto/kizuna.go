package dto

// KizunaMetricResponse はきずなLv.の内訳1指標ぶん。
// 指標の表示名や説明文は webapp 側が持つ（UIの文言はUIの責務）ため、キーだけを返す。
type KizunaMetricResponse struct {
	Key    string  `json:"key"`
	Weight int     `json:"weight"`
	Value  float64 `json:"value"`
	// Points の合計が Level に一致する
	Points    int `json:"points"`
	MaxPoints int `json:"max_points"`
}

type KizunaDeckResponse struct {
	DeckId  string                  `json:"deck_id"`
	Level   int                     `json:"level"`
	Metrics []*KizunaMetricResponse `json:"metrics"`
}

type KizunaResponse struct {
	UserId string `json:"user_id"`
	// MaxLevel は 255 固定。クライアント側で上限を直書きさせないために返す。
	MaxLevel int                   `json:"max_level"`
	Decks    []*KizunaDeckResponse `json:"decks"`
}
