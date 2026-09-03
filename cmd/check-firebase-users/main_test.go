package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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

	t.Run("正常系_DB上は退会済みならA", func(t *testing.T) {
		label, state := classifyFirebaseOnly("uid_deleted_in_db", dbUsers)

		assert.Equal(t, "A:退会済み", label)
		assert.Contains(t, state, "DB上は退会済み")
		assert.Contains(t, state, deletedAt.Format(time.RFC3339))
	})

	t.Run("正常系_DBに行が無ければB", func(t *testing.T) {
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

func TestBuildSlackMessage(t *testing.T) {
	deletedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	firebaseUsers := map[string]*firebaseUser{
		"uid_ok":            {UID: "uid_ok"},
		"uid_firebase_only": {UID: "uid_firebase_only"},
		"uid_deleted_in_db": {UID: "uid_deleted_in_db"},
	}
	dbUsers := map[string]*dbUser{
		"uid_ok":            {ID: "uid_ok"},
		"uid_deleted_in_db": {ID: "uid_deleted_in_db", DeletedAt: &deletedAt},
		"uid_db_only":       {ID: "uid_db_only"},
	}

	t.Run("正常系_差異の内訳とUIDが本文に含まれる", func(t *testing.T) {
		message := buildSlackMessage(
			[]string{"uid_deleted_in_db", "uid_firebase_only"},
			[]string{"uid_db_only"},
			dbUsers,
			firebaseUsers,
		)

		assert.Contains(t, message, "firebase: 3 件 / db(有効): 2 件")
		assert.Contains(t, message, "*firebase_only: 2 件* (A:退会済み 1 件 / B:登録未完了 1 件)")
		// 分類を取り違えると「退会済みの人へ再登録を促す」ような誤った対処に繋がるため、
		// UIDとラベルの対応まで確認する
		assert.Contains(t, message, "`uid_deleted_in_db` [A:退会済み]")
		assert.Contains(t, message, "`uid_firebase_only` [B:登録未完了] DBに行なし")
		assert.Contains(t, message, "*db_only: 1 件*")
		assert.Contains(t, message, "`uid_db_only`")
	})

	t.Run("正常系_一方の差異しか無ければもう一方の見出しは出さない", func(t *testing.T) {
		message := buildSlackMessage([]string{}, []string{"uid_db_only"}, dbUsers, firebaseUsers)

		assert.NotContains(t, message, "firebase_only")
		assert.Contains(t, message, "*db_only: 1 件*")
	})

	t.Run("正常系_上限を超えたUIDは件数だけ伝える", func(t *testing.T) {
		var firebaseOnly []string
		manyDBUsers := map[string]*dbUser{}
		for i := 0; i < maxSlackListItems+3; i++ {
			uid := fmt.Sprintf("uid_%02d", i)
			firebaseOnly = append(firebaseOnly, uid)
		}

		message := buildSlackMessage(firebaseOnly, nil, manyDBUsers, firebaseUsers)

		assert.Contains(t, message, "`uid_19`")
		assert.NotContains(t, message, "`uid_20`")
		assert.Contains(t, message, "...他 3 件")
	})
}

func TestNotifyToSlack(t *testing.T) {
	t.Run("正常系_JSONのtextとして送信される", func(t *testing.T) {
		var (
			gotContentType string
			gotBody        []byte
		)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotContentType = r.Header.Get("Content-Type")
			gotBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		defer server.Close()

		err := notifyToSlack(server.URL, "差異があります")

		assert.NoError(t, err)
		assert.Equal(t, "application/json", gotContentType)

		var payload struct {
			Text string `json:"text"`
		}
		assert.NoError(t, json.Unmarshal(gotBody, &payload))
		assert.Equal(t, "差異があります", payload.Text)
	})

	t.Run("異常系_200以外はボディを添えてエラーを返す", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("invalid_payload"))
		}))
		defer server.Close()

		err := notifyToSlack(server.URL, "差異があります")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "400")
		assert.Contains(t, err.Error(), "invalid_payload")
	})

	t.Run("異常系_送信先へ到達できなければエラーを返す", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := server.URL
		server.Close()

		err := notifyToSlack(url, "差異があります")

		assert.Error(t, err)
	})
}
