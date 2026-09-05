package entity

import "time"

// DeckCodePost は「みんなの公開デッキ」へ公開したデッキコードの投稿。
// 1つのデッキコード(バージョン)につき公開中の投稿は最大1件で、取り下げると
// UnpublishedAt が入る。取り下げ後に公開し直すと別の投稿(別ID)として作り直す
// (いいねは引き継がない仕様のため)。
//
// User / DeckName / PokemonSprites / Code / RecentLikers / LikedByMe は
// 一覧表示のためにインフラ層が読み込み時に詰める。DesignationTier は
// 投稿者の現在の称号ティア(0=称号なし)で、usecase 層が詰める。
type DeckCodePost struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	UserId    string
	DeckId    string
	// DeckCodeId は公開したデッキコード(バージョン)。
	DeckCodeId string
	// PublishedAt は公開した日時。タイムライン上の投稿時刻であり、環境の判定にも使う。
	PublishedAt time.Time
	// UnpublishedAt は取り下げた日時。公開中はゼロ値。
	UnpublishedAt time.Time
	// HiddenAt は運営が非表示にした日時。通常はゼロ値。APIからは書き込まない。
	HiddenAt time.Time
	// AceSpecCardId / AceSpecCardName / AceSpecImageURL は公開時にデッキコードから判定した ACE SPEC。
	// 入っていないデッキでは空文字。表示にも使う(webapp が一覧でカードごとに acespec API を引かない)。
	// AceSpecImageURL は他の付随情報と同じく、コンストラクタの後で詰める。
	AceSpecCardId   string
	AceSpecCardName string
	AceSpecImageURL string
	// LikeCount はいいね数。インフラ層が deck_code_post_likes を数えて詰める(列としては持たない)。
	LikeCount int
	// ImportCount は「デッキ登録」で自分のデッキとして取り込んだ人数。インフラ層が
	// deck_code_post_imports を数えて詰める(列としては持たない)。同じ人が何度取り込んでも1と数える。
	// AceSpecImageURL と同じく、コンストラクタの後で詰める。
	ImportCount int

	User            *User
	DeckName        string
	PokemonSprites  []*PokemonSprite
	Code            string
	RecentLikers    []*User
	LikedByMe       bool
	DesignationTier int
}

func NewDeckCodePost(
	id string,
	createdAt time.Time,
	updatedAt time.Time,
	userId string,
	deckId string,
	deckCodeId string,
	publishedAt time.Time,
	unpublishedAt time.Time,
	hiddenAt time.Time,
	aceSpecCardId string,
	aceSpecCardName string,
	likeCount int,
) *DeckCodePost {
	return &DeckCodePost{
		ID:              id,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		UserId:          userId,
		DeckId:          deckId,
		DeckCodeId:      deckCodeId,
		PublishedAt:     publishedAt,
		UnpublishedAt:   unpublishedAt,
		HiddenAt:        hiddenAt,
		AceSpecCardId:   aceSpecCardId,
		AceSpecCardName: aceSpecCardName,
		LikeCount:       likeCount,
	}
}

// IsActive は公開中(取り下げも運営の非表示もされていない)かを返す。
func (p *DeckCodePost) IsActive() bool {
	return p.UnpublishedAt.IsZero() && p.HiddenAt.IsZero()
}

// DeckCodePostLiker はある投稿にいいねしたユーザ1人分(一覧シート用)。
type DeckCodePostLiker struct {
	User      *User
	CreatedAt time.Time
	// DesignationTier は usecase 層が詰める(0=称号なし)。
	DesignationTier int
}

// DeckCodePostUserSummary は投稿者ページの見出しに出す集計値。
type DeckCodePostUserSummary struct {
	// PostCount は公開中の投稿数。
	PostCount int
	// LikeCountTotal は公開中の投稿がもらったいいねの合計。
	LikeCountTotal int
}

// DeckCodePostLikeDigest は、ある期間に投稿へ付いたいいねのまとめ(通知用)。
// 投稿1件につき1行で、期間内のいいね数と最後に押した人の名前を持つ。
type DeckCodePostLikeDigest struct {
	PostId string
	// OwnerUserId は投稿者(通知の宛先)。
	OwnerUserId string
	DeckName    string
	// LikeCount は期間内に付いたいいねの数(投稿者自身のいいねは含めない)。
	LikeCount int
	// LatestLikerName は期間内で最後にいいねした人の名前(「◯◯さんほかN人」の◯◯)。
	LatestLikerName string
}

// DeckCodePostAceSpecCount は、公開中の投稿で使われている ACE SPEC 1種と、その投稿数(絞り込みの候補)。
//
// 同じカードでも収録セットごとに card_id が違う(「アンフェアスタンプ」は 47870 / 49349 / 45640 …)。
// 利用者にとっては1枚のカードなので、カード名で束ねて数える。
type DeckCodePostAceSpecCount struct {
	CardName string
	ImageURL string
	Count    int
}

// AceSpecCard はデッキコードから判定した ACE SPEC カード(deckcard-api の応答)。
type AceSpecCard struct {
	CardId   string
	CardName string
	ImageURL string
}
