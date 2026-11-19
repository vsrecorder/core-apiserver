package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type DeckCodeCreateParam struct {
	UserId     string
	DeckId     string
	Code       string
	PrivateFlg bool
	Memo       string
	TagIds     []string
	// PokemonSprites はこのバージョンに設定するスプライト。nil と空スライスで意味が違う。
	// nil は「指定なし(直前のバージョンの設定を引き継ぐ)」、空スライスは「未設定で作る」。
	PokemonSprites []*PokemonSpriteParam
}

type DeckCodeUpdateParam struct {
	PrivateFlg bool
	Memo       string
	TagIds     []string
	// PokemonSprites は nil と空スライスで意味が違う。
	// nil は「スプライトに触れない(現在の設定を保つ)」、空スライスは「未設定に戻す」。
	PokemonSprites []*PokemonSpriteParam
}

func NewDeckCodeCreateParam(
	userId string,
	deckId string,
	code string,
	privateFlg bool,
	memo string,
	tagIds []string,
	pokemonSprites []*PokemonSpriteParam,
) *DeckCodeCreateParam {
	return &DeckCodeCreateParam{
		UserId:         userId,
		DeckId:         deckId,
		Code:           code,
		PrivateFlg:     privateFlg,
		Memo:           memo,
		TagIds:         tagIds,
		PokemonSprites: pokemonSprites,
	}
}

func NewDeckCodeUpdateParam(
	privateFlg bool,
	memo string,
	tagIds []string,
	pokemonSprites []*PokemonSpriteParam,
) *DeckCodeUpdateParam {
	return &DeckCodeUpdateParam{
		PrivateFlg:     privateFlg,
		Memo:           memo,
		TagIds:         tagIds,
		PokemonSprites: pokemonSprites,
	}
}

type DeckCodeInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.DeckCode, error)

	FindByDeckId(
		ctx context.Context,
		deckId string,
	) ([]*entity.DeckCode, error)

	Create(
		ctx context.Context,
		param *DeckCodeCreateParam,
	) (*entity.DeckCode, error)

	Update(
		ctx context.Context,
		id string,
		param *DeckCodeUpdateParam,
	) (*entity.DeckCode, error)

	Delete(
		ctx context.Context,
		id string,
	) error
}

type DeckCode struct {
	repository repository.DeckCodeInterface
	// deckRepository は作成時に deck_id が本人のデッキかを確かめるために使う(ownership.go)。
	deckRepository  repository.DeckInterface
	deckAsset       repository.DeckAssetInterface
	tag             repository.TagInterface
	badgeEvaluation BadgeEvaluationInterface
	// transactionManager はデッキコード本体(deck_codes)とタグの付与(deck_code_tags)・
	// スプライト(deck_code_pokemon_sprites)を1つのトランザクションにまとめるために使う。
	transactionManager repository.TransactionManager
	// deckCodePost はみんなの公開デッキへの投稿。削除したバージョンの投稿を連動して
	// 取り下げるために使う。nil なら連動しない(投稿機能を持たないテスト用)。
	deckCodePost repository.DeckCodePostInterface
}

func NewDeckCode(
	repository repository.DeckCodeInterface,
	deckRepository repository.DeckInterface,
	deckAsset repository.DeckAssetInterface,
	tag repository.TagInterface,
	badgeEvaluation BadgeEvaluationInterface,
	transactionManager repository.TransactionManager,
	deckCodePost repository.DeckCodePostInterface,
) DeckCodeInterface {
	return &DeckCode{repository, deckRepository, deckAsset, tag, badgeEvaluation, transactionManager, deckCodePost}
}

// syncDeckCodeTags は deckCodeId について、userId が付与できる有効なタグ(自分のタグ or
// プリセット)だけを残して deck_code_tags を更新し、付与後のタグを返す。
// 挙動は Deck.syncDeckTags と同じ。
func (u *DeckCode) syncDeckCodeTags(
	ctx context.Context,
	deckCodeId string,
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

	if err := u.tag.ReplaceDeckCodeTags(ctx, deckCodeId, attachableTagIds); err != nil {
		logError(ctx, err)
		return nil, err
	}

	return orderedTags, nil
}

// newDeckCodePokemonSprites は入力のスプライト指定をエンティティへ変換する。
// 戻り値は常に非 nil(空なら空スライス)にして、レスポンスへ載せる側が nil を気にしなくて済むようにする。
func newDeckCodePokemonSprites(params []*PokemonSpriteParam) []*entity.PokemonSprite {
	sprites := make([]*entity.PokemonSprite, 0, len(params))
	for _, param := range params {
		sprites = append(sprites, entity.NewPokemonSpriteWithPosition(param.ID, param.Position))
	}

	return sprites
}

