package main

import (
	"strings"
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

func TestDisplayName(t *testing.T) {
	names := map[string]string{
		"uid_named":   "たいち",
		"uid_no_name": "", // users.name は任意項目のため、DB上 NULL だと空文字になる
	}

	assert.Equal(t, "たいち", displayName(names, "uid_named"))
	assert.Equal(t, "名前未設定", displayName(names, "uid_no_name"))
	// 退会ユーザの一覧に無いIDは通常ありえないが、空欄で表示されないことを保証する
	assert.Equal(t, "名前未設定", displayName(names, "uid_unknown"))
}

func TestFilterByTables(t *testing.T) {
	findings := []finding{
		{table: "records", userId: "uid_1", count: 3},
		{table: "notifications", userId: "uid_1", count: 5},
		{table: "decks", userId: "uid_2", count: 1},
	}

	targets := []tableSpec{{name: "records"}, {name: "decks"}}

	ret := filterByTables(findings, targets)

	assert.Equal(t, []finding{
		{table: "records", userId: "uid_1", count: 3},
		{table: "decks", userId: "uid_2", count: 1},
	}, ret)
}

func TestDeleteTargets(t *testing.T) {
	specs := []tableSpec{
		{name: "records", category: categoryLeak, deleteQuery: "UPDATE ..."},
		{name: "notifications", category: categoryUnhandled, deleteQuery: "DELETE ..."},
		// 他のユーザのデータからの参照。削除するSQLを持たない
		{name: "matches.opponents_user_id", category: categoryReference},
	}

	t.Run("正常系_既定では削除漏れのみを対象にする", func(t *testing.T) {
		targets := deleteTargets(specs, false)

		assert.Len(t, targets, 1)
		assert.Equal(t, "records", targets[0].name)
	})

	t.Run("正常系_includeUnhandledなら未対応も対象にする", func(t *testing.T) {
		targets := deleteTargets(specs, true)

		assert.Len(t, targets, 2)
		assert.Equal(t, "records", targets[0].name)
		assert.Equal(t, "notifications", targets[1].name)
	})

	t.Run("正常系_参照はどちらの場合も対象にしない", func(t *testing.T) {
		for _, includeUnhandled := range []bool{false, true} {
			for _, target := range deleteTargets(specs, includeUnhandled) {
				assert.NotEqual(t, categoryReference, target.category, "他人のデータを削除対象にしてはいけない")
			}
		}
	})
}

// specs は退会処理(internal/usecase/user.go の User.Delete)と1対1で対応させる前提のため、
// 定義そのものの取り違え(分類漏れ・SQLの列数違い等)を検出する。
func TestSpecs(t *testing.T) {
	assert.NotEmpty(t, specs)

	names := make(map[string]bool)
	for _, spec := range specs {
		t.Run("正常系_"+spec.name, func(t *testing.T) {
			assert.False(t, names[spec.name], "テーブル定義が重複している")
			names[spec.name] = true

			assert.NotEmpty(t, spec.note, "検出時に原因の見当がつくよう note は必須")
			assert.Contains(t, spec.query, "u.deleted_at IS NOT NULL", "退会ユーザに絞り込んでいない")
			assert.Contains(t, spec.query, "GROUP BY", "退会ユーザごとの件数を返していない")

			if spec.category == categoryReference {
				// 他のユーザが作成したデータのため、削除できてしまってはいけない
				assert.Empty(t, spec.deleteQuery, "参照に削除するSQLを持たせてはいけない")
				return
			}

			assert.NotEmpty(t, spec.deleteQuery, "削除対象なのに削除するSQLが無い")
			assert.Contains(t, spec.deleteQuery, "u.deleted_at IS NOT NULL", "退会ユーザに絞り込んでいない")
			// -user 指定時に deleteQuery へ AND で条件を足すため、WHERE と ownerColumn が要る
			assert.Contains(t, spec.deleteQuery, "WHERE", "AND で条件を足せる形になっていない")
			assert.NotEmpty(t, spec.ownerColumn, "-user で絞り込めない")

			// 論理削除を持つテーブルを物理削除してしまわないこと(退会処理の実装と揃える)
			if strings.Contains(spec.query, "t.deleted_at IS NULL") {
				assert.Contains(t, spec.deleteQuery, "SET deleted_at = now()", "論理削除すべきテーブルを物理削除している")
				assert.Contains(t, spec.deleteQuery, "t.deleted_at IS NULL", "削除済みの行まで対象にしている")
			}
		})
	}

	// 退会処理が削除するテーブルが定義から漏れていないことを確認する
	for _, name := range []string{
		"records", "matches", "games", "decks", "deck_codes", "users_players", "unofficial_events",
	} {
		assert.True(t, names[name], "%s の定義が無い", name)
	}
}
