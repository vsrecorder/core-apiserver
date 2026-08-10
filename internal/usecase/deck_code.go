package usecase

import (
	"context"
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
}

type DeckCodeUpdateParam struct {
	PrivateFlg bool
	Memo       string
	TagIds     []string
}

func NewDeckCodeCreateParam(
	userId string,
	deckId string,
	code string,
	privateFlg bool,
	memo string,
	tagIds []string,
) *DeckCodeCreateParam {
	return &DeckCodeCreateParam{
		UserId:     userId,
		DeckId:     deckId,
		Code:       code,
		PrivateFlg: privateFlg,
		Memo:       memo,
		TagIds:     tagIds,
	}
}

func NewDeckCodeUpdateParam(
	privateFlg bool,
	memo string,
	tagIds []string,
) *DeckCodeUpdateParam {
	return &DeckCodeUpdateParam{
		PrivateFlg: privateFlg,
		Memo:       memo,
		TagIds:     tagIds,
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
	repository      repository.DeckCodeInterface
	deckAsset       repository.DeckAssetInterface
	tag             repository.TagInterface
	badgeEvaluation BadgeEvaluationInterface
}

func NewDeckCode(
	repository repository.DeckCodeInterface,
	deckAsset repository.DeckAssetInterface,
	tag repository.TagInterface,
	badgeEvaluation BadgeEvaluationInterface,
) DeckCodeInterface {
	return &DeckCode{repository, deckAsset, tag, badgeEvaluation}
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
		return nil, err
	}

	// FindAttachableByIds の戻り順は不定なので、付与順(tagIds)に整列してから採番する。
	orderedTags, attachableTagIds := orderAttachableTagsByIds(tags, tagIds)

	if err := u.tag.ReplaceDeckCodeTags(ctx, deckCodeId, attachableTagIds); err != nil {
		return nil, err
	}

	return orderedTags, nil
}

func (u *DeckCode) FindById(
	ctx context.Context,
	id string,
) (*entity.DeckCode, error) {
	deckcode, err := u.repository.FindById(ctx, id)

	if err != nil {
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
		return nil, err
	}

	return deckcodes, nil
}

func (u *DeckCode) Create(
	ctx context.Context,
	param *DeckCodeCreateParam,
) (*entity.DeckCode, error) {
	id, err := generateId()
	if err != nil {
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
			return nil, err
		}

		if err := u.deckAsset.UploadDeckImage(ctx, deckcode.Code); err != nil {
			return nil, err
		}
	}

	if err := u.repository.Save(ctx, deckcode); err != nil {
		return nil, err
	}

	// タグの付与はデッキコード本体とは別テーブルのため Save とは分けて反映する。
	tags, err := u.syncDeckCodeTags(ctx, deckcode.ID, param.UserId, param.TagIds)
	if err != nil {
		return nil, err
	}
	deckcode.Tags = tags

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

	if err := u.repository.Save(ctx, deckcode); err != nil {
		return nil, err
	}

	// タグの付与を param.TagIds の集合に合わせて更新する。
	tags, err := u.syncDeckCodeTags(ctx, deckcode.ID, ret.UserId, param.TagIds)
	if err != nil {
		return nil, err
	}
	deckcode.Tags = tags

	return deckcode, nil
}

func (u *DeckCode) Delete(
	ctx context.Context,
	id string,
) error {
	err := u.repository.Delete(ctx, id)

	if err != nil {
		return err
	}

	return nil
}
