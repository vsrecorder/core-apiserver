package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

// deckCodePostRecentLikerLimit は一覧の各投稿に埋め込む「いいねした人」のアイコン数。
// webapp が重ねて表示する上限(5人)と揃える。
const deckCodePostRecentLikerLimit = 5

// 投稿の状態の条件(repository.DeckCodePostInterface のコメント参照)。
const (
	// deckCodePostVisibleCondition は閲覧者向けの「公開中」(取り下げ済み・運営の非表示を除く)。
	deckCodePostVisibleCondition = "deck_code_posts.unpublished_at IS NULL AND deck_code_posts.hidden_at IS NULL"
	// deckCodePostOccupyingCondition は投稿者向けの「公開中」(取り下げていない。運営の非表示は含む)。
	// 部分一意索引 idx_deck_code_posts_active_deck_code_id と同じ条件。
	deckCodePostOccupyingCondition = "deck_code_posts.unpublished_at IS NULL"
)

type DeckCodePost struct {
	db *gorm.DB
}

func NewDeckCodePost(
	db *gorm.DB,
) repository.DeckCodePostInterface {
	return &DeckCodePost{db}
}

// deckCodePostRow は投稿に、投稿者(users)・デッキ名(decks)・コード(deck_codes)・いいね数を
// JOIN した1行。一覧で N+1 を避けるため1クエリで引く。
type deckCodePostRow struct {
	ID              string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	UserId          string
	DeckId          string
	DeckCodeId      string
	PublishedAt     time.Time
	UnpublishedAt   *time.Time
	HiddenAt        *time.Time
	AceSpecCardId   string
	AceSpecCardName string
	AceSpecImageURL string
	LikeCount       int
	UserName        string
	UserImageURL    string
	UserCreatedAt   time.Time
	DeckName        string
	Code            string
	LikedByMe       bool
}

// deckCodePostLikerRow はいいねした人の1行(users を JOIN 済み)。
type deckCodePostLikerRow struct {
	PostId        string
	CreatedAt     time.Time
	UserId        string
	UserName      string
	UserImageURL  string
	UserCreatedAt time.Time
}

func timeOrZero(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}

	return *t
}

func timeOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}

	return &t
}

// toDeckCodePostEntity は投稿本体(model)をエンティティにする。結合情報(いいね数を含む)は持たない。
func toDeckCodePostEntity(m *model.DeckCodePost) *entity.DeckCodePost {
	post := entity.NewDeckCodePost(
		m.ID, m.CreatedAt, m.UpdatedAt, m.UserId, m.DeckId, m.DeckCodeId, m.PublishedAt,
		timeOrZero(m.UnpublishedAt), timeOrZero(m.HiddenAt),
		m.AceSpecCardId, m.AceSpecCardName, 0,
	)
	post.AceSpecImageURL = m.AceSpecImageURL

	return post
}

// baseQuery は投稿と投稿者・デッキ・コード・いいね数を JOIN した読み取りの土台。
// viewerUserId が空でなければ、閲覧者がいいね済みかを liked_by_me として同時に引く。
// 退会したユーザ・削除したデッキ・削除したコードに紐づく投稿は結合で落ちる
// (退会・削除時に取り下げているので通常は残らないが、二重の防御)。
func (i *DeckCodePost) baseQuery(ctx context.Context, viewerUserId string) *gorm.DB {
	db := dbFromContext(ctx, i.db).WithContext(ctx)

	likedByMe := "FALSE AS liked_by_me"
	args := []any{}
	if viewerUserId != "" {
		likedByMe = "EXISTS (SELECT 1 FROM deck_code_post_likes l WHERE l.post_id = deck_code_posts.id AND l.user_id = ?) AS liked_by_me"
		args = append(args, viewerUserId)
	}

	return db.Table("deck_code_posts").
		Select(`
			deck_code_posts.id AS id,
			deck_code_posts.created_at AS created_at,
			deck_code_posts.updated_at AS updated_at,
			deck_code_posts.user_id AS user_id,
			deck_code_posts.deck_id AS deck_id,
			deck_code_posts.deck_code_id AS deck_code_id,
			deck_code_posts.published_at AS published_at,
			deck_code_posts.unpublished_at AS unpublished_at,
			deck_code_posts.hidden_at AS hidden_at,
			deck_code_posts.ace_spec_card_id AS ace_spec_card_id,
			deck_code_posts.ace_spec_card_name AS ace_spec_card_name,
			deck_code_posts.ace_spec_image_url AS ace_spec_image_url,
			(SELECT COUNT(*) FROM deck_code_post_likes lc WHERE lc.post_id = deck_code_posts.id) AS like_count,
			users.name AS user_name,
			users.image_url AS user_image_url,
			users.created_at AS user_created_at,
			decks.name AS deck_name,
			deck_codes.code AS code,
			`+likedByMe, args...).
		Joins("JOIN users ON users.id = deck_code_posts.user_id AND users.deleted_at IS NULL").
		Joins("JOIN decks ON decks.id = deck_code_posts.deck_id AND decks.deleted_at IS NULL").
		Joins("JOIN deck_codes ON deck_codes.id = deck_code_posts.deck_code_id AND deck_codes.deleted_at IS NULL")
}

