package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// specs は物理削除の対象そのものであり、定義を取り違えると復旧できない削除になる。
// 定義の形と、消してはいけないものが対象に入っていないことを検査する。
func TestSpecs(t *testing.T) {
	require.NotEmpty(t, specs)

	names := make(map[string]bool)

	for _, spec := range specs {
		t.Run("正常系_"+spec.name, func(t *testing.T) {
			assert.False(t, names[spec.name], "テーブル定義が重複している")
			names[spec.name] = true

			assert.NotEmpty(t, spec.note, "件数表示で何のデータか分かるよう note は必須")
			assert.Contains(t, spec.where, "@user_id", "対象ユーザに絞り込んでいない")

			// 「数えた行」と「消す行」が食い違うと、消し残しに気づけないまま完了してしまう。
			// 両方が同じ where から組み立てられていることを保証する。
			assert.Equal(t, "SELECT COUNT(*) FROM "+spec.name+" WHERE "+spec.where, spec.countQuery())
			assert.Equal(t, "DELETE FROM "+spec.name+" WHERE "+spec.where, spec.deleteQuery())

			// 他人のデータを書き換えない(消すのは対象ユーザの親を参照している行だけ)
			assert.NotContains(t, spec.deleteQuery(), "UPDATE", "行の削除以外を行っている")
			assert.NotContains(t, spec.where, "opponents_user_id", "他人の対戦記録の参照を対象にしている")

			if spec.foreignWhere != "" {
				assert.Contains(t, spec.foreignWhere, "user_id <> @user_id", "他ユーザの行に絞り込んでいない")
				assert.Equal(t, "SELECT COUNT(*) FROM "+spec.name+" WHERE "+spec.foreignWhere, spec.foreignCountQuery())
			}
		})
	}

	t.Run("正常系_usersの行は削除対象にしない", func(t *testing.T) {
		// users を消すと usecase.User.Create の IsWithdrawn による再登録拒否が効かなくなる
		assert.False(t, names["users"], "users を削除対象にしてはいけない")
	})

	// 「user_id を持つテーブル」と「それらへFKで繋がる中間テーブル」の全部が対象。
	// cmd/check-deleted-users-data の specs と1対1で対応する(あちらは残存確認、こちらは物理削除)。
	t.Run("正常系_対象テーブルに漏れがない", func(t *testing.T) {
		for _, name := range []string{
			// user_id を持つもの
			"records", "matches", "games", "decks", "deck_codes", "unofficial_events",
			"tags", "user_favorite_decks", "user_streaks", "user_daily_activities",
			"user_badges", "user_environment_badges", "notifications", "users_players",
			"push_subscriptions", "push_deliveries", "user_acquisitions", "user_gyms",
			// 中間テーブル(user_id を持たず、上のテーブルからFKでたどる)
			"deck_tags", "deck_code_tags", "record_tags", "match_tags",
			"match_pokemon_sprites", "deck_pokemon_sprites",
		} {
			assert.True(t, names[name], "%s の定義が無い", name)
		}
	})
}

// specs の並び順はそのまま削除順になる。FK制約があるため、子テーブルを先に消さないと
// 親テーブルの物理削除が失敗する。並べ替えたときに気づけるよう依存関係を検査する。
func TestSpecsDeleteOrder(t *testing.T) {
	index := make(map[string]int, len(specs))
	for i, spec := range specs {
		index[spec.name] = i
	}

	for _, dependency := range []struct {
		child  string
		parent string
	}{
		{"match_tags", "matches"},
		{"match_pokemon_sprites", "matches"},
		{"games", "matches"},
		{"matches", "records"},
		{"record_tags", "records"},
		{"deck_code_tags", "deck_codes"},
		{"deck_codes", "decks"},
		{"deck_tags", "decks"},
		{"deck_pokemon_sprites", "decks"},
		{"user_favorite_decks", "decks"},
		// タグは各種中間テーブルから参照されるため、それらより後に消す
		{"deck_tags", "tags"},
		{"deck_code_tags", "tags"},
		{"record_tags", "tags"},
		{"match_tags", "tags"},
	} {
		childIndex, ok := index[dependency.child]
		require.True(t, ok, "%s の定義が無い", dependency.child)

		parentIndex, ok := index[dependency.parent]
		require.True(t, ok, "%s の定義が無い", dependency.parent)

		assert.Less(t, childIndex, parentIndex,
			"%s は %s より先に削除する必要がある(FK制約)", dependency.child, dependency.parent)
	}
}

// 親のサブクエリは各テーブルの条件へ埋め込まれるため、対象の取り違えが広範囲に波及する。
func TestTargetSubqueries(t *testing.T) {
	t.Run("正常系_親は対象ユーザのものだけを選ぶ", func(t *testing.T) {
		assert.Equal(t, "SELECT id FROM records WHERE user_id = @user_id", targetRecords)
		assert.Equal(t, "SELECT id FROM decks WHERE user_id = @user_id", targetDecks)
	})

	t.Run("正常系_親を消せるよう親経由の子も対象に含める", func(t *testing.T) {
		// deck_codes.deck_id / matches.record_id はFKのため、残っていると親を物理削除できない
		assert.Contains(t, targetDeckCodes, "deck_id IN ("+targetDecks+")")
		assert.Contains(t, targetMatches, "record_id IN ("+targetRecords+")")
	})
}

// プリセットタグ(user_id が空文字)は全ユーザ共通のマスタで、消すと他のユーザに影響する。
func TestTagsSpecDoesNotTouchPresets(t *testing.T) {
	var tagsSpec *tableSpec
	for i := range specs {
		if specs[i].name == "tags" {
			tagsSpec = &specs[i]
			break
		}
	}

	require.NotNil(t, tagsSpec)

	// user_id が空文字のプリセットは、対象ユーザのIDと一致しない限り選ばれない。
	// 対象ユーザは「退会済みユーザとして存在すること」を確認済みのため空文字にはならない。
	assert.Equal(t, "user_id = @user_id", tagsSpec.where)
	assert.Empty(t, tagsSpec.foreignWhere, "プリセットを巻き込み対象にしてはいけない")
}

func TestTotal(t *testing.T) {
	t.Run("正常系_件数を合計する", func(t *testing.T) {
		counts := []tableCount{
			{table: "records", count: 3},
			{table: "matches", count: 5},
		}

		assert.Equal(t, int64(8), total(counts))
	})

	t.Run("正常系_対象が無ければ0", func(t *testing.T) {
		assert.Equal(t, int64(0), total(nil))
	})
}

func TestDisplayName(t *testing.T) {
	t.Run("正常系_名前が入っていればそのまま返す", func(t *testing.T) {
		assert.Equal(t, "たいち", displayName("たいち"))
	})

	t.Run("正常系_users_nameがNULLなら空欄にしない", func(t *testing.T) {
		assert.Equal(t, "名前未設定", displayName(""))
	})
}

// 生成するSQLは名前付きプレースホルダを使う。? に置き換わっていると、条件へ何度も
// user_id が現れるテーブルで引数の数が合わずに失敗する。
func TestQueriesUseNamedParameter(t *testing.T) {
	for _, spec := range specs {
		assert.NotContains(t, spec.countQuery(), "?", "%s: 位置プレースホルダを使っている", spec.name)
		assert.NotContains(t, spec.deleteQuery(), "?", "%s: 位置プレースホルダを使っている", spec.name)
		assert.Equal(t, 1, strings.Count(spec.countQuery(), "SELECT COUNT(*)"))
	}
}
