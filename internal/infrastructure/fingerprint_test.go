package infrastructure

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeFingerprint(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T){
		"EmptyReturnsEmpty":                test_NormalizeFingerprint_EmptyReturnsEmpty,
		"SingleSprite":                     test_NormalizeFingerprint_SingleSprite,
		"KeyIsOrderIndependent":            test_NormalizeFingerprint_KeyIsOrderIndependent,
		"OrderedPreservesOriginalOrder":    test_NormalizeFingerprint_OrderedPreservesOriginalOrder,
		"DuplicatesAreRemovedFromOrdered":  test_NormalizeFingerprint_DuplicatesAreRemovedFromOrdered,
		"SameSetProducesSameKeyRegardless": test_NormalizeFingerprint_SameSetProducesSameKeyRegardlessOfOrder,
		"DifferentSetProducesDifferentKey": test_NormalizeFingerprint_DifferentSetProducesDifferentKey,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_NormalizeFingerprint_EmptyReturnsEmpty(t *testing.T) {
	key, ordered := NormalizeFingerprint([]string{})
	require.Equal(t, "", key)
	require.Nil(t, ordered)
}

func test_NormalizeFingerprint_SingleSprite(t *testing.T) {
	key, ordered := NormalizeFingerprint([]string{"0006"})
	require.Equal(t, "0006", key)
	require.Equal(t, []string{"0006"}, ordered)
}

// 集計キーはどの並び順でも同一になることを検証する（グルーピングは順番を考慮しない）。
func test_NormalizeFingerprint_KeyIsOrderIndependent(t *testing.T) {
	keyA, _ := NormalizeFingerprint([]string{"0359", "0006", "0018"})
	keyB, _ := NormalizeFingerprint([]string{"0006", "0018", "0359"})
	keyC, _ := NormalizeFingerprint([]string{"0018", "0359", "0006"})

	require.Equal(t, keyA, keyB)
	require.Equal(t, keyA, keyC)
}

// 表示用スプライト列（ordered）はソートされず、元データの並び順をそのまま保つことを検証する。
func test_NormalizeFingerprint_OrderedPreservesOriginalOrder(t *testing.T) {
	_, orderedA := NormalizeFingerprint([]string{"0359", "0006", "0018"})
	_, orderedB := NormalizeFingerprint([]string{"0006", "0018", "0359"})

	require.Equal(t, []string{"0359", "0006", "0018"}, orderedA)
	require.Equal(t, []string{"0006", "0018", "0359"}, orderedB)
}

// 表示用スプライト列は重複排除だけ行い、最初に出現した位置の順序を保つことを検証する。
func test_NormalizeFingerprint_DuplicatesAreRemovedFromOrdered(t *testing.T) {
	key, ordered := NormalizeFingerprint([]string{"0018", "0006", "0018", "0006"})
	require.Equal(t, []string{"0018", "0006"}, ordered)
	// キーは重複排除・ソート済みなので並びに依存しない
	require.Equal(t, "0006,0018", key)
}

// 先頭が異なっていても、スプライトの集合が同じであれば同一キーになることを検証する
// （集計キーは順番を考慮しないため）。
func test_NormalizeFingerprint_SameSetProducesSameKeyRegardlessOfOrder(t *testing.T) {
	keyA, _ := NormalizeFingerprint([]string{"0006", "0018"})
	keyB, _ := NormalizeFingerprint([]string{"0018", "0006"})
	require.Equal(t, keyA, keyB)
}

// スプライトの構成そのものが異なれば別キーとして扱われることを検証する。
func test_NormalizeFingerprint_DifferentSetProducesDifferentKey(t *testing.T) {
	keyA, _ := NormalizeFingerprint([]string{"0006", "0018"})
	keyB, _ := NormalizeFingerprint([]string{"0006", "0359"})
	require.NotEqual(t, keyA, keyB)
}
