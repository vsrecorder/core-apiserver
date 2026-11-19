package dto

import "time"

type DeckCodeCreateRequest struct {
	DeckId         string   `json:"deck_id"`
	Code           string   `json:"code"`
	PrivateCodeFlg bool     `json:"private_code_flg"`
	Memo           string   `json:"memo"`
	TagIds         []string `json:"tag_ids"`
	// PokemonSprites はこのバージョンに設定するスプライト。省略(null)と空配列で意味が違う。
	//   省略   : 同じデッキの直前のバージョンの設定を引き継ぐ(無ければ未設定)
	//   空配列 : 未設定で作る(表示側が親デッキのスプライトへフォールバックする)
	// この項目を知らない古いクライアントから作ったバージョンだけが未設定に戻らないようにするため。
	PokemonSprites []*PokemonSpriteRequest `json:"pokemon_sprites"`
}

type DeckCodeUpdateRequest struct {
	PrivateCodeFlg bool     `json:"private_code_flg"`
	Memo           string   `json:"memo"`
	TagIds         []string `json:"tag_ids"`
	// PokemonSprites は省略(null)と空配列で意味が違う。
	//   省略   : スプライトに触れない(現在の設定を保つ)
	//   空配列 : 未設定に戻す
	// この項目を知らない古いクライアント(メモ・タグだけを送る)の更新で、設定済みの
	// スプライトが消えないようにするため。
	PokemonSprites []*PokemonSpriteRequest `json:"pokemon_sprites"`
}

type DeckCodeResponse struct {
	ID             string         `json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UserId         string         `json:"user_id"`
	DeckId         string         `json:"deck_id"`
	Code           string         `json:"code"`
	PrivateCodeFlg bool           `json:"private_code_flg"`
	Memo           string         `json:"memo"`
	Tags           []*TagResponse `json:"tags"`
	// PokemonSprites はこのバージョンに設定したスプライト。未設定なら空配列で、
	// 表示側は親デッキの pokemon_sprites へフォールバックする。
	PokemonSprites []*PokemonSpriteResponse `json:"pokemon_sprites"`
}

type DeckCodeGetByIdResponse struct {
	DeckCodeResponse
}

type DeckCodeCreateResponse struct {
	DeckCodeResponse
}

type DeckCodeUpdateResponse struct {
	DeckCodeResponse
}
