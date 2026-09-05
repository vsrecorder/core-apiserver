package dto

import "time"

type DeckCodePostCreateRequest struct {
	DeckCodeId string `json:"deck_code_id"`
}

// DeckCodePostUserResponse は投稿者・いいねした人の公開情報。
// DesignationTier は現在の称号ティア(0=称号なし)。ランクは webapp がティアから導出する。
type DeckCodePostUserResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ImageURL        string `json:"image_url"`
	DesignationTier int    `json:"designation_tier"`
}

type DeckCodePostResponse struct {
	ID          string    `json:"id"`
	PublishedAt time.Time `json:"published_at"`
	// UnpublishedAt は取り下げ日時。公開中はゼロ値(年が1)。
	UnpublishedAt time.Time `json:"unpublished_at"`
	// Hidden は運営が非表示にしているか。投稿者本人向けの応答(デッキの投稿一覧・個別ページ)でだけ
	// true になり得る(閲覧者向けの一覧には非表示の投稿は出ない)。
	Hidden          bool                       `json:"hidden"`
	User            DeckCodePostUserResponse   `json:"user"`
	DeckId          string                     `json:"deck_id"`
	DeckName        string                     `json:"deck_name"`
	PokemonSprites  []*PokemonSpriteResponse   `json:"pokemon_sprites"`
	DeckCodeId      string                     `json:"deck_code_id"`
	Code            string                     `json:"code"`
	AceSpecCardId   string                     `json:"ace_spec_card_id"`
	AceSpecCardName string                     `json:"ace_spec_card_name"`
	AceSpecImageURL string                     `json:"ace_spec_image_url"`
	LikeCount       int                        `json:"like_count"`
	ImportCount     int                        `json:"import_count"`
	LikedByMe       bool                       `json:"liked_by_me"`
	RecentLikers    []DeckCodePostUserResponse `json:"recent_likers"`
}

// DeckCodePostAceSpecCountResponse は ACE SPEC での絞り込み候補1件(公開中の投稿で使われている ACE SPEC と投稿数)。
type DeckCodePostAceSpecCountResponse struct {
	CardName string `json:"card_name"`
	ImageURL string `json:"image_url"`
	Count    int    `json:"count"`
}

type DeckCodePostGetAceSpecsResponse struct {
	Environment *DeckCodePostEnvironmentResponse   `json:"environment"`
	AceSpecs    []DeckCodePostAceSpecCountResponse `json:"acespecs"`
}

// DeckCodePostEnvironmentResponse は一覧の絞り込みに使った環境。
type DeckCodePostEnvironmentResponse struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	FromDate time.Time `json:"from_date"`
	ToDate   time.Time `json:"to_date"`
}

type DeckCodePostGetResponse struct {
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Sort   string `json:"sort"`
	// Environment は絞り込みに使った環境。今日に対応する環境が無い場合は null。
	Environment *DeckCodePostEnvironmentResponse `json:"environment"`
	Posts       []DeckCodePostResponse           `json:"posts"`
}

type DeckCodePostGetByIdResponse struct {
	DeckCodePostResponse
}

type DeckCodePostCreateResponse struct {
	DeckCodePostResponse
}

type DeckCodePostLikeResponse struct {
	DeckCodePostResponse
}

type DeckCodePostLikerResponse struct {
	User      DeckCodePostUserResponse `json:"user"`
	CreatedAt time.Time                `json:"created_at"`
}

type DeckCodePostGetLikersResponse struct {
	Limit  int                         `json:"limit"`
	Offset int                         `json:"offset"`
	Likers []DeckCodePostLikerResponse `json:"likers"`
}

type DeckCodePostGetByUserIdResponse struct {
	User           DeckCodePostUserResponse `json:"user"`
	PostCount      int                      `json:"post_count"`
	LikeCountTotal int                      `json:"like_count_total"`
	Limit          int                      `json:"limit"`
	Offset         int                      `json:"offset"`
	Posts          []DeckCodePostResponse   `json:"posts"`
}

type DeckCodePostGetByDeckIdResponse []DeckCodePostResponse
