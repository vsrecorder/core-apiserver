package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// MaxFavoriteDecksPerUser は1ユーザがお気に入りにできるデッキの数。
//
// お気に入りは user_favorite_decks テーブルで管理していて、DB側には件数の制約を
// 置いていない。増やすときはこの値を変えるだけでよく、上限を超えた分は
// お気に入りにした日時が古いものから外れる。
const MaxFavoriteDecksPerUser = 1

type DeckCreateParam struct {
	UserId             string
	Name               string
	PrivateFlg         bool
	DeckCode           string
	PrivateDeckCodeFlg bool
	PokemonSprites     []*PokemonSpriteParam
	TagIds             []string
}

func NewDeckCreateParam(
	userId string,
	name string,
	privateFlg bool,
	deckcode string,
	privateDeckCodeFlg bool,
	pokemonSprites []*PokemonSpriteParam,
	tagIds []string,
) *DeckCreateParam {
	return &DeckCreateParam{
		UserId:             userId,
		Name:               name,
		PrivateFlg:         privateFlg,
		DeckCode:           deckcode,
		PrivateDeckCodeFlg: privateDeckCodeFlg,
		PokemonSprites:     pokemonSprites,
		TagIds:             tagIds,
	}
}

type DeckUpdateParam struct {
	Name           string
	PrivateFlg     bool
	PokemonSprites []*PokemonSpriteParam
	TagIds         []string
}

func NewDeckUpdateParam(
	name string,
	privateFlg bool,
	pokemonSprites []*PokemonSpriteParam,
	tagIds []string,
) *DeckUpdateParam {
	return &DeckUpdateParam{
		Name:           name,
		PrivateFlg:     privateFlg,
		PokemonSprites: pokemonSprites,
		TagIds:         tagIds,
	}
}

type DeckInterface interface {
	Find(
		ctx context.Context,
		limit int,
		offset int,
	) ([]*entity.Deck, error)

	FindAll(
		ctx context.Context,
		uid string,
	) ([]*entity.Deck, error)

	FindOnCursor(
		ctx context.Context,
		limit int,
		cursor time.Time,
	) ([]*entity.Deck, error)

	FindById(
		ctx context.Context,
		id string,
	) (*entity.Deck, error)

	FindByUserId(
		ctx context.Context,
		uid string,
		archivedFlg bool,
		limit int,
		offset int,
	) ([]*entity.Deck, error)

	FindByUserIdOnCursor(
		ctx context.Context,
		uid string,
		archivedFlg bool,
		limit int,
		cursor time.Time,
	) ([]*entity.Deck, error)

	Create(
		ctx context.Context,
		param *DeckCreateParam,
	) (*entity.Deck, error)

	Update(
		ctx context.Context,
		id string,
		param *DeckUpdateParam,
	) (*entity.Deck, error)

	Archive(
		ctx context.Context,
		id string,
	) (*entity.Deck, error)

	Unarchive(
		ctx context.Context,
		id string,
	) (*entity.Deck, error)

	Favorite(
		ctx context.Context,
		id string,
	) (*entity.Deck, error)

	Unfavorite(
		ctx context.Context,
		id string,
	) (*entity.Deck, error)

	Delete(
		ctx context.Context,
		id string,
	) error
}

type Deck struct {
	repository         repository.DeckInterface
	deckAsset          repository.DeckAssetInterface
	userFavoriteDeck   repository.UserFavoriteDeckInterface
	tag                repository.TagInterface
	transactionManager repository.TransactionManager
	badgeEvaluation    BadgeEvaluationInterface
}

func NewDeck(
	repository repository.DeckInterface,
	deckAsset repository.DeckAssetInterface,
	userFavoriteDeck repository.UserFavoriteDeckInterface,
	tag repository.TagInterface,
	transactionManager repository.TransactionManager,
	badgeEvaluation BadgeEvaluationInterface,
) DeckInterface {
	return &Deck{
		repository,
		deckAsset,
		userFavoriteDeck,
		tag,
		transactionManager,
		badgeEvaluation,
	}
}

