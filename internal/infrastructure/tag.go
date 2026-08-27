package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type Tag struct {
	db *gorm.DB
}

func NewTag(
	db *gorm.DB,
) repository.TagInterface {
	return &Tag{db}
}

// tagLinkTable は「エンティティ ⇔ タグ」中間テーブルのメタ情報。
// 新しい付与先へ広げるときは、この値を1つ足して Replace メソッドを薄く生やすだけでよい。
// name / ownerColumn はコード内定数のみを渡す前提で、SQLへ直接埋め込む。
type tagLinkTable struct {
	name        string // 例: "deck_tags"
	ownerColumn string // 例: "deck_id"
}

var (
	deckTagLink     = tagLinkTable{name: "deck_tags", ownerColumn: "deck_id"}
	deckCodeTagLink = tagLinkTable{name: "deck_code_tags", ownerColumn: "deck_code_id"}
	matchTagLink    = tagLinkTable{name: "match_tags", ownerColumn: "match_id"}
	recordTagLink   = tagLinkTable{name: "record_tags", ownerColumn: "record_id"}
)

func (i *Tag) FindByUserId(
	ctx context.Context,
	uid string,
) ([]*entity.Tag, error) {
	var tagModels []*model.Tag

	if tx := dbFromContext(ctx, i.db).
		Where("user_id = ?", uid).
		Order("created_at DESC").
		Find(&tagModels); tx.Error != nil {
		return nil, tx.Error
	}

	ret := make([]*entity.Tag, 0, len(tagModels))
	for _, m := range tagModels {
		ret = append(ret, entity.NewTag(m.ID, m.CreatedAt, m.UpdatedAt, m.UserId, m.Name, m.Color, m.PresetFlg, m.PresetCategory, m.TextColor))
	}

	return ret, nil
}

func (i *Tag) FindPresets(
	ctx context.Context,
	category string,
) ([]*entity.Tag, error) {
	var tagModels []*model.Tag

	tx := dbFromContext(ctx, i.db).Where("preset_flg = ?", true)
	// category 指定があればその群だけに絞る(空文字は全プリセット)。
	if category != "" {
		tx = tx.Where("preset_category = ?", category)
	}

	// プリセット(ACE SPEC)は card id 昇順で作成し、ULIDは生成順に単調増加するため、
	// id 昇順で引くと card id 昇順(≒収録順)で並ぶ。
	// 大会順位のプリセットも同じ規約で、優勝から順に並ぶIDを振ってある。
	if tx := tx.Order("id ASC").Find(&tagModels); tx.Error != nil {
		return nil, tx.Error
	}

	ret := make([]*entity.Tag, 0, len(tagModels))
	for _, m := range tagModels {
		ret = append(ret, entity.NewTag(m.ID, m.CreatedAt, m.UpdatedAt, m.UserId, m.Name, m.Color, m.PresetFlg, m.PresetCategory, m.TextColor))
	}

	return ret, nil
}

func (i *Tag) FindById(
	ctx context.Context,
	id string,
) (*entity.Tag, error) {
	var m *model.Tag

	if tx := dbFromContext(ctx, i.db).Where("id = ?", id).First(&m); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, wrapError(tx.Error)
	}

	return entity.NewTag(m.ID, m.CreatedAt, m.UpdatedAt, m.UserId, m.Name, m.Color, m.PresetFlg, m.PresetCategory, m.TextColor), nil
}

func (i *Tag) FindAttachableByIds(
	ctx context.Context,
	ids []string,
	uid string,
) ([]*entity.Tag, error) {
	if len(ids) == 0 {
		return []*entity.Tag{}, nil
	}

	var tagModels []*model.Tag

	// 付与できるのは「自分のタグ」か「プリセットタグ」。
	if tx := dbFromContext(ctx, i.db).
		Where("id IN ? AND (user_id = ? OR preset_flg = ?)", ids, uid, true).
		Find(&tagModels); tx.Error != nil {
		return nil, tx.Error
	}

	ret := make([]*entity.Tag, 0, len(tagModels))
	for _, m := range tagModels {
		ret = append(ret, entity.NewTag(m.ID, m.CreatedAt, m.UpdatedAt, m.UserId, m.Name, m.Color, m.PresetFlg, m.PresetCategory, m.TextColor))
	}

	return ret, nil
}

