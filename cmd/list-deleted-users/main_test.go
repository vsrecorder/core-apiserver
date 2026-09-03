package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jst は日付の境界の解釈を検証するためのタイムゾーン。CIのrunnerはUTC、開発機・本番はJSTで
// 動くため、time.Local を使うと同じテストが環境によって別の結果になる。
var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

func TestBuildFilterCondition(t *testing.T) {
	t.Run("正常系_未指定なら絞り込まない", func(t *testing.T) {
		cond, err := buildFilterCondition("", "", "", false, jst)

		require.NoError(t, err)
		assert.Empty(t, cond.UserID)
		assert.True(t, cond.Since.IsZero())
		assert.True(t, cond.Until.IsZero())
		assert.Empty(t, cond.String())
	})

	t.Run("正常系_untilは指定した日を含むよう翌日0時になる", func(t *testing.T) {
		cond, err := buildFilterCondition("uid_1", "2026-01-01", "2026-03-31", false, jst)

		require.NoError(t, err)
		assert.Equal(t, "uid_1", cond.UserID)
		assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, jst), cond.Since)
		assert.Equal(t, time.Date(2026, 4, 1, 0, 0, 0, 0, jst), cond.Until)
	})

	t.Run("正常系_日付の境界は渡したLocationで解釈する", func(t *testing.T) {
		cond, err := buildFilterCondition("", "2026-01-01", "", false, time.UTC)

		require.NoError(t, err)
		// JSTで解釈した場合の 2026-01-01 00:00 は UTC では前日15時。Location を取り違えると
		// 境界付近の退会ユーザが1件ずれるため、渡した Location がそのまま使われることを確認する。
		assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), cond.Since)
		assert.False(t, cond.Since.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, jst)))
	})

	t.Run("異常系_日付として解釈できない", func(t *testing.T) {
		_, err := buildFilterCondition("", "2026/01/01", "", false, jst)

		assert.Error(t, err)
	})

	t.Run("異常系_untilが日付として解釈できない", func(t *testing.T) {
		_, err := buildFilterCondition("", "", "2026-13-01", false, jst)

		assert.Error(t, err)
	})
}

func TestFilterConditionString(t *testing.T) {
	t.Run("正常系_untilは指定された日に戻して表示する", func(t *testing.T) {
		cond, err := buildFilterCondition("uid_1", "2026-01-01", "2026-03-31", false, jst)

		require.NoError(t, err)
		assert.Equal(t, "user-id=uid_1 since=2026-01-01 until=2026-03-31", cond.String())
	})
}

func TestFilterUsers(t *testing.T) {
	// 退会日の新しい順(listDeletedUsers の並び)で用意する
	users := []*deletedUser{
		{ID: "uid_3", DeletedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, jst)},
		{ID: "uid_2", DeletedAt: time.Date(2026, 3, 31, 23, 59, 59, 0, jst)},
		{ID: "uid_1", DeletedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, jst)},
		{ID: "uid_0", DeletedAt: time.Date(2025, 12, 31, 23, 59, 59, 0, jst)},
	}

	t.Run("正常系_条件が無ければ全件を並び順のまま返す", func(t *testing.T) {
		cond, err := buildFilterCondition("", "", "", false, jst)
		require.NoError(t, err)

		matched := filterUsers(users, cond)

		require.Len(t, matched, 4)
		assert.Equal(t, "uid_3", matched[0].ID)
		assert.Equal(t, "uid_0", matched[3].ID)
	})

	t.Run("正常系_期間の両端を含む", func(t *testing.T) {
		cond, err := buildFilterCondition("", "2026-01-01", "2026-03-31", false, jst)
		require.NoError(t, err)

		matched := filterUsers(users, cond)

		// since の当日0時ちょうど(uid_1)と until の当日23:59:59(uid_2)は含み、
		// その外側(uid_0 / uid_3)は含まない
		require.Len(t, matched, 2)
		assert.Equal(t, "uid_2", matched[0].ID)
		assert.Equal(t, "uid_1", matched[1].ID)
	})

	t.Run("正常系_user_idで絞り込む", func(t *testing.T) {
		cond, err := buildFilterCondition("uid_1", "", "", false, jst)
		require.NoError(t, err)

		matched := filterUsers(users, cond)

		require.Len(t, matched, 1)
		assert.Equal(t, "uid_1", matched[0].ID)
	})

	t.Run("正常系_合致しなければ空になる", func(t *testing.T) {
		cond, err := buildFilterCondition("uid_not_found", "", "", false, jst)
		require.NoError(t, err)

		assert.Empty(t, filterUsers(users, cond))
	})
}

