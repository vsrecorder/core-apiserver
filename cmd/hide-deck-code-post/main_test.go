package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// planActions の振り分け(非表示なら表示中のものだけ、解除なら非表示のものだけを変更する)を検証する。
func TestPlanActions(t *testing.T) {
	now := time.Now()
	visible := target{ID: "visible"}
	hidden := target{ID: "hidden", HiddenAt: &now}

	t.Run("正常系_非表示は表示中の投稿だけを対象にし非表示済みはスキップする", func(t *testing.T) {
		apply, skip := planActions([]target{visible, hidden}, false)

		require.Equal(t, []target{visible}, apply)
		require.Equal(t, []target{hidden}, skip)
	})

	t.Run("正常系_解除は非表示の投稿だけを対象にし表示中はスキップする", func(t *testing.T) {
		apply, skip := planActions([]target{visible, hidden}, true)

		require.Equal(t, []target{hidden}, apply)
		require.Equal(t, []target{visible}, skip)
	})

	t.Run("正常系_候補が無ければ空のまま返す", func(t *testing.T) {
		apply, skip := planActions([]target{}, false)

		require.Empty(t, apply)
		require.Empty(t, skip)
	})
}