func (i *Tag) FindByUserIdAndName(
	ctx context.Context,
	uid string,
	name string,
) (*entity.Tag, error) {
	var m *model.Tag

	if tx := dbFromContext(ctx, i.db).
		Where("user_id = ? AND name = ?", uid, name).
		First(&m); tx.Error != nil {
		return nil, wrapError(tx.Error)
	}

	return entity.NewTag(m.ID, m.CreatedAt, m.UpdatedAt, m.UserId, m.Name, m.Color, m.PresetFlg, m.PresetCategory, m.TextColor), nil
}

func (i *Tag) Save(
	ctx context.Context,
	e *entity.Tag,
) error {
	tag := model.NewTag(e.ID, e.CreatedAt, e.UpdatedAt, e.UserId, e.Name, e.Color, e.PresetFlg, e.PresetCategory, e.TextColor)

	return dbFromContext(ctx, i.db).Save(tag).Error
}

func (i *Tag) Delete(
	ctx context.Context,
	id string,
) error {
	// タグ本体は論理削除、中間テーブルの行は物理削除する。
	// 付与先(デッキ/デッキコード/記録/対戦結果)が削除済みタグを参照し続けないよう、まとめて行う。
	return dbFromContext(ctx, i.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tag_id = ?", id).Delete(&model.DeckTag{}).Error; err != nil {
			logError(ctx, err)
			return err
		}

		if err := tx.Where("tag_id = ?", id).Delete(&model.DeckCodeTag{}).Error; err != nil {
			logError(ctx, err)
			return err
		}

		if err := tx.Where("tag_id = ?", id).Delete(&model.MatchTag{}).Error; err != nil {
			logError(ctx, err)
			return err
		}

		if err := tx.Where("tag_id = ?", id).Delete(&model.RecordTag{}).Error; err != nil {
			logError(ctx, err)
			return err
		}

		return tx.Where("id = ?", id).Delete(&model.Tag{}).Error
	})
}

// DeleteByUserId は退会時に、そのユーザのタグをまとめて論理削除し、
// 併せてそのタグの中間テーブルの行を物理削除する(Delete の一括版)。
func (i *Tag) DeleteByUserId(
	ctx context.Context,
	uid string,
) error {
	// プリセットは user_id='' で全ユーザー共通のため、空のIDで消しにいかせない。
	// 呼び出し側の取り違えでプリセットが全消しになるのを防ぐ。
	if uid == "" {
		return nil
	}

	return dbFromContext(ctx, i.db).Transaction(func(tx *gorm.DB) error {
		// 中間テーブルを先に消す。tags を論理削除した後だと、GORM が付ける
		// deleted_at IS NULL でサブクエリが0件になり、中間テーブルの行が消し残る。
		//
		// サブクエリは毎回 tx.Model(...) から組み立てる。GORM の *gorm.DB は
		// 条件を積んでいくため、同じインスタンスを使い回すと条件が混ざる。
		tagIds := func() *gorm.DB {
			return tx.Model(&model.Tag{}).Select("id").Where("user_id = ?", uid)
		}

		for _, link := range []any{
			&model.DeckTag{},
			&model.DeckCodeTag{},
			&model.MatchTag{},
			&model.RecordTag{},
		} {
			if err := tx.Where("tag_id IN (?)", tagIds()).Delete(link).Error; err != nil {
				logError(ctx, err)
				return err
			}
		}

		if err := tx.Where("user_id = ?", uid).Delete(&model.Tag{}).Error; err != nil {
			logError(ctx, err)
			return err
		}

		return nil
	})
}

func (i *Tag) ReplaceDeckTags(
	ctx context.Context,
	deckId string,
	tagIds []string,
) error {
	return i.replaceTags(ctx, deckTagLink, deckId, tagIds)
}

func (i *Tag) ReplaceDeckCodeTags(
	ctx context.Context,
	deckCodeId string,
	tagIds []string,
) error {
	return i.replaceTags(ctx, deckCodeTagLink, deckCodeId, tagIds)
}

func (i *Tag) ReplaceMatchTags(
	ctx context.Context,
	matchId string,
	tagIds []string,
) error {
	return i.replaceTags(ctx, matchTagLink, matchId, tagIds)
}

func (i *Tag) ReplaceRecordTags(
	ctx context.Context,
	recordId string,
	tagIds []string,
) error {
	return i.replaceTags(ctx, recordTagLink, recordId, tagIds)
}