// resolveCreatePokemonSprites は新しいバージョンに設定するスプライトを決める。
//
// 指定があれば(空スライスを含めて)それをそのまま使う。nil(項目ごと省略)のときは、同じデッキの
// 直前のバージョン(作成日時が最も新しいもの)の設定を引き継ぐ。同じデッキ名のまま相棒を入れ替えた
// バージョンの次も、たいていは同じ相棒のままだからで、画面は元にしたバージョンの設定を初期値に
// して必ず送ってくる。ここで補うのは、この項目を知らない古いクライアント(デプロイ直後に残る
// 古い画面)から作られたバージョンだけが未設定に戻り、記録のスプライトがデッキのものへ戻るのを防ぐため。
//
// versions は同じデッキの既存バージョン(新しい順)。
// 直前のバージョンが無い(最初のバージョン)・未設定のときは空を返す。戻り値は常に非 nil。
func resolveCreatePokemonSprites(
	param *DeckCodeCreateParam,
	versions []*entity.DeckCode,
) []*entity.PokemonSprite {
	if param.PokemonSprites != nil {
		return newDeckCodePokemonSprites(param.PokemonSprites)
	}

	if len(versions) == 0 {
		return []*entity.PokemonSprite{}
	}

	return copyPokemonSprites(versions[0].PokemonSprites)
}

func (u *DeckCode) FindById(
	ctx context.Context,
	id string,
) (*entity.DeckCode, error) {
	deckcode, err := u.repository.FindById(ctx, id)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return deckcode, nil
}

func (u *DeckCode) FindByDeckId(
	ctx context.Context,
	deckId string,
) ([]*entity.DeckCode, error) {
	deckcodes, err := u.repository.FindByDeckId(ctx, deckId)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return deckcodes, nil
}

