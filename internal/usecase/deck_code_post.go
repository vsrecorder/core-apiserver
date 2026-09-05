package usecase

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	// DeckCodePostRepublishInterval は同じデッキコードを公開し直せる間隔。
	// 取り下げ→公開でタイムラインの先頭へ上げ直す操作を抑えるための制限。
	DeckCodePostRepublishInterval = 24 * time.Hour

	// DeckCodePostPopularWindow は「人気」の並び順で数えるいいねの期間(直近7日間)。
	DeckCodePostPopularWindow = 7 * 24 * time.Hour

	// designationTierCacheTTL は投稿者の称号ティアをキャッシュする時間。
	// 称号の判定は記録・公式結果の集計を伴い1ユーザあたり数クエリかかるため、
	// 一覧のたびに投稿者全員分を計算しない。称号は日に何度も変わるものではないので
	// 多少古くても差し支えない。
	designationTierCacheTTL = 10 * time.Minute

	// designationTierCacheMaxEntries はキャッシュに保持するユーザ数の上限。
	// 超えたら期限切れを掃除し、それでも超えていれば全て捨てる(プロセス内メモリを
	// 無制限に増やさないための単純な方式)。
	designationTierCacheMaxEntries = 10000
)

// DeckCodePostFindParam は一覧の条件。
type DeckCodePostFindParam struct {
	// Sort は repository.DeckCodePostSortNew / Popular。空は新しい順。
	Sort string
	// EnvironmentId を指定するとその環境の期間に公開された投稿に絞る。
	// 空なら現在の環境(今日が属する期間)。
	EnvironmentId string
	// AceSpecCardName を指定するとその ACE SPEC を採用した投稿に絞る(収録セット違いも含む)。
	AceSpecCardName string
	// PokemonSpriteIds を指定するとそれらをすべて持つデッキの投稿に絞る(最大2体)。
	PokemonSpriteIds []string
	// ViewerUserId は閲覧者(未ログインなら空)。いいね済みの判定に使う。
	ViewerUserId string
	Limit        int
	Offset       int
}

// DeckCodePostAceSpecCountsResult は ACE SPEC での絞り込み候補。絞り込みに使った環境も返す。
type DeckCodePostAceSpecCountsResult struct {
	Environment *entity.Environment
	AceSpecs    []*entity.DeckCodePostAceSpecCount
}

// DeckCodePostFindResult は一覧の応答。絞り込みに使った環境も返し、
// 画面のチップ表示に使う(環境が未登録の期間では nil)。
type DeckCodePostFindResult struct {
	Environment *entity.Environment
	Posts       []*entity.DeckCodePost
}

// DeckCodePostUserView は投稿者ページの応答。
type DeckCodePostUserView struct {
	User            *entity.User
	DesignationTier int
	Summary         *entity.DeckCodePostUserSummary
	Posts           []*entity.DeckCodePost
}

type DeckCodePostInterface interface {
	Find(
		ctx context.Context,
		param *DeckCodePostFindParam,
	) (*DeckCodePostFindResult, error)

	// FindAceSpecCounts は環境内の公開中の投稿で使われている ACE SPEC を投稿数順に返す。
	// environmentId が空なら現在の環境。
	FindAceSpecCounts(
		ctx context.Context,
		environmentId string,
	) (*DeckCodePostAceSpecCountsResult, error)

	// FindById は取り下げ済み・非表示の投稿も返す(個別ページの 410 判定は controller が行う)。
	FindById(
		ctx context.Context,
		id string,
		viewerUserId string,
	) (*entity.DeckCodePost, error)

	// FindActiveByDeckId はデッキの取り下げていない投稿(全バージョン分)を返す。
	FindActiveByDeckId(
		ctx context.Context,
		deckId string,
	) ([]*entity.DeckCodePost, error)

	// FindByUserId は投稿者ページの内容を返す。公開中の投稿が1件も無いユーザでも、
	// 投稿者の公開情報(名前・アイコン・称号)と0件の集計を返す。
	// ユーザ自体が無ければ ErrRecordNotFound。
	FindByUserId(
		ctx context.Context,
		uid string,
		viewerUserId string,
		limit int,
		offset int,
	) (*DeckCodePostUserView, error)

	// FindLikers は公開中の投稿にいいねした人を返す。投稿が無い・公開中でなければ ErrRecordNotFound。
	FindLikers(
		ctx context.Context,
		postId string,
		limit int,
		offset int,
	) ([]*entity.DeckCodePostLiker, error)

	// Publish は uid のデッキコードをみんなの公開デッキに載せる。
	// 既に公開中(取り下げていない)ならその投稿を返す(冪等)。
	// 運営が非表示にしたコードは ErrDeckCodePostHidden、24時間以内の公開し直しは ErrRepublishTooSoon。
	Publish(
		ctx context.Context,
		uid string,
		deckCodeId string,
	) (*entity.DeckCodePost, error)

	// Unpublish は投稿を取り下げる。取り下げ済みなら何もしない。
	Unpublish(
		ctx context.Context,
		id string,
	) error

	// Like は公開中の投稿にいいねし、いいね後の投稿を返す。公開中でなければ ErrRecordNotFound。
	Like(
		ctx context.Context,
		id string,
		uid string,
	) (*entity.DeckCodePost, error)

	// Unlike はいいねを取り消し、取り消し後の投稿を返す。
	Unlike(
		ctx context.Context,
		id string,
		uid string,
	) (*entity.DeckCodePost, error)

	// RecordImport は uid が「取り込む」を使ったことを記録する(同じ人は1回として数える)。
	// 公開中の投稿だけを対象にし、そうでなければ ErrRecordNotFound。
	RecordImport(
		ctx context.Context,
		id string,
		uid string,
	) error
}