// findMany は組み立て済みのクエリを実行し、スプライトと直近のいいねした人を付けて返す。
func (i *DeckCodePost) findMany(ctx context.Context, q *gorm.DB) ([]*entity.DeckCodePost, error) {
	var rows []*deckCodePostRow
	if tx := q.Scan(&rows); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	return i.assemble(ctx, rows)
}

// findOne は findMany の1件版。無ければ apperror.ErrRecordNotFound。
func (i *DeckCodePost) findOne(ctx context.Context, q *gorm.DB) (*entity.DeckCodePost, error) {
	posts, err := i.findMany(ctx, q.Limit(1))
	if err != nil {
		return nil, err
	}
	if len(posts) == 0 {
		return nil, apperror.ErrRecordNotFound
	}

	return posts[0], nil
}

// assemble は JOIN 済みの行にスプライトと直近のいいねした人を付けてエンティティにする。
func (i *DeckCodePost) assemble(ctx context.Context, rows []*deckCodePostRow) ([]*entity.DeckCodePost, error) {
	if len(rows) == 0 {
		return []*entity.DeckCodePost{}, nil
	}

	db := dbFromContext(ctx, i.db)

	deckIds := make([]string, 0, len(rows))
	postIds := make([]string, 0, len(rows))
	for _, r := range rows {
		deckIds = append(deckIds, r.DeckId)
		postIds = append(postIds, r.ID)
	}

	spritesByDeckId, err := findDeckPokemonSpritesByDeckIds(ctx, db, deckIds)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	likersByPostId, err := i.findRecentLikersByPostIds(ctx, postIds)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	ret := make([]*entity.DeckCodePost, 0, len(rows))
	for _, r := range rows {
		post := entity.NewDeckCodePost(
			r.ID, r.CreatedAt, r.UpdatedAt, r.UserId, r.DeckId, r.DeckCodeId, r.PublishedAt,
			timeOrZero(r.UnpublishedAt), timeOrZero(r.HiddenAt),
			r.AceSpecCardId, r.AceSpecCardName, r.LikeCount,
		)
		post.AceSpecImageURL = r.AceSpecImageURL
		post.User = entity.NewUser(r.UserId, r.UserCreatedAt, r.UserName, normalizeUserImageURL(r.UserImageURL))
		post.DeckName = r.DeckName
		post.Code = r.Code
		post.LikedByMe = r.LikedByMe

		sprites := spritesByDeckId[r.DeckId]
		if sprites == nil {
			sprites = []*entity.PokemonSprite{}
		}
		post.PokemonSprites = sprites

		likers := likersByPostId[r.ID]
		if likers == nil {
			likers = []*entity.User{}
		}
		post.RecentLikers = likers

		ret = append(ret, post)
	}

	return ret, nil
}

