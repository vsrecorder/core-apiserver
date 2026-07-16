package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterByCategory(t *testing.T) {
	findings := []finding{
		{table: "records", category: categoryLeak, userId: "uid_1", count: 3},
		{table: "notifications", category: categoryUnhandled, userId: "uid_1", count: 5},
		{table: "decks", category: categoryLeak, userId: "uid_2", count: 1},
		{table: "matches.opponents_user_id", category: categoryReference, userId: "uid_3", count: 2},
	}

	leaks := filterByCategory(findings, categoryLeak)
	assert.Equal(t, []finding{
		{table: "records", category: categoryLeak, userId: "uid_1", count: 3},
		{table: "decks", category: categoryLeak, userId: "uid_2", count: 1},
	}, leaks)

	unhandled := filterByCategory(findings, categoryUnhandled)
	assert.Len(t, unhandled, 1)
	assert.Equal(t, "notifications", unhandled[0].table)

	references := filterByCategory(findings, categoryReference)
	assert.Len(t, references, 1)
	assert.Equal(t, "matches.opponents_user_id", references[0].table)
}

func TestFilterByCategory_該当なしの場合(t *testing.T) {
	findings := []finding{
		{table: "notifications", category: categoryUnhandled, userId: "uid_1", count: 5},
	}

	// 削除漏れが1件も無い(= 退会処理としては正常)ケース
	assert.Empty(t, filterByCategory(findings, categoryLeak))
}

func TestSummarizeByTable(t *testing.T) {
	findings := []finding{
		{table: "records", userId: "uid_1", count: 3, note: "note_records"},
		{table: "records", userId: "uid_2", count: 2, note: "note_records"},
		{table: "decks", userId: "uid_1", count: 1, note: "note_decks"},
	}

	summaries := summarizeByTable(findings)

	// テーブル単位に合計され、検出結果に現れた順(= テーブル定義の順)を保つ
	assert.Equal(t, []tableSummary{
		{table: "records", count: 5, userCount: 2, note: "note_records"},
		{table: "decks", count: 1, userCount: 1, note: "note_decks"},
	}, summaries)
}

func TestSummarizeByTable_検出結果が空の場合(t *testing.T) {
	assert.Empty(t, summarizeByTable(nil))
}

// specs は退会処理(internal/usecase/user.go の User.Delete)と1対1で対応させる前提のため、
// 定義そのものの取り違え(分類漏れ・SQLの列数違い等)を検出する。
func TestSpecs(t *testing.T) {
	assert.NotEmpty(t, specs)

	names := make(map[string]bool)
	for _, spec := range specs {
		t.Run(spec.name, func(t *testing.T) {
			assert.False(t, names[spec.name], "テーブル定義が重複している")
			names[spec.name] = true

			assert.NotEmpty(t, spec.note, "検出時に原因の見当がつくよう note は必須")
			assert.Contains(t, spec.query, "u.deleted_at IS NOT NULL", "退会ユーザに絞り込んでいない")
			assert.Contains(t, spec.query, "GROUP BY", "退会ユーザごとの件数を返していない")
		})
	}

	// 退会処理が削除するテーブルが定義から漏れていないことを確認する
	for _, name := range []string{
		"records", "matches", "games", "decks", "deck_codes", "users_players", "unofficial_events",
	} {
		assert.True(t, names[name], "%s の定義が無い", name)
	}
}