// replaceTags は ownerId が持つ中間テーブルの行を tagIds の集合に一致させる。
// 「全削除 → 再INSERT」で表す(件数が小さくデッキごとのタグ付与に十分)。
// link.name / link.ownerColumn はコード内定数のみのため SQL へ直接埋め込む。
func (i *Tag) replaceTags(
	ctx context.Context,
	link tagLinkTable,
	ownerId string,
	tagIds []string,
) error {
	return dbFromContext(ctx, i.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			"DELETE FROM "+link.name+" WHERE "+link.ownerColumn+" = ?", ownerId,
		).Error; err != nil {
			return err
		}

		// 重複IDが来ても複合主キー違反にならないよう重複を除く。
		// position は付与した順(tagIds の並び)を1始まりで保持し、読み出し時の表示順に使う。
		seen := make(map[string]struct{}, len(tagIds))
		position := 1
		for _, tagId := range tagIds {
			if tagId == "" {
				continue
			}
			if _, ok := seen[tagId]; ok {
				continue
			}
			seen[tagId] = struct{}{}

			if err := tx.Exec(
				"INSERT INTO "+link.name+" ("+link.ownerColumn+", tag_id, position) VALUES (?, ?, ?)", ownerId, tagId, position,
			).Error; err != nil {
				return err
			}
			position++
		}

		return nil
	})
}

// tagJoinOwner はタグと中間テーブルを JOIN した結果を受けるための構造体。
type tagJoinOwner struct {
	OwnerId        string
	ID             string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	UserId         string
	Name           string
	Color          string
	PresetFlg      bool
	PresetCategory string
	TextColor      string
}

// findTagsByOwnerIds は中間テーブル(link)経由で、複数の付与先のタグを1クエリでまとめて取得し、
// owner_id ごとに束ねて返す。一覧処理での N+1 を避けるため、付与先を複数扱う箇所では必ずこれを使う。
// 論理削除済みのタグ(tags.deleted_at IS NOT NULL)は除外する。
func findTagsByOwnerIds(
	ctx context.Context,
	db *gorm.DB,
	link tagLinkTable,
	ownerIds []string,
) (map[string][]*entity.Tag, error) {
	if len(ownerIds) == 0 {
		return map[string][]*entity.Tag{}, nil
	}

	var rows []*tagJoinOwner
	if tx := db.WithContext(ctx).
		Table(link.name).
		Select(
			link.name+"."+link.ownerColumn+" AS owner_id, "+
				"tags.id AS id, tags.created_at AS created_at, tags.updated_at AS updated_at, "+
				"tags.user_id AS user_id, tags.name AS name, tags.color AS color, "+
				"tags.preset_flg AS preset_flg, tags.preset_category AS preset_category, "+
				"tags.text_color AS text_color",
		).
		Joins("JOIN tags ON tags.id = "+link.name+".tag_id AND tags.deleted_at IS NULL").
		Where(link.name+"."+link.ownerColumn+" IN ?", ownerIds).
		// 付与した順(position 昇順)で表示する。position が同値(移行前の既存行など)は
		// タグの作成日時降順で安定させる。
		Order(link.name + ".position ASC, tags.created_at DESC").
		Scan(&rows); tx.Error != nil {
		return nil, tx.Error
	}

	ret := make(map[string][]*entity.Tag, len(ownerIds))
	for _, r := range rows {
		ret[r.OwnerId] = append(
			ret[r.OwnerId],
			entity.NewTag(r.ID, r.CreatedAt, r.UpdatedAt, r.UserId, r.Name, r.Color, r.PresetFlg, r.PresetCategory, r.TextColor),
		)
	}

	return ret, nil
}

// findTagsByDeckIds / findTagsByDeckCodeIds は付与先ごとの薄いラッパ。
func findTagsByDeckIds(
	ctx context.Context,
	db *gorm.DB,
	deckIds []string,
) (map[string][]*entity.Tag, error) {
	return findTagsByOwnerIds(ctx, db, deckTagLink, deckIds)
}

func findTagsByDeckCodeIds(
	ctx context.Context,
	db *gorm.DB,
	deckCodeIds []string,
) (map[string][]*entity.Tag, error) {
	return findTagsByOwnerIds(ctx, db, deckCodeTagLink, deckCodeIds)
}

func findTagsByMatchIds(
	ctx context.Context,
	db *gorm.DB,
	matchIds []string,
) (map[string][]*entity.Tag, error) {
	return findTagsByOwnerIds(ctx, db, matchTagLink, matchIds)
}

func findTagsByRecordIds(
	ctx context.Context,
	db *gorm.DB,
	recordIds []string,
) (map[string][]*entity.Tag, error) {
	return findTagsByOwnerIds(ctx, db, recordTagLink, recordIds)
}
