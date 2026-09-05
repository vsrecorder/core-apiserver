package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 個別ページの URL 組み立て(webapp のルーティングと揃える)を検証する。
func TestDeckCodePostPageURL(t *testing.T) {
	t.Run("正常系_ベースURLと投稿IDを繋ぐ", func(t *testing.T) {
		require.Equal(
			t,
			"https://vsrecorder.mobi/shared_decks/01HD7Y3K8D6FDHMHTZ2GT41TP1",
			deckCodePostPageURL("https://vsrecorder.mobi", "01HD7Y3K8D6FDHMHTZ2GT41TP1"),
		)
	})

	t.Run("正常系_末尾のスラッシュは重ねない", func(t *testing.T) {
		require.Equal(
			t,
			"http://localhost:3000/shared_decks/post-1",
			deckCodePostPageURL("http://localhost:3000/", "post-1"),
		)
	})
}