func (u *DeckCode) Create(
	ctx context.Context,
	param *DeckCodeCreateParam,
) (*entity.DeckCode, error) {
	// 他人のデッキにデッキコードを足せると、そのデッキの公開一覧の「最新デッキコード」を
	// 差し替えられてしまう。外部サイトへの取得を始める前に弾く。
	if err := verifyDeckOwnership(ctx, u.deckRepository, param.UserId, param.DeckId); err != nil {
		logError(ctx, err)
		return nil, err
	}

	id, err := generateId()
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	createdAt := time.Now().Local()

	deckcode := entity.NewDeckCode(
		id,
		createdAt,
		param.UserId,
		param.DeckId,
		param.Code,
		param.PrivateFlg,
		param.Memo,
	)

	if deckcode.Code != "" {
		// 先にデッキコードのHTMLページをアップロードする。
		// デッキ画像のアップロードでは、そのデッキコードが存在するか確認することができないが、
		// デッキコードのHTMLページのアップロードでは、デッキコードが正しいかどうかを確認することができるため、
		// 先にデッキコードのHTMLページをアップロードすることで、デッキコードが正しいかどうかを確認することができる。
		// アップロードに失敗した場合はデッキ作成を中止する。
		if err := u.deckAsset.UploadDeckResultHTML(ctx, deckcode.Code); err != nil {
			logError(ctx, err)
			return nil, err
		}

		if err := u.deckAsset.UploadDeckImage(ctx, deckcode.Code); err != nil {
			logError(ctx, err)
			return nil, err
		}
	}

	// デッキコード本体とタグの付与は別テーブルだが、片方だけ成功して食い違わないよう
	// 1つのトランザクションにまとめる。ここが失敗したときだけ deck_codes に行が残らず、
	// 呼び出し側はそのまま再試行できる(記録・対戦結果・デッキの作成と同じ方針)。
	if err := u.transactionManager.Do(ctx, func(ctx context.Context) error {
		// 既存のバージョンは本体の保存より前に引く(保存後だと自分自身が最新になる)。
		// スプライトの引き継ぎ元(直前のバージョン)と、デッキのスプライトを揃えるときに使う。
		// 「未設定で作る」(空スライス)ならどちらも要らないので引かない。
		var versions []*entity.DeckCode
		if param.PokemonSprites == nil || len(param.PokemonSprites) > 0 {
			var err error
			if versions, err = u.repository.FindByDeckId(ctx, param.DeckId); err != nil {
				logError(ctx, err)
				return err
			}
		}
		pokemonSprites := resolveCreatePokemonSprites(param, versions)

		if err := u.repository.Save(ctx, deckcode); err != nil {
			logError(ctx, err)
			return err
		}

		// タグの付与はデッキコード本体とは別テーブルのため Save とは分けて反映する。
		tags, err := u.syncDeckCodeTags(ctx, deckcode.ID, param.UserId, param.TagIds)
		if err != nil {
			logError(ctx, err)
			return err
		}
		deckcode.Tags = tags

		// スプライトも別テーブルのため Save とは分けて反映する。空なら行を作らない(未設定)。
		if len(pokemonSprites) > 0 {
			if err := u.repository.ReplacePokemonSprites(ctx, deckcode.ID, pokemonSprites); err != nil {
				logError(ctx, err)
				return err
			}
		}
		deckcode.PokemonSprites = pokemonSprites

		// 作ったバージョンが最新になるので、デッキのスプライトもこれに揃える。
		if err := syncDeckSpritesWithLatestVersion(ctx, u.deckRepository, u.repository, deckcode.DeckId, versions, pokemonSprites); err != nil {
			logError(ctx, err)
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	if deckcode.Code != "" {
		u.badgeEvaluation.EvaluateOnDeckCodeCreated(ctx, param.UserId, deckcode)
	}

	return deckcode, nil
}

func (u *DeckCode) Update(
	ctx context.Context,
	id string,
	param *DeckCodeUpdateParam,
) (*entity.DeckCode, error) {
	// 指定されたidのDeckCodeが存在するか確認
	ret, err := u.repository.FindById(ctx, id)
	if err == apperror.ErrRecordNotFound {
		return nil, err
	} else if err != nil {
		logError(ctx, err)
		return nil, err
	}

	deckcode := entity.NewDeckCode(
		id,
		ret.CreatedAt,
		ret.UserId,
		ret.DeckId,
		ret.Code,
		param.PrivateFlg,
		param.Memo,
	)

	// スプライトを設定するときは、最新バージョンかどうかを確かめるために同じデッキのバージョンを引く
	// (最新バージョンならデッキのスプライトも揃える)。触れない・未設定に戻すときは要らない。
	var otherVersions []*entity.DeckCode
	isLatest := false
	if len(param.PokemonSprites) > 0 {
		versions, err := u.repository.FindByDeckId(ctx, ret.DeckId)
		if err != nil {
			logError(ctx, err)
			return nil, err
		}

		isLatest = len(versions) > 0 && versions[0].ID == id
		for _, version := range versions {
			if version.ID != id {
				otherVersions = append(otherVersions, version)
			}
		}
	}

	// 作成時と同じく、本体とタグの付与は1つのトランザクションにまとめる。
	if err := u.transactionManager.Do(ctx, func(ctx context.Context) error {
		if err := u.repository.Save(ctx, deckcode); err != nil {
			logError(ctx, err)
			return err
		}

		// タグの付与を param.TagIds の集合に合わせて更新する。
		tags, err := u.syncDeckCodeTags(ctx, deckcode.ID, ret.UserId, param.TagIds)
		if err != nil {
			logError(ctx, err)
			return err
		}
		deckcode.Tags = tags

		// スプライトは指定があったときだけ置き換える。nil(項目ごと省略)は現在の設定を保つ。
		// この項目を知らない古いクライアントの更新で、設定済みのスプライトを消さないため。
		if param.PokemonSprites == nil {
			deckcode.PokemonSprites = ret.PokemonSprites
			return nil
		}

		pokemonSprites := newDeckCodePokemonSprites(param.PokemonSprites)
		if err := u.repository.ReplacePokemonSprites(ctx, deckcode.ID, pokemonSprites); err != nil {
			logError(ctx, err)
			return err
		}
		deckcode.PokemonSprites = pokemonSprites

		// 最新バージョンのスプライトを変えたら、デッキのスプライトもこれに揃える。
		if isLatest {
			if err := syncDeckSpritesWithLatestVersion(ctx, u.deckRepository, u.repository, ret.DeckId, otherVersions, pokemonSprites); err != nil {
				logError(ctx, err)
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return deckcode, nil
}

func (u *DeckCode) Delete(
	ctx context.Context,
	id string,
) error {
	// 最新バージョンを消すと、ひとつ前のバージョンが最新に繰り上がる。繰り上がるバージョンに
	// スプライトの設定があれば、デッキのスプライトもそれに揃える(未設定ならデッキのまま)。
	// 繰り上がりの判定は削除の前に済ませる(削除はトランザクションの中で、外からはまだ見えない)。
	target, err := u.repository.FindById(ctx, id)
	if errors.Is(err, apperror.ErrRecordNotFound) {
		return err
	} else if err != nil {
		logError(ctx, err)
		return err
	}

	versions, err := u.repository.FindByDeckId(ctx, target.DeckId)
	if err != nil {
		logError(ctx, err)
		return err
	}

	var promoted *entity.DeckCode
	var otherVersions []*entity.DeckCode
	if len(versions) > 1 && versions[0].ID == id {
		promoted = versions[1]
		otherVersions = versions[2:]
	}

	// 削除したバージョンの投稿(みんなの公開デッキ)も同じトランザクションで取り下げる。
	return u.transactionManager.Do(ctx, func(ctx context.Context) error {
		if err := u.deckCodePost.UnpublishByDeckCodeId(ctx, id, time.Now().Local()); err != nil {
			logError(ctx, err)
			return err
		}

		if err := u.repository.Delete(ctx, id); err != nil {
			logError(ctx, err)
			return err
		}

		if promoted != nil {
			if err := syncDeckSpritesWithLatestVersion(ctx, u.deckRepository, u.repository, target.DeckId, otherVersions, promoted.PokemonSprites); err != nil {
				logError(ctx, err)
				return err
			}
		}

		return nil
	})
}