// findRecentLikersByPostIds は各投稿の直近のいいねした人(最大 deckCodePostRecentLikerLimit 人)を
// 1クエリで引く。投稿ごとの上限は窓関数で切る。
func (i *DeckCodePost) findRecentLikersByPostIds(ctx context.Context, postIds []string) (map[string][]*entity.User, error) {
	if len(postIds) == 0 {
		return map[string][]*entity.User{}, nil
	}

	var rows []*deckCodePostLikerRow
	if tx := dbFromContext(ctx, i.db).WithContext(ctx).Raw(`
		SELECT
			l.post_id AS post_id,
			l.created_at AS created_at,
			users.id AS user_id,
			users.name AS user_name,
			users.image_url AS user_image_url,
			users.created_at AS user_created_at
		FROM (
			SELECT post_id, user_id, created_at,
				ROW_NUMBER() OVER (PARTITION BY post_id ORDER BY created_at DESC, user_id) AS rn
			FROM deck_code_post_likes
			WHERE post_id IN ?
		) l
		JOIN users ON users.id = l.user_id AND users.deleted_at IS NULL
		WHERE l.rn <= ?
		ORDER BY l.post_id, l.created_at DESC
	`, postIds, deckCodePostRecentLikerLimit).Scan(&rows); tx.Error != nil {
		return nil, tx.Error
	}

	ret := make(map[string][]*entity.User, len(postIds))
	for _, r := range rows {
		ret[r.PostId] = append(ret[r.PostId], entity.NewUser(r.UserId, r.UserCreatedAt, r.UserName, normalizeUserImageURL(r.UserImageURL)))
	}

	return ret, nil
}

// applyPeriod は公開日時の範囲(環境の期間)を条件に足す。
func applyPeriod(q *gorm.DB, filter *repository.DeckCodePostFilter) *gorm.DB {
	if !filter.From.IsZero() {
		q = q.Where("deck_code_posts.published_at >= ?", filter.From)
	}
	if !filter.To.IsZero() {
		q = q.Where("deck_code_posts.published_at < ?", filter.To)
	}

	return q
}

func (i *DeckCodePost) Find(
	ctx context.Context,
	filter *repository.DeckCodePostFilter,
	limit int,
	offset int,
) ([]*entity.DeckCodePost, error) {
	q := applyPeriod(i.baseQuery(ctx, filter.ViewerUserId).Where(deckCodePostVisibleCondition), filter)

	if filter.AceSpecCardName != "" {
		// 収録セット違いをまとめて拾うため、カードIDではなく名前で絞る
		q = q.Where("deck_code_posts.ace_spec_card_name = ?", filter.AceSpecCardName)
	}
	// 指定されたスプライトをすべて持つデッキに絞る(位置は問わない)。2体指定なら両方を持つデッキ
	for _, spriteId := range filter.PokemonSpriteIds {
		q = q.Where(
			"EXISTS (SELECT 1 FROM deck_pokemon_sprites dps WHERE dps.deck_id = deck_code_posts.deck_id AND dps.pokemon_sprite_id = ?)",
			spriteId,
		)
	}

	if filter.Sort == repository.DeckCodePostSortPopular {
		// 人気順は「直近のいいね数」で並べる。候補の投稿だけを相関副問い合わせで数え、
		// 無い投稿は 0 として扱う。同数は新しい順にする。
		q = q.Order(gorm.Expr(
			"(SELECT COUNT(*) FROM deck_code_post_likes rl WHERE rl.post_id = deck_code_posts.id AND rl.created_at >= ?) DESC",
			filter.PopularSince,
		))
	}

	return i.findMany(ctx, q.Order("deck_code_posts.published_at DESC").Limit(limit).Offset(offset))
}

// FindAceSpecCounts は期間内の閲覧者向けに公開中の投稿で使われている ACE SPEC を投稿数の多い順に返す。
// 同じカードでも収録セットごとに card_id が違うため、カード名で束ねて数える。
// 画像URLは収録セットによって違うので、代表として1つ(MAX)を採る。
func (i *DeckCodePost) FindAceSpecCounts(
	ctx context.Context,
	filter *repository.DeckCodePostFilter,
) ([]*entity.DeckCodePostAceSpecCount, error) {
	q := dbFromContext(ctx, i.db).WithContext(ctx).
		Table("deck_code_posts").
		Select(`
			deck_code_posts.ace_spec_card_name AS card_name,
			MAX(deck_code_posts.ace_spec_image_url) AS image_url,
			COUNT(*) AS count
		`).
		Joins("JOIN decks ON decks.id = deck_code_posts.deck_id AND decks.deleted_at IS NULL").
		Where(deckCodePostVisibleCondition).
		Where("deck_code_posts.ace_spec_card_name <> ''")
	q = applyPeriod(q, filter)

	var rows []*struct {
		CardName string
		ImageURL string
		Count    int
	}
	if tx := q.Group("deck_code_posts.ace_spec_card_name").
		Order("count DESC, card_name ASC").
		Scan(&rows); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	ret := make([]*entity.DeckCodePostAceSpecCount, 0, len(rows))
	for _, r := range rows {
		ret = append(ret, &entity.DeckCodePostAceSpecCount{
			CardName: r.CardName, ImageURL: r.ImageURL, Count: r.Count,
		})
	}

	return ret, nil
}