// syncDeckTags は付与先について、param で指定されたタグIDのうち userId が付与できる
// 有効なタグ(自分のタグ or プリセット)だけを残して中間テーブルを更新し、付与後のタグを返す。
// 他人のタグIDや存在しないIDは黙って捨てる(フロントからは付与可能なタグしか送られない想定で、
// 不正な値はエラーにせず無視して防御する)。
func (u *Deck) syncDeckTags(
	ctx context.Context,
	deckId string,
	userId string,
	tagIds []string,
) ([]*entity.Tag, error) {
	tags, err := u.tag.FindAttachableByIds(ctx, tagIds, userId)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	// FindAttachableByIds の戻り順は不定なので、付与順(tagIds)に整列してから採番する。
	orderedTags, attachableTagIds := orderAttachableTagsByIds(tags, tagIds)

	if err := u.tag.ReplaceDeckTags(ctx, deckId, attachableTagIds); err != nil {
		logError(ctx, err)
		return nil, err
	}

	return orderedTags, nil
}

func (u *Deck) Find(
	ctx context.Context,
	limit int,
	offset int,
) ([]*entity.Deck, error) {
	decks, err := u.repository.Find(ctx, limit, offset)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return decks, nil
}

func (u *Deck) FindAll(
	ctx context.Context,
	uid string,
) ([]*entity.Deck, error) {
	decks, err := u.repository.FindAll(ctx, uid)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return decks, nil
}

func (u *Deck) FindOnCursor(
	ctx context.Context,
	limit int,
	cursor time.Time,
) ([]*entity.Deck, error) {
	decks, err := u.repository.FindOnCursor(ctx, limit, cursor)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return decks, nil
}

func (u *Deck) FindById(
	ctx context.Context,
	id string,
) (*entity.Deck, error) {
	deck, err := u.repository.FindById(ctx, id)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return deck, nil
}

func (u *Deck) FindByUserId(
	ctx context.Context,
	uid string,
	archivedFlg bool,
	limit int,
	offset int,
) ([]*entity.Deck, error) {
	decks, err := u.repository.FindByUserId(ctx, uid, archivedFlg, limit, offset)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return decks, nil
}

func (u *Deck) FindByUserIdOnCursor(
	ctx context.Context,
	uid string,
	archivedFlg bool,
	limit int,
	cursor time.Time,
) ([]*entity.Deck, error) {
	decks, err := u.repository.FindByUserIdOnCursor(ctx, uid, archivedFlg, limit, cursor)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return decks, nil
}

func (u *Deck) Create(
	ctx context.Context,
	param *DeckCreateParam,
) (*entity.Deck, error) {
	deckId, err := generateId()
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	createdAt := time.Now().Local()
	archivedAt := time.Time{}
	favoritedAt := time.Time{}
	LatestDeckCode := entity.NewDeckCode("", time.Time{}, "", "", "", false, "")

	if param.DeckCode != "" {
		deckCodeId, err := generateId()
		if err != nil {
			logError(ctx, err)
			return nil, err
		}

		memo := ""
		LatestDeckCode = entity.NewDeckCode(
			deckCodeId,
			createdAt,
			param.UserId,
			deckId,
			param.DeckCode,
			param.PrivateDeckCodeFlg,
			memo,
		)

		if LatestDeckCode.Code != "" {
			// 先にデッキコードのHTMLページをアップロードする。
			// デッキ画像のアップロードでは、そのデッキコードが存在するか確認することができないが、
			// デッキコードのHTMLページのアップロードでは、デッキコードが正しいかどうかを確認することができるため、
			// 先にデッキコードのHTMLページをアップロードすることで、デッキコードが正しいかどうかを確認することができる。
			// アップロードに失敗した場合はデッキ作成を中止する。
			if err := u.deckAsset.UploadDeckResultHTML(ctx, LatestDeckCode.Code); err != nil {
				logError(ctx, err)
				return nil, err
			}

			if err := u.deckAsset.UploadDeckImage(ctx, LatestDeckCode.Code); err != nil {
				logError(ctx, err)
				return nil, err
			}
		}
	}

	var pokemonSprites []*entity.PokemonSprite
	for _, pokemonSprite := range param.PokemonSprites {
		pokemonSprites = append(
			pokemonSprites,
			entity.NewPokemonSpriteWithPosition(pokemonSprite.ID, pokemonSprite.Position),
		)
	}

	deck := entity.NewDeck(
		deckId,
		createdAt,
		archivedAt,
		favoritedAt,
		param.UserId,
		param.Name,
		param.PrivateFlg,
		LatestDeckCode,
		pokemonSprites,
	)

	if err := u.repository.Save(ctx, deck); err != nil {
		logError(ctx, err)
		return nil, err
	}

	// タグの付与はデッキ本体とは別テーブルのため Save とは分けて反映する。
	tags, err := u.syncDeckTags(ctx, deck.ID, param.UserId, param.TagIds)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}
	deck.Tags = tags

	// デッキ登録と同時にデッキコード(バージョン)も作られ、かつタグが付与された場合は、
	// 同じタグをそのデッキコードにも付ける。tags は所有権チェック済みのものを流用する。
	if LatestDeckCode.ID != "" && len(tags) > 0 {
		tagIds := make([]string, 0, len(tags))
		for _, tag := range tags {
			tagIds = append(tagIds, tag.ID)
		}
		if err := u.tag.ReplaceDeckCodeTags(ctx, LatestDeckCode.ID, tagIds); err != nil {
			logError(ctx, err)
			return nil, err
		}
		// deck.LatestDeckCode は LatestDeckCode と同じ実体を指すため、レスポンスにも反映される。
		LatestDeckCode.Tags = tags
	}

	if _, err := u.badgeEvaluation.EvaluateOnDeckCreated(ctx, param.UserId, deck); err != nil {
		logError(ctx, err)
		return nil, err
	}

	return deck, nil
}