func TestUsageDays(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, jst)

	t.Run("正常系_24時間を1日として切り捨てる", func(t *testing.T) {
		assert.Equal(t, 10, usageDays(createdAt, createdAt.Add(10*24*time.Hour)))
		assert.Equal(t, 10, usageDays(createdAt, createdAt.Add(10*24*time.Hour+23*time.Hour)))
	})

	t.Run("正常系_1日に満たなければ0", func(t *testing.T) {
		assert.Equal(t, 0, usageDays(createdAt, createdAt.Add(23*time.Hour)))
	})

	t.Run("正常系_Locationが違っても実時間の差で求める", func(t *testing.T) {
		// 同じ瞬間を別のLocationで表しても日数は変わらない
		assert.Equal(t, 1, usageDays(createdAt, createdAt.Add(24*time.Hour).In(time.UTC)))
	})

	t.Run("異常系_退会日が登録日より前でも負の値を返さない", func(t *testing.T) {
		assert.Equal(t, 0, usageDays(createdAt, createdAt.Add(-24*time.Hour)))
	})
}

func TestDisplayName(t *testing.T) {
	t.Run("正常系_名前が入っていればそのまま返す", func(t *testing.T) {
		assert.Equal(t, "たいち", displayName("たいち"))
	})

	t.Run("正常系_空文字なら未設定と分かる表示にする", func(t *testing.T) {
		assert.Equal(t, "(未設定)", displayName(""))
	})
}

func TestFilterUsers_物理削除済み(t *testing.T) {
	purgedAt := time.Date(2026, 4, 2, 0, 0, 0, 0, jst)

	users := []*deletedUser{
		{ID: "uid_purged", DeletedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, jst), PurgedAt: &purgedAt},
		{ID: "uid_deleted", DeletedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, jst)},
	}

	t.Run("正常系_既定では物理削除済みを一覧に出さない", func(t *testing.T) {
		cond, err := buildFilterCondition("", "", "", false, jst)
		require.NoError(t, err)

		matched := filterUsers(users, cond)

		require.Len(t, matched, 1)
		assert.Equal(t, "uid_deleted", matched[0].ID)
	})

	t.Run("正常系_includePurgedなら物理削除済みも出す", func(t *testing.T) {
		cond, err := buildFilterCondition("", "", "", true, jst)
		require.NoError(t, err)

		assert.Len(t, filterUsers(users, cond), 2)
	})

	t.Run("正常系_user_idで名指ししても既定では出さない", func(t *testing.T) {
		// 明示的に指定されても扱いを変えない。0件になる理由は report がヒントとして表示する
		cond, err := buildFilterCondition("uid_purged", "", "", false, jst)
		require.NoError(t, err)

		assert.Empty(t, filterUsers(users, cond))
	})

	t.Run("正常系_条件の表示にinclude-purgedが出る", func(t *testing.T) {
		cond, err := buildFilterCondition("", "", "", true, jst)
		require.NoError(t, err)

		assert.Equal(t, "include-purged", cond.String())
	})
}

func TestPurgedLabel(t *testing.T) {
	t.Run("正常系_物理削除済みなら日時を添える", func(t *testing.T) {
		purgedAt := time.Date(2026, 4, 2, 15, 4, 5, 0, time.UTC)

		assert.Equal(t, " purged_at=2026-04-02T15:04:05Z", purgedLabel(&purgedAt))
	})

	t.Run("正常系_未実行なら何も付けない", func(t *testing.T) {
		assert.Empty(t, purgedLabel(nil))
	})
}