func (i *DeckCodePost) FindById(
	ctx context.Context,
	id string,
	viewerUserId string,
) (*entity.DeckCodePost, error) {
	return i.findOne(ctx, i.baseQuery(ctx, viewerUserId).Where("deck_code_posts.id = ?", id))
}

func (i *DeckCodePost) FindLiteById(
	ctx context.Context,
	id string,
) (*entity.DeckCodePost, error) {
	var m *model.DeckCodePost
	if tx := dbFromContext(ctx, i.db).Where("id = ?", id).First(&m); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, wrapError(tx.Error)
	}

	return toDeckCodePostEntity(m), nil
}

func (i *DeckCodePost) FindActiveByDeckCodeId(
	ctx context.Context,
	deckCodeId string,
) (*entity.DeckCodePost, error) {
	return i.findOne(ctx, i.baseQuery(ctx, "").
		Where("deck_code_posts.deck_code_id = ?", deckCodeId).
		Where(deckCodePostOccupyingCondition))
}

func (i *DeckCodePost) FindLatestByDeckCodeId(
	ctx context.Context,
	deckCodeId string,
) (*entity.DeckCodePost, error) {
	var m *model.DeckCodePost
	if tx := dbFromContext(ctx, i.db).
		Where("deck_code_id = ?", deckCodeId).
		Order("published_at DESC").
		First(&m); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, wrapError(tx.Error)
	}

	return toDeckCodePostEntity(m), nil
}

func (i *DeckCodePost) FindActiveByDeckId(
	ctx context.Context,
	deckId string,
) ([]*entity.DeckCodePost, error) {
	return i.findMany(ctx, i.baseQuery(ctx, "").
		Where("deck_code_posts.deck_id = ?", deckId).
		Where(deckCodePostOccupyingCondition).
		Order("deck_code_posts.published_at DESC"))
}

func (i *DeckCodePost) FindByUserId(
	ctx context.Context,
	uid string,
	viewerUserId string,
	limit int,
	offset int,
) ([]*entity.DeckCodePost, error) {
	return i.findMany(ctx, i.baseQuery(ctx, viewerUserId).
		Where("deck_code_posts.user_id = ?", uid).
		Where(deckCodePostVisibleCondition).
		Order("deck_code_posts.published_at DESC").
		Limit(limit).
		Offset(offset))
}

func (i *DeckCodePost) SummarizeByUserId(
	ctx context.Context,
	uid string,
) (*entity.DeckCodePostUserSummary, error) {
	var row struct {
		PostCount      int
		LikeCountTotal int
	}

	if tx := dbFromContext(ctx, i.db).WithContext(ctx).
		Table("deck_code_posts").
		Select("COUNT(DISTINCT deck_code_posts.id) AS post_count, COUNT(l.post_id) AS like_count_total").
		Joins("LEFT JOIN deck_code_post_likes l ON l.post_id = deck_code_posts.id").
		Where("deck_code_posts.user_id = ?", uid).
		Where(deckCodePostVisibleCondition).
		Scan(&row); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	return &entity.DeckCodePostUserSummary{PostCount: row.PostCount, LikeCountTotal: row.LikeCountTotal}, nil
}

func (i *DeckCodePost) Save(
	ctx context.Context,
	e *entity.DeckCodePost,
) error {
	m := model.NewDeckCodePost(
		e.ID, e.CreatedAt, e.UpdatedAt, e.UserId, e.DeckId, e.DeckCodeId, e.PublishedAt,
		timeOrNil(e.UnpublishedAt), timeOrNil(e.HiddenAt),
		e.AceSpecCardId, e.AceSpecCardName, e.AceSpecImageURL,
	)

	if err := dbFromContext(ctx, i.db).Save(m).Error; err != nil {
		logError(ctx, err)
		// 同じコードの投稿が枠を占有している(部分一意索引)ときは、呼び出し側が既存の投稿を返せるようにする
		return wrapUniqueViolation(err)
	}

	return nil
}