// designationTierCache は投稿者の称号ティアの短期キャッシュ(プロセス内)。
type designationTierCache struct {
	mu      sync.Mutex
	entries map[string]designationTierCacheEntry
}

type designationTierCacheEntry struct {
	tier      int
	expiresAt time.Time
}

func newDesignationTierCache() *designationTierCache {
	return &designationTierCache{entries: map[string]designationTierCacheEntry{}}
}

func (c *designationTierCache) get(uid string, now time.Time) (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.entries[uid]
	if !ok || now.After(e.expiresAt) {
		return 0, false
	}

	return e.tier, true
}

func (c *designationTierCache) set(uid string, tier int, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= designationTierCacheMaxEntries {
		for k, e := range c.entries {
			if now.After(e.expiresAt) {
				delete(c.entries, k)
			}
		}
		if len(c.entries) >= designationTierCacheMaxEntries {
			c.entries = map[string]designationTierCacheEntry{}
		}
	}

	c.entries[uid] = designationTierCacheEntry{tier: tier, expiresAt: now.Add(designationTierCacheTTL)}
}

type DeckCodePost struct {
	repository             repository.DeckCodePostInterface
	deckRepository         repository.DeckInterface
	deckCodeRepository     repository.DeckCodeInterface
	userRepository         repository.UserInterface
	environmentRepository  repository.EnvironmentInterface
	championshipSeriesRepo repository.ChampionshipSeriesInterface
	// designation は投稿者の称号ティアを引くために使う。nil なら称号は出さない(0)。
	designation DesignationInterface
	// deckCard は公開時の ACE SPEC 判定(deckcard-api)。nil なら判定しない。
	deckCard  repository.DeckCardInterface
	tierCache *designationTierCache
}

func NewDeckCodePost(
	repository repository.DeckCodePostInterface,
	deckRepository repository.DeckInterface,
	deckCodeRepository repository.DeckCodeInterface,
	userRepository repository.UserInterface,
	environmentRepository repository.EnvironmentInterface,
	championshipSeriesRepo repository.ChampionshipSeriesInterface,
	designation DesignationInterface,
	deckCard repository.DeckCardInterface,
) DeckCodePostInterface {
	return &DeckCodePost{
		repository:             repository,
		deckRepository:         deckRepository,
		deckCodeRepository:     deckCodeRepository,
		userRepository:         userRepository,
		environmentRepository:  environmentRepository,
		championshipSeriesRepo: championshipSeriesRepo,
		designation:            designation,
		deckCard:               deckCard,
		tierCache:              newDesignationTierCache(),
	}
}

// environmentPeriod は環境の期間を [from_date 0時, to_date の翌日 0時) の半開区間で返す。
// DATE 型の値はドライバが UTC の 0 時として返すため、年月日だけを取り出して
// ローカル(JST)の 0 時に組み直す(既存の PeriodDateRange と同じ扱い)。
func environmentPeriod(env *entity.Environment) (time.Time, time.Time) {
	fy, fm, fd := env.FromDate.Date()
	ty, tm, td := env.ToDate.Date()
	from := time.Date(fy, fm, fd, 0, 0, 0, 0, time.Local)
	to := time.Date(ty, tm, td, 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)

	return from, to
}

