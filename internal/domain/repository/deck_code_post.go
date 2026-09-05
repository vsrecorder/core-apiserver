package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// DeckCodePostSort は一覧の並び順。
const (
	// DeckCodePostSortNew は公開日時の新しい順。
	DeckCodePostSortNew = "new"
	// DeckCodePostSortPopular は直近7日間のいいね数の多い順(同数は新しい順)。
	DeckCodePostSortPopular = "popular"
)

// DeckCodePostFilter は公開中の投稿一覧の絞り込み条件。
type DeckCodePostFilter struct {
	// Sort は DeckCodePostSortNew / DeckCodePostSortPopular。空文字は新しい順。
	Sort string
	// From / To は公開日時の範囲(From <= published_at < To)。ゼロ値は制限なし。
	// 環境の期間で絞るために使う。
	From time.Time
	To   time.Time
	// PopularSince は人気順で数えるいいねの開始日時(直近7日間など)。
	PopularSince time.Time
	// AceSpecCardName を指定するとその ACE SPEC を採用した投稿だけに絞る。
	// 収録セット違い(card_id 違い)を1枚として扱うため、IDではなく名前で絞る。
	AceSpecCardName string
	// PokemonSpriteIds を指定すると、デッキのスプライト(1体目・2体目)にそれらをすべて含む投稿だけに絞る
	// (位置は問わない。2体指定なら両方を持つデッキ)。webapp のスプライト選択と同じく最大2件。
	PokemonSpriteIds []string
	// ViewerUserId は閲覧者。空でなければ LikedByMe を判定する。
	ViewerUserId string
}