// unpublishWhere は条件に合う取り下げていない投稿を取り下げ、そのいいねを消す。
// 取り下げた投稿のいいねは「公開し直しても戻らない」仕様なので、行ごと消す。
func (i *DeckCodePost) unpublishWhere(
	ctx context.Context,
	unpublishedAt time.Time,
	query string,
	args ...any,
) error {
	return dbFromContext(ctx, i.db).Transaction(func(tx *gorm.DB) error {
		// いいねは投稿を取り下げる前に消す(取り下げ後は unpublished_at IS NULL の条件で拾えなくなる)。
		targets := tx.Model(&model.DeckCodePost{}).Select("id").Where(query, args...).Where("unpublished_at IS NULL")
		if err := tx.Where("post_id IN (?)", targets).Delete(&model.DeckCodePostLike{}).Error; err != nil {
			logError(ctx, err)
			return err
		}

		if err := tx.Model(&model.DeckCodePost{}).
			Where(query, args...).
			Where("unpublished_at IS NULL").
			Updates(map[string]any{
				"unpublished_at": unpublishedAt,
				"updated_at":     unpublishedAt,
			}).Error; err != nil {
			logError(ctx, err)
			return err
		}

		return nil
	})
}

func (i *DeckCodePost) Unpublish(
	ctx context.Context,
	id string,
	unpublishedAt time.Time,
) error {
	return i.unpublishWhere(ctx, unpublishedAt, "id = ?", id)
}

func (i *DeckCodePost) UnpublishByDeckId(
	ctx context.Context,
	deckId string,
	unpublishedAt time.Time,
) error {
	return i.unpublishWhere(ctx, unpublishedAt, "deck_id = ?", deckId)
}

func (i *DeckCodePost) UnpublishByDeckCodeId(
	ctx context.Context,
	deckCodeId string,
	unpublishedAt time.Time,
) error {
	return i.unpublishWhere(ctx, unpublishedAt, "deck_code_id = ?", deckCodeId)
}

func (i *DeckCodePost) Like(
	ctx context.Context,
	postId string,
	uid string,
	createdAt time.Time,
) error {
	// 二重押しは主キー違反ではなく「何もしない」で受ける。
	if err := dbFromContext(ctx, i.db).Exec(
		"INSERT INTO deck_code_post_likes (post_id, user_id, created_at) VALUES (?, ?, ?) ON CONFLICT DO NOTHING",
		postId, uid, createdAt,
	).Error; err != nil {
		logError(ctx, err)
		return err
	}

	return nil
}

func (i *DeckCodePost) Unlike(
	ctx context.Context,
	postId string,
	uid string,
) error {
	if err := dbFromContext(ctx, i.db).
		Where("post_id = ? AND user_id = ?", postId, uid).
		Delete(&model.DeckCodePostLike{}).Error; err != nil {
		logError(ctx, err)
		return err
	}

	return nil
}

func (i *DeckCodePost) RecordImport(
	ctx context.Context,
	postId string,
	uid string,
	createdAt time.Time,
) error {
	// 同じ人の2回目以降は主キー違反ではなく「何もしない」で受ける(回数を水増しできないようにする)。
	if err := dbFromContext(ctx, i.db).Exec(
		"INSERT INTO deck_code_post_imports (post_id, user_id, created_at) VALUES (?, ?, ?) ON CONFLICT DO NOTHING",
		postId, uid, createdAt,
	).Error; err != nil {
		logError(ctx, err)
		return err
	}

	return nil
}