// resolvedEnvironment は一覧の対象環境と、公開日時の絞り込み範囲。
type resolvedEnvironment struct {
	env  *entity.Environment
	from time.Time
	to   time.Time
}

// resolveEnvironment は一覧の対象環境を決める。指定が無ければ今日が属する環境。
// 指定された環境が無ければ ErrRecordNotFound。今日に対応する環境が無い場合は env が nil(絞り込みなし)。
//
// 現在の環境(今日が属する、開始日が最新の環境)については終了日で区切らない。
// 環境テーブルは次の環境が登録されるまで更新されないことがあり、終了日を過ぎた後に
// 公開した投稿が「どの環境にも属さない」ことになって一覧から消えるのを防ぐ。
// (次の環境が始まるまでは、今公開される投稿はすべて現在の環境のものになる。)
func (u *DeckCodePost) resolveEnvironment(ctx context.Context, environmentId string) (*resolvedEnvironment, error) {
	var env *entity.Environment
	if environmentId != "" {
		found, err := u.environmentRepository.FindById(ctx, environmentId)
		if err != nil {
			logError(ctx, err)
			return nil, err
		}
		env = found
	}

	current, err := u.environmentRepository.FindByDate(ctx, timeNow())
	if err != nil && !errors.Is(err, apperror.ErrRecordNotFound) {
		logError(ctx, err)
		return nil, err
	}

	if env == nil {
		env = current
	}
	if env == nil {
		return &resolvedEnvironment{}, nil
	}

	from, to := environmentPeriod(env)
	if current != nil && current.ID == env.ID {
		to = time.Time{}
	}

	return &resolvedEnvironment{env: env, from: from, to: to}, nil
}

// designationTiers は uids それぞれの現在の称号ティアを返す。判定に失敗しても一覧を止めないため、
// エラーはログに残して 0(称号なし)として扱う。現在のシーズンは1回だけ引く。
func (u *DeckCodePost) designationTiers(ctx context.Context, uids []string) map[string]int {
	tiers := make(map[string]int, len(uids))
	if u.designation == nil || u.championshipSeriesRepo == nil {
		return tiers
	}

	now := timeNow()
	missing := make([]string, 0, len(uids))
	for _, uid := range uids {
		if _, ok := tiers[uid]; ok {
			continue
		}
		if tier, ok := u.tierCache.get(uid, now); ok {
			tiers[uid] = tier
			continue
		}
		tiers[uid] = 0
		missing = append(missing, uid)
	}
	if len(missing) == 0 {
		return tiers
	}

	season, err := CurrentSeasonLabel(ctx, u.championshipSeriesRepo, now)
	if err != nil {
		logWarn(ctx, err)
		return tiers
	}

	for _, uid := range missing {
		view, err := u.designation.GetByUserId(ctx, uid, season)
		if err != nil {
			logWarn(ctx, err)
			continue
		}

		tier := 0
		if view != nil && view.Current != nil {
			tier = view.Current.Tier
		}
		u.tierCache.set(uid, tier, now)
		tiers[uid] = tier
	}

	return tiers
}

// designationTier は1人分の designationTiers。
func (u *DeckCodePost) designationTier(ctx context.Context, uid string) int {
	return u.designationTiers(ctx, []string{uid})[uid]
}

// attachDesignationTiers は投稿者ごとに称号ティアを引いて詰める。
func (u *DeckCodePost) attachDesignationTiers(ctx context.Context, posts []*entity.DeckCodePost) {
	uids := make([]string, 0, len(posts))
	for _, post := range posts {
		uids = append(uids, post.UserId)
	}

	tiers := u.designationTiers(ctx, uids)
	for _, post := range posts {
		post.DesignationTier = tiers[post.UserId]
	}
}

func (u *DeckCodePost) Find(
	ctx context.Context,
	param *DeckCodePostFindParam,
) (*DeckCodePostFindResult, error) {
	resolved, err := u.resolveEnvironment(ctx, param.EnvironmentId)
	if err != nil {
		return nil, err
	}

	filter := &repository.DeckCodePostFilter{
		Sort:             param.Sort,
		From:             resolved.from,
		To:               resolved.to,
		PopularSince:     timeNow().Add(-DeckCodePostPopularWindow),
		AceSpecCardName:  param.AceSpecCardName,
		PokemonSpriteIds: param.PokemonSpriteIds,
		ViewerUserId:     param.ViewerUserId,
	}

	posts, err := u.repository.Find(ctx, filter, param.Limit, param.Offset)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	u.attachDesignationTiers(ctx, posts)

	return &DeckCodePostFindResult{Environment: resolved.env, Posts: posts}, nil
}

