package controller

import (
	"math/rand"
	"time"

	ulid "github.com/oklog/ulid/v2"
)

var (
	entropy = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// timeNow は現在時刻の取得関数。現在のシーズン判定に使う時刻をテストから
// 固定できるよう変数にしている。
var timeNow = time.Now

func generateId() (string, error) {
	ms := ulid.Timestamp(time.Now())
	id, err := ulid.New(ms, entropy)

	return id.String(), err
}