func (u *Deck) Update(
	ctx context.Context,
	id string,
	param *DeckUpdateParam,
) (*entity.Deck, error) {
	// 指定されたidのDeckが存在するか確認
	ret, err := u.repository.FindById(ctx, id)
	if err == apperror.ErrRecordNotFound {
		return nil, err
	} else if err != nil {
		logError(ctx, err)
		return nil, err
	}

	var pokemonSprites []*entity.PokemonSprite
	for _, pokemonSprite := range param.PokemonSprites {
		pokemonSprites = append(
			pokemonSprites,
			entity.NewPokemonSpriteWithPosition(pokemonSprite.ID, pokemonSprite.Position),
		)
	}

	deck := entity.NewDeck(
		id,
		ret.CreatedAt,
		ret.ArchivedAt,
		ret.FavoritedAt,
		ret.UserId,
		param.Name,
		param.PrivateFlg,
		ret.LatestDeckCode,
		pokemonSprites,
	)

	if err := u.repository.Save(ctx, deck); err != nil {
		logError(ctx, err)
		return nil, err
	}

	// タグの付与を param.TagIds の集合に合わせて更新する。
	tags, err := u.syncDeckTags(ctx, deck.ID, ret.UserId, param.TagIds)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}
	deck.Tags = tags

	return deck, nil
}

func (u *Deck) Archive(
	ctx context.Context,
	id string,
) (*entity.Deck, error) {
	// 指定されたidのDeckが存在するか確認
	ret, err := u.repository.FindById(ctx, id)
	if err == apperror.ErrRecordNotFound {
		return nil, err
	} else if err != nil {
		logError(ctx, err)
		return nil, err
	}

	archivedAt := time.Now().Local()

	// アーカイブしたデッキは一覧の先頭にもデッキ選択にも現れなくなるため、
	// お気に入りも同時に解除する。使わなくなったデッキがお気に入りの枠を
	// 占有し続けるのを防ぐ。
	favoritedAt := time.Time{}

	deck := entity.NewDeck(
		id,
		ret.CreatedAt,
		archivedAt,
		favoritedAt,
		ret.UserId,
		ret.Name,
		ret.PrivateFlg,
		ret.LatestDeckCode,
		ret.PokemonSprites,
	)

	// アーカイブとお気に入りの解除は別テーブルへの更新になるため、
	// 片方だけが成功して食い違わないよう1つのトランザクションにまとめる。
	if err := u.transactionManager.Do(ctx, func(ctx context.Context) error {
		if err := u.repository.Save(ctx, deck); err != nil {
			logError(ctx, err)
			return err
		}

		return u.userFavoriteDeck.Delete(ctx, ret.UserId, id)
	}); err != nil {
		return nil, err
	}

	// アーカイブはタグを変更しないので、読み込んだ付与タグをそのままレスポンスに引き継ぐ。
	deck.Tags = ret.Tags

	return deck, nil
}