func (i *DeckCodePost) FindLikers(
	ctx context.Context,
	postId string,
	limit int,
	offset int,
) ([]*entity.DeckCodePostLiker, error) {
	var rows []*deckCodePostLikerRow
	if tx := dbFromContext(ctx, i.db).WithContext(ctx).
		Table("deck_code_post_likes").
		Select(`
			deck_code_post_likes.post_id AS post_id,
			deck_code_post_likes.created_at AS created_at,
			users.id AS user_id,
			users.name AS user_name,
			users.image_url AS user_image_url,
			users.created_at AS user_created_at
		`).
		Joins("JOIN users ON users.id = deck_code_post_likes.user_id AND users.deleted_at IS NULL").
		Where("deck_code_post_likes.post_id = ?", postId).
		Order("deck_code_post_likes.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	ret := make([]*entity.DeckCodePostLiker, 0, len(rows))
	for _, r := range rows {
		ret = append(ret, &entity.DeckCodePostLiker{
			User:      entity.NewUser(r.UserId, r.UserCreatedAt, r.UserName, normalizeUserImageURL(r.UserImageURL)),
			CreatedAt: r.CreatedAt,
		})
	}

	return ret, nil
}

// FindLikeDigests は期間内のいいねを閲覧者向けに公開中の投稿ごとにまとめる(日次のまとめ通知用)。
// 「最後にいいねした人」は同じ期間・投稿のいいねを created_at の新しい順で1件引く。
// 投稿者自身のいいねは通知の対象にしないため除く。
func (i *DeckCodePost) FindLikeDigests(
	ctx context.Context,
	from time.Time,
	to time.Time,
) ([]*entity.DeckCodePostLikeDigest, error) {
	var rows []*struct {
		PostId          string
		OwnerUserId     string
		DeckName        string
		LikeCount       int
		LatestLikerName string
	}

	if tx := dbFromContext(ctx, i.db).WithContext(ctx).Raw(`
		SELECT
			p.id AS post_id,
			p.user_id AS owner_user_id,
			d.name AS deck_name,
			COUNT(*) AS like_count,
			COALESCE((
				SELECT u.name
				FROM deck_code_post_likes l2
				JOIN users u ON u.id = l2.user_id AND u.deleted_at IS NULL
				WHERE l2.post_id = p.id AND l2.user_id <> p.user_id AND l2.created_at >= ? AND l2.created_at < ?
				ORDER BY l2.created_at DESC
				LIMIT 1
			), '') AS latest_liker_name
		FROM deck_code_post_likes l
		JOIN deck_code_posts p ON p.id = l.post_id AND p.unpublished_at IS NULL AND p.hidden_at IS NULL
		JOIN decks d ON d.id = p.deck_id AND d.deleted_at IS NULL
		WHERE l.created_at >= ? AND l.created_at < ? AND l.user_id <> p.user_id
		GROUP BY p.id, p.user_id, d.name
		ORDER BY p.id
	`, from, to, from, to).Scan(&rows); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	ret := make([]*entity.DeckCodePostLikeDigest, 0, len(rows))
	for _, r := range rows {
		ret = append(ret, &entity.DeckCodePostLikeDigest{
			PostId:          r.PostId,
			OwnerUserId:     r.OwnerUserId,
			DeckName:        r.DeckName,
			LikeCount:       r.LikeCount,
			LatestLikerName: r.LatestLikerName,
		})
	}

	return ret, nil
}

// DeleteByUserId は退会時に、そのユーザに関わる投稿といいねを物理削除する。
//
// 消す投稿は「本人の投稿」と「本人のデッキに紐づく投稿」。後者は、他人が本人のデッキに
// 作ったコードで公開した投稿で、デッキの物理削除(cmd/purge-deleted-user-data)で FK に
// 阻まれないように一緒に消す。いいね・取り込み記録は投稿より先に消す(FK の子)。
func (i *DeckCodePost) DeleteByUserId(
	ctx context.Context,
	uid string,
) error {
	return dbFromContext(ctx, i.db).Transaction(func(tx *gorm.DB) error {
		// 本人が押したいいね・本人の取り込み記録
		if err := tx.Where("user_id = ?", uid).Delete(&model.DeckCodePostLike{}).Error; err != nil {
			logError(ctx, err)
			return err
		}
		if err := tx.Where("user_id = ?", uid).Delete(&model.DeckCodePostImport{}).Error; err != nil {
			logError(ctx, err)
			return err
		}

		// 本人の投稿と、本人のデッキに紐づく投稿(他人のコードによるものを含む)に付いたいいね・取り込み記録
		targets := tx.Model(&model.DeckCodePost{}).Select("id").
			Where("user_id = ? OR deck_id IN (SELECT id FROM decks WHERE user_id = ?)", uid, uid)
		if err := tx.Where("post_id IN (?)", targets).Delete(&model.DeckCodePostLike{}).Error; err != nil {
			logError(ctx, err)
			return err
		}
		if err := tx.Where("post_id IN (?)", targets).Delete(&model.DeckCodePostImport{}).Error; err != nil {
			logError(ctx, err)
			return err
		}
		if err := tx.Where("user_id = ? OR deck_id IN (SELECT id FROM decks WHERE user_id = ?)", uid, uid).
			Delete(&model.DeckCodePost{}).Error; err != nil {
			logError(ctx, err)
			return err
		}

		return nil
	})
}