func (u *DeckCodePost) FindAceSpecCounts(
	ctx context.Context,
	environmentId string,
) (*DeckCodePostAceSpecCountsResult, error) {
	resolved, err := u.resolveEnvironment(ctx, environmentId)
	if err != nil {
		return nil, err
	}

	aceSpecs, err := u.repository.FindAceSpecCounts(ctx, &repository.DeckCodePostFilter{From: resolved.from, To: resolved.to})
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return &DeckCodePostAceSpecCountsResult{Environment: resolved.env, AceSpecs: aceSpecs}, nil
}

func (u *DeckCodePost) FindById(
	ctx context.Context,
	id string,
	viewerUserId string,
) (*entity.DeckCodePost, error) {
	post, err := u.repository.FindById(ctx, id, viewerUserId)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	post.DesignationTier = u.designationTier(ctx, post.UserId)

	return post, nil
}

func (u *DeckCodePost) FindActiveByDeckId(
	ctx context.Context,
	deckId string,
) ([]*entity.DeckCodePost, error) {
	posts, err := u.repository.FindActiveByDeckId(ctx, deckId)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return posts, nil
}

func (u *DeckCodePost) FindByUserId(
	ctx context.Context,
	uid string,
	viewerUserId string,
	limit int,
	offset int,
) (*DeckCodePostUserView, error) {
	user, err := u.userRepository.FindById(ctx, uid)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	summary, err := u.repository.SummarizeByUserId(ctx, uid)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	posts, err := u.repository.FindByUserId(ctx, uid, viewerUserId, limit, offset)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	tier := u.designationTier(ctx, uid)
	for _, post := range posts {
		post.DesignationTier = tier
	}

	return &DeckCodePostUserView{User: user, DesignationTier: tier, Summary: summary, Posts: posts}, nil
}

func (u *DeckCodePost) FindLikers(
	ctx context.Context,
	postId string,
	limit int,
	offset int,
) ([]*entity.DeckCodePostLiker, error) {
	// 取り下げ済み・非表示の投稿のいいねは公開しない(投稿が消えた後に誰が押したかを引けないようにする)。
	post, err := u.repository.FindLiteById(ctx, postId)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}
	if !post.IsActive() {
		return nil, apperror.ErrRecordNotFound
	}

	likers, err := u.repository.FindLikers(ctx, postId, limit, offset)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	uids := make([]string, 0, len(likers))
	for _, liker := range likers {
		uids = append(uids, liker.User.ID)
	}
	tiers := u.designationTiers(ctx, uids)
	for _, liker := range likers {
		liker.DesignationTier = tiers[liker.User.ID]
	}

	return likers, nil
}

// findAceSpec は公開時の ACE SPEC 判定。deckcard-api の失敗で公開を止めないため、
// 失敗は警告ログに残して「判定なし」を返す(webapp は保存値が空の投稿だけ acespec API で補う)。
func (u *DeckCodePost) findAceSpec(ctx context.Context, code string) *entity.AceSpecCard {
	if u.deckCard == nil || code == "" {
		return nil
	}

	card, err := u.deckCard.FindAceSpec(ctx, code)
	if err != nil {
		logWarn(ctx, err)
		return nil
	}

	return card
}

