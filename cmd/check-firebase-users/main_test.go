package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDiff(t *testing.T) {
	deletedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)

	firebaseUsers := map[string]*firebaseUser{
		// 正常: Firebase・DBの両方に有効なユーザーとして存在する
		"uid_ok": {UID: "uid_ok"},
		// 異常: DBに行が無い(ユーザー作成がDB登録前に失敗した等)
		"uid_firebase_only": {UID: "uid_firebase_only"},
		// 異常: DB上は退会済みなのにFirebaseのユーザーが残っている
		"uid_deleted_in_db": {UID: "uid_deleted_in_db"},
	}

	dbUsers := map[string]*dbUser{
		"uid_ok":            {ID: "uid_ok"},
		"uid_deleted_in_db": {ID: "uid_deleted_in_db", DeletedAt: &deletedAt},
		// 異常: DBには有効なユーザーとして存在するがFirebaseに存在しない
		"uid_db_only": {ID: "uid_db_only"},
		// 正常: 退会済みでFirebaseにも存在しない(差異として扱わない)
		"uid_deleted_both": {ID: "uid_deleted_both", DeletedAt: &deletedAt},
	}

	firebaseOnly, dbOnly := diff(firebaseUsers, dbUsers)

	assert.Equal(t, []string{"uid_deleted_in_db", "uid_firebase_only"}, firebaseOnly)
	assert.Equal(t, []string{"uid_db_only"}, dbOnly)
}

func TestClassifyFirebaseOnly(t *testing.T) {
	deletedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)

	dbUsers := map[string]*dbUser{
		"uid_deleted_in_db": {ID: "uid_deleted_in_db", DeletedAt: &deletedAt},
	}

	t.Run("DB上は退会済みならA", func(t *testing.T) {
		label, state := classifyFirebaseOnly("uid_deleted_in_db", dbUsers)

		assert.Equal(t, "A:退会済み", label)
		assert.Contains(t, state, "DB上は退会済み")
		assert.Contains(t, state, deletedAt.Format(time.RFC3339))
	})

	t.Run("DBに行が無ければB", func(t *testing.T) {
		label, state := classifyFirebaseOnly("uid_firebase_only", dbUsers)

		assert.Equal(t, "B:登録未完了", label)
		assert.Equal(t, "DBに行なし", state)
	})
}

func TestDiff_差異が無い場合(t *testing.T) {
	deletedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)

	firebaseUsers := map[string]*firebaseUser{
		"uid_1": {UID: "uid_1"},
		"uid_2": {UID: "uid_2"},
	}

	dbUsers := map[string]*dbUser{
		"uid_1": {ID: "uid_1"},
		"uid_2": {ID: "uid_2"},
		// 退会済みかつFirebaseにも存在しないユーザーは差異にならない
		"uid_3": {ID: "uid_3", DeletedAt: &deletedAt},
	}

	firebaseOnly, dbOnly := diff(firebaseUsers, dbUsers)

	assert.Empty(t, firebaseOnly)
	assert.Empty(t, dbOnly)
}
