package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// orderAttachableTagsByIds は FindAttachableByIds の(並び順が不定の)戻り値を
// 付与順(tagIds の並び)へ整列し直す。position 採番の入力になるため、
// 「最初に付与したタグ=position 1」以降も付与順になることを保証する要となる。
func TestOrderAttachableTagsByIds(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newTag := func(id string) *entity.Tag {
		return entity.NewTag(id, now, now, "u1", id, "", false)
	}

	t.Run("正常系_付与順(tagIds)の並びに整列し直す", func(t *testing.T) {
		// FindAttachableByIds はわざと付与順と異なる順で返す
		found := []*entity.Tag{newTag("c"), newTag("a"), newTag("b")}
		tagIds := []string{"a", "b", "c"}

		tags, ids := orderAttachableTagsByIds(found, tagIds)

		require.Equal(t, []string{"a", "b", "c"}, ids)
		require.Len(t, tags, 3)
		require.Equal(t, "a", tags[0].ID)
		require.Equal(t, "b", tags[1].ID)
		require.Equal(t, "c", tags[2].ID)
	})

	t.Run("正常系_付与不可なIDは取り除かれる", func(t *testing.T) {
		// tagIds には付与不可(x)も含むが、found には無い
		found := []*entity.Tag{newTag("b"), newTag("a")}
		tagIds := []string{"a", "x", "b"}

		tags, ids := orderAttachableTagsByIds(found, tagIds)

		require.Equal(t, []string{"a", "b"}, ids)
		require.Len(t, tags, 2)
		require.Equal(t, "a", tags[0].ID)
		require.Equal(t, "b", tags[1].ID)
	})

	t.Run("正常系_重複IDは最初の1回だけ採用される", func(t *testing.T) {
		found := []*entity.Tag{newTag("a"), newTag("b")}
		tagIds := []string{"a", "b", "a"}

		tags, ids := orderAttachableTagsByIds(found, tagIds)

		require.Equal(t, []string{"a", "b"}, ids)
		require.Len(t, tags, 2)
	})

	t.Run("正常系_空入力は空を返す", func(t *testing.T) {
		tags, ids := orderAttachableTagsByIds(nil, nil)
		require.Empty(t, tags)
		require.Empty(t, ids)
	})
}
