package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type TagCreateParam struct {
	UserId string
	Name   string
	Color  string
}

func NewTagCreateParam(
	userId string,
	name string,
	color string,
) *TagCreateParam {
	return &TagCreateParam{
		UserId: userId,
		Name:   name,
		Color:  color,
	}
}

type TagUpdateParam struct {
	Name  string
	Color string
}

func NewTagUpdateParam(
	name string,
	color string,
) *TagUpdateParam {
	return &TagUpdateParam{
		Name:  name,
		Color: color,
	}
}

type TagInterface interface {
	FindByUserId(
		ctx context.Context,
		uid string,
	) ([]*entity.Tag, error)

	FindPresets(
		ctx context.Context,
	) ([]*entity.Tag, error)

	Create(
		ctx context.Context,
		param *TagCreateParam,
	) (*entity.Tag, error)

	Update(
		ctx context.Context,
		id string,
		param *TagUpdateParam,
	) (*entity.Tag, error)

	Delete(
		ctx context.Context,
		id string,
	) error
}

type Tag struct {
	repository repository.TagInterface
}

func NewTag(
	repository repository.TagInterface,
) TagInterface {
	return &Tag{repository}
}

func (u *Tag) FindByUserId(
	ctx context.Context,
	uid string,
) ([]*entity.Tag, error) {
	return u.repository.FindByUserId(ctx, uid)
}

func (u *Tag) FindPresets(
	ctx context.Context,
) ([]*entity.Tag, error) {
	return u.repository.FindPresets(ctx)
}

// Create はタグを作成する。同名のタグが既にある場合は作らず既存のものを返す
// (find-or-create)。タグ選択UIで「入力した名前のタグを作る」操作が、
// 既存タグの重複作成エラーにならないようにするため。
func (u *Tag) Create(
	ctx context.Context,
	param *TagCreateParam,
) (*entity.Tag, error) {
	existing, err := u.repository.FindByUserIdAndName(ctx, param.UserId, param.Name)
	if err != nil && !errors.Is(err, apperror.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	id, err := generateId()
	if err != nil {
		return nil, err
	}

	now := time.Now().Local()
	// APIから作られるのは常にユーザー個別タグ(preset_flg=false)。プリセットは backfill が作る。
	tag := entity.NewTag(id, now, now, param.UserId, param.Name, param.Color, false)

	if err := u.repository.Save(ctx, tag); err != nil {
		return nil, err
	}

	return tag, nil
}

// Update はタグ名・色を変更する。所有者チェックは controller の authorization で担保する。
// 別のタグと同名になる場合は ErrAlreadyExists を返す。
func (u *Tag) Update(
	ctx context.Context,
	id string,
	param *TagUpdateParam,
) (*entity.Tag, error) {
	ret, err := u.repository.FindById(ctx, id)
	if err != nil {
		return nil, err
	}

	// リネームで既存タグと衝突しないか確認する(自分自身への変更は許可)。
	if param.Name != ret.Name {
		conflict, err := u.repository.FindByUserIdAndName(ctx, ret.UserId, param.Name)
		if err != nil && !errors.Is(err, apperror.ErrRecordNotFound) {
			return nil, err
		}
		if conflict != nil && conflict.ID != id {
			return nil, apperror.ErrAlreadyExists
		}
	}

	tag := entity.NewTag(id, ret.CreatedAt, time.Now().Local(), ret.UserId, param.Name, param.Color, ret.PresetFlg)

	if err := u.repository.Save(ctx, tag); err != nil {
		return nil, err
	}

	return tag, nil
}

func (u *Tag) Delete(
	ctx context.Context,
	id string,
) error {
	return u.repository.Delete(ctx, id)
}

// orderAttachableTagsByIds は、FindAttachableByIds が返す(DBの都合で並び順が不定の)
// タグ群を、元の付与順(tagIds の並び)に整列し直す。付与不可なID(他人のタグ・存在しない)や
// 重複は取り除く。これにより ReplaceXxxTags への入力が付与順になり、
// 「最初に付与したタグ=position 1」、それ以降も付与順で position が採番される。
func orderAttachableTagsByIds(
	tags []*entity.Tag,
	tagIds []string,
) ([]*entity.Tag, []string) {
	tagById := make(map[string]*entity.Tag, len(tags))
	for _, tag := range tags {
		tagById[tag.ID] = tag
	}

	orderedTags := make([]*entity.Tag, 0, len(tags))
	orderedIds := make([]string, 0, len(tags))
	for _, id := range tagIds {
		tag, ok := tagById[id]
		if !ok {
			continue // 付与不可、または既に採用済みの重複IDは飛ばす
		}
		delete(tagById, id) // 同一IDの重複は最初の1回だけにする
		orderedTags = append(orderedTags, tag)
		orderedIds = append(orderedIds, id)
	}

	return orderedTags, orderedIds
}