// 投稿の「状態」は2つの列で表す。
//   - unpublished_at: 投稿者が取り下げた(または連動して取り下がった)。以後その行は使わない。
//   - hidden_at: 運営が非表示にした。タイムライン・個別ページには出さないが、投稿者から見れば
//     「公開中」のままで、取り下げることはできる。公開中の枠(1コードにつき1件)も占有し続ける。
//
// そのため「閲覧者向けの公開中」は両方が NULL、「投稿者向けの公開中(枠を占有している)」は
// unpublished_at が NULL、と使い分ける。部分一意索引は後者に合わせてある。
type DeckCodePostInterface interface {
	// Find は閲覧者向けに公開中(取り下げ・非表示でない)の投稿を filter に従って返す。
	// 投稿者・デッキ名・スプライト・コード・直近のいいねした人も詰めて返す。
	Find(
		ctx context.Context,
		filter *DeckCodePostFilter,
		limit int,
		offset int,
	) ([]*entity.DeckCodePost, error)

	// FindAceSpecCounts は filter(期間)に合う閲覧者向けに公開中の投稿で使われている ACE SPEC を、
	// 投稿数の多い順に返す(ACE SPEC での絞り込みの候補)。filter の Sort / AceSpecCardName /
	// PokemonSpriteIds は使わない。判定できていない(空の)投稿は数えない。
	FindAceSpecCounts(
		ctx context.Context,
		filter *DeckCodePostFilter,
	) ([]*entity.DeckCodePostAceSpecCount, error)

	// FindById は取り下げ済み・非表示も含めて1件返す(個別ページで 410 を返す判定に使う)。
	// viewerUserId が空でなければ LikedByMe を判定する。
	// 存在しなければ apperror.ErrRecordNotFound。
	FindById(
		ctx context.Context,
		id string,
		viewerUserId string,
	) (*entity.DeckCodePost, error)

	// FindLiteById は投稿の本体だけ(投稿者・デッキ・いいねした人を結合せず)を1件返す。
	// 所有者や状態の確認だけが要る場面(認可・取り下げ・いいね前の確認)で使う。
	// 存在しなければ apperror.ErrRecordNotFound。
	FindLiteById(
		ctx context.Context,
		id string,
	) (*entity.DeckCodePost, error)

	// FindActiveByDeckCodeId はデッキコードの「枠を占有している」投稿(取り下げていないもの。
	// 運営の非表示は含む)を返す。無ければ apperror.ErrRecordNotFound。
	FindActiveByDeckCodeId(
		ctx context.Context,
		deckCodeId string,
	) (*entity.DeckCodePost, error)

	// FindLatestByDeckCodeId はデッキコードの最新の投稿(取り下げ済みを含む)を返す。
	// 同じコードの公開し直しの間隔を制限するために使う。無ければ apperror.ErrRecordNotFound。
	FindLatestByDeckCodeId(
		ctx context.Context,
		deckCodeId string,
	) (*entity.DeckCodePost, error)

	// FindActiveByDeckId はデッキの取り下げていない投稿(全バージョン分。運営の非表示は含む)を返す。
	// デッキ詳細モーダル・バージョン履歴で公開スイッチの状態を出すために使う。
	FindActiveByDeckId(
		ctx context.Context,
		deckId string,
	) ([]*entity.DeckCodePost, error)

	// FindByUserId は uid の閲覧者向けに公開中の投稿を新しい順で返す(投稿者ページ)。
	FindByUserId(
		ctx context.Context,
		uid string,
		viewerUserId string,
		limit int,
		offset int,
	) ([]*entity.DeckCodePost, error)

	// SummarizeByUserId は uid の閲覧者向けに公開中の投稿数といいね合計を返す(投稿者ページの見出し)。
	SummarizeByUserId(
		ctx context.Context,
		uid string,
	) (*entity.DeckCodePostUserSummary, error)

	// Save は投稿を保存する。同じデッキコードの投稿が枠を占有していれば apperror.ErrAlreadyExists。
	Save(
		ctx context.Context,
		entity *entity.DeckCodePost,
	) error

	// Unpublish は投稿を取り下げ、そのいいねを削除する(公開し直しても戻らない仕様)。
	Unpublish(
		ctx context.Context,
		id string,
		unpublishedAt time.Time,
	) error

	// UnpublishByDeckId はデッキの取り下げていない投稿をすべて取り下げる(アーカイブ・削除の連動)。
	// 取り下げた投稿のいいねは削除する。
	UnpublishByDeckId(
		ctx context.Context,
		deckId string,
		unpublishedAt time.Time,
	) error

	// UnpublishByDeckCodeId はデッキコードの取り下げていない投稿を取り下げる(バージョン削除の連動)。
	UnpublishByDeckCodeId(
		ctx context.Context,
		deckCodeId string,
		unpublishedAt time.Time,
	) error

	// Like は uid のいいねを追加する。既にいいね済みなら何もしない。
	Like(
		ctx context.Context,
		postId string,
		uid string,
		createdAt time.Time,
	) error

	// Unlike は uid のいいねを取り消す。いいねしていなければ何もしない。
	Unlike(
		ctx context.Context,
		postId string,
		uid string,
	) error

	// RecordImport は uid が投稿を「取り込む」で使ったことを記録する(運営の指標)。
	// 同じ人が何度使っても1回として数える(既に記録があれば何もしない)。
	RecordImport(
		ctx context.Context,
		postId string,
		uid string,
		createdAt time.Time,
	) error

	// FindLikers は投稿にいいねしたユーザを新しい順で返す。
	FindLikers(
		ctx context.Context,
		postId string,
		limit int,
		offset int,
	) ([]*entity.DeckCodePostLiker, error)

	// FindLikeDigests は from <= created_at < to に付いたいいねを、閲覧者向けに公開中の投稿ごとに
	// まとめて返す(日次のまとめ通知用)。投稿者自身が押したいいねは数えない。
	FindLikeDigests(
		ctx context.Context,
		from time.Time,
		to time.Time,
	) ([]*entity.DeckCodePostLikeDigest, error)

	// DeleteByUserId は退会時に、そのユーザの投稿と、そのユーザのデッキに紐づく投稿(他人が
	// 作ったコードで公開したものを含む)と、それらへのいいね・取り込み記録、そのユーザが押した
	// いいね・取り込み記録をまとめて物理削除する。
	DeleteByUserId(
		ctx context.Context,
		uid string,
	) error
}

// DeckCardInterface はデッキコードからカード情報を引く外部サービス(deckcard-api)への問い合わせ。
type DeckCardInterface interface {
	// FindAceSpec はデッキコードに入っている ACE SPEC カードを返す。
	// 入っていない(deckcard-api が 204 を返す)場合は nil, nil。
	FindAceSpec(
		ctx context.Context,
		deckCode string,
	) (*entity.AceSpecCard, error)
}