func (u *Deck) Unarchive(
	ctx context.Context,
	id string,
) (*entity.Deck, error) {
	// 指定されたidのDeckが存在するか確認
	ret, err := u.repository.FindById(ctx, id)
	if err == apperror.ErrRecordNotFound {
		return nil, err
	} else if err != nil {
		logError(ctx, err)
		return nil, err
	}

	archivedAt := time.Time{}

	// アーカイブ時にお気に入りは解除済み(ret.FavoritedAt はゼロ値)のため、
	// アーカイブ解除でお気に入りが復活することはない。
	deck := entity.NewDeck(
		id,
		ret.CreatedAt,
		archivedAt,
		ret.FavoritedAt,
		ret.UserId,
		ret.Name,
		ret.PrivateFlg,
		ret.LatestDeckCode,
		ret.PokemonSprites,
	)

	if err := u.repository.Save(ctx, deck); err != nil {
		logError(ctx, err)
		return nil, err
	}

	// アーカイブ解除もタグを変更しないので、付与タグをそのまま引き継ぐ。
	deck.Tags = ret.Tags

	return deck, nil
}

// Favorite は指定されたデッキをお気に入りにする。
//
// 上限(MaxFavoriteDecksPerUser)に達している場合は、お気に入りにした日時が
// 最も古いものから外して枠を空ける。上限が1の間はこれが「入れ替え」になり、
// ★を押せば必ず設定できる。上限を増やしてもこの挙動のまま破綻しない。
func (u *Deck) Favorite(
	ctx context.Context,
	id string,
) (*entity.Deck, error) {
	// 指定されたidのDeckが存在するか確認
	ret, err := u.repository.FindById(ctx, id)
	if err == apperror.ErrRecordNotFound {
		return nil, err
	} else if err != nil {
		logError(ctx, err)
		return nil, err
	}

	favoritedAt := time.Now().Local()

	// 件数の確認から追加までを1つのトランザクションで行う。
	// 分けると、同時に2回設定された際に上限を超えて登録され得る。
	if err := u.transactionManager.Do(ctx, func(ctx context.Context) error {
		favorites, err := u.userFavoriteDeck.FindByUserId(ctx, ret.UserId)
		if err != nil {
			logError(ctx, err)
			return err
		}

		for _, favorite := range favorites {
			// 既にお気に入りなら日時を変えない(★の二度押しで順番が変わらないように)
			if favorite.DeckId == id {
				favoritedAt = favorite.CreatedAt
				return nil
			}
		}

		// 1件追加すると上限を超える場合、超える分だけ古いものから外す。
		// favorites は古い順に並んでいる。
		excess := len(favorites) + 1 - MaxFavoriteDecksPerUser
		for i := 0; i < excess && i < len(favorites); i++ {
			if err := u.userFavoriteDeck.Delete(
				ctx, ret.UserId, favorites[i].DeckId,
			); err != nil {
				return err
			}
		}

		return u.userFavoriteDeck.Create(
			ctx,
			entity.NewUserFavoriteDeck(ret.UserId, id, favoritedAt),
		)
	}); err != nil {
		return nil, err
	}

	ret.FavoritedAt = favoritedAt

	return ret, nil
}

// Unfavorite は指定されたデッキのお気に入りを解除する。
func (u *Deck) Unfavorite(
	ctx context.Context,
	id string,
) (*entity.Deck, error) {
	// 指定されたidのDeckが存在するか確認
	ret, err := u.repository.FindById(ctx, id)
	if err == apperror.ErrRecordNotFound {
		return nil, err
	} else if err != nil {
		logError(ctx, err)
		return nil, err
	}

	if err := u.userFavoriteDeck.Delete(ctx, ret.UserId, id); err != nil {
		logError(ctx, err)
		return nil, err
	}

	ret.FavoritedAt = time.Time{}

	return ret, nil
}

func (u *Deck) Delete(
	ctx context.Context,
	id string,
) error {
	err := u.repository.Delete(ctx, id)

	if err != nil {
		logError(ctx, err)
		return err
	}

	return nil
}