func (u *DeckCodePost) Publish(
	ctx context.Context,
	uid string,
	deckCodeId string,
) (*entity.DeckCodePost, error) {
	deckCode, err := u.deckCodeRepository.FindById(ctx, deckCodeId)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}
	// 他人のデッキコードは「存在しない」として扱い、IDの存在を教えない。
	if deckCode.UserId != uid {
		return nil, apperror.ErrRecordNotFound
	}

	deck, err := u.deckRepository.FindById(ctx, deckCode.DeckId)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}
	// コードだけでなくデッキも本人のものに限る。投稿はデッキに紐づき、デッキの持ち主の
	// 退会・削除に連動して消えるため、他人のデッキに作ったコードを公開させない。
	if deck.UserId != uid {
		return nil, apperror.ErrRecordNotFound
	}
	// アーカイブしたデッキは公開できない(アーカイブ時に取り下げる仕様と揃える)。
	if !deck.ArchivedAt.IsZero() {
		return nil, apperror.ErrDeckArchived
	}

	// 既に公開中(取り下げていない)ならその投稿を返す(スイッチの二重操作を冪等にする)。
	existing, err := u.repository.FindActiveByDeckCodeId(ctx, deckCodeId)
	if err == nil {
		existing.DesignationTier = u.designationTier(ctx, uid)
		return existing, nil
	}
	if !errors.Is(err, apperror.ErrRecordNotFound) {
		logError(ctx, err)
		return nil, err
	}

	now := timeNow()

	latest, err := u.repository.FindLatestByDeckCodeId(ctx, deckCodeId)
	if err != nil && !errors.Is(err, apperror.ErrRecordNotFound) {
		logError(ctx, err)
		return nil, err
	}
	// 運営が非表示にした投稿は、取り下げて別の投稿として作り直しても表示に戻せない
	// (解除は cmd/hide-deck-code-post -unhide で運営が行う)。
	if latest != nil && !latest.HiddenAt.IsZero() {
		return nil, apperror.ErrDeckCodePostHidden
	}
	// 取り下げ→公開でタイムラインの先頭へ上げ直す操作を抑える。
	if latest != nil && now.Sub(latest.PublishedAt) < DeckCodePostRepublishInterval {
		return nil, apperror.ErrRepublishTooSoon
	}

	id, err := generateId()
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	aceSpecCardId, aceSpecCardName, aceSpecImageURL := "", "", ""
	if card := u.findAceSpec(ctx, deckCode.Code); card != nil {
		aceSpecCardId, aceSpecCardName, aceSpecImageURL = card.CardId, card.CardName, card.ImageURL
	}

	post := entity.NewDeckCodePost(
		id, now, now, uid, deck.ID, deckCode.ID, now,
		time.Time{}, time.Time{},
		aceSpecCardId, aceSpecCardName, 0,
	)
	post.AceSpecImageURL = aceSpecImageURL

	if err := u.repository.Save(ctx, post); err != nil {
		// 同時に2回押されて先に公開された(部分一意索引に当たった)場合は、その投稿を返す。
		if errors.Is(err, apperror.ErrAlreadyExists) {
			existing, findErr := u.repository.FindActiveByDeckCodeId(ctx, deckCodeId)
			if findErr != nil {
				logError(ctx, findErr)
				return nil, findErr
			}
			existing.DesignationTier = u.designationTier(ctx, uid)
			return existing, nil
		}

		logError(ctx, err)
		return nil, err
	}

	// 投稿者・デッキ名・スプライトを詰めた形で返す。
	return u.FindById(ctx, id, uid)
}

func (u *DeckCodePost) Unpublish(
	ctx context.Context,
	id string,
) error {
	post, err := u.repository.FindLiteById(ctx, id)
	if err != nil {
		logError(ctx, err)
		return err
	}
	if !post.UnpublishedAt.IsZero() {
		return nil
	}

	if err := u.repository.Unpublish(ctx, id, timeNow()); err != nil {
		logError(ctx, err)
		return err
	}

	return nil
}

func (u *DeckCodePost) Like(
	ctx context.Context,
	id string,
	uid string,
) (*entity.DeckCodePost, error) {
	post, err := u.repository.FindLiteById(ctx, id)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}
	// 取り下げ済み・非表示の投稿にはいいねできない。
	if !post.IsActive() {
		return nil, apperror.ErrRecordNotFound
	}

	if err := u.repository.Like(ctx, id, uid, timeNow()); err != nil {
		logError(ctx, err)
		return nil, err
	}

	return u.FindById(ctx, id, uid)
}

func (u *DeckCodePost) Unlike(
	ctx context.Context,
	id string,
	uid string,
) (*entity.DeckCodePost, error) {
	// 存在しない投稿への取り消しは何も消さず、続く FindById が NotFound を返す。
	if err := u.repository.Unlike(ctx, id, uid); err != nil {
		logError(ctx, err)
		return nil, err
	}

	return u.FindById(ctx, id, uid)
}

func (u *DeckCodePost) RecordImport(
	ctx context.Context,
	id string,
	uid string,
) error {
	post, err := u.repository.FindLiteById(ctx, id)
	if err != nil {
		logError(ctx, err)
		return err
	}
	// 取り下げ済み・非表示の投稿の取り込みは数えない。
	if !post.IsActive() {
		return apperror.ErrRecordNotFound
	}

	if err := u.repository.RecordImport(ctx, id, uid, timeNow()); err != nil {
		logError(ctx, err)
		return err
	}

	return nil
}
