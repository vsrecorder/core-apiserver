package dto

type PokemonSpriteRequest struct {
	ID string `json:"id"`
	// 表示枠の位置(1 or 2)。省略時(0)は後方互換のため配列インデックスから採番する。
	Position uint `json:"position,omitempty"`
}

type PokemonSpriteResponse struct {
	ID string `json:"id"`
	// 表示枠の位置(1 or 2)。position を持たない集計系などでは 0 のため omitempty で省略する。
	Position uint `json:"position,omitempty"`
}
