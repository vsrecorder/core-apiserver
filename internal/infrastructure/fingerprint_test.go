package infrastructure

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeFingerprint(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T){
		"EmptyReturnsEmpty":                test_NormalizeFingerprint_EmptyReturnsEmpty,
		"SingleSprite":                     test_NormalizeFingerprint_SingleSprite,
		"OrderIsFullyNormalized":           test_NormalizeFingerprint_OrderIsFullyNormalized,
		"DuplicatesAreRemoved":             test_NormalizeFingerprint_DuplicatesAreRemoved,
		"SameSetProducesSameKeyRegardless": test_NormalizeFingerprint_SameSetProducesSameKeyRegardlessOfOrder,
		"DifferentSetProducesDifferentKey": test_NormalizeFingerprint_DifferentSetProducesDifferentKey,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_NormalizeFingerprint_EmptyReturnsEmpty(t *testing.T) {
	key, normalized := NormalizeFingerprint([]string{})
	require.Equal(t, "", key)
	require.Nil(t, normalized)
}

func test_NormalizeFingerprint_SingleSprite(t *testing.T) {
	key, normalized := NormalizeFingerprint([]string{"0006"})
	require.Equal(t, "0006", key)
	require.Equal(t, []string{"0006"}, normalized)
}

// 先頭を含め、どの並び順でも同一キーに正規化されることを検証する（順番は考慮しない）。
func test_NormalizeFingerprint_OrderIsFullyNormalized(t *testing.T) {
	keyA, normA := NormalizeFingerprint([]string{"0359", "0006", "0018"})
	keyB, normB := NormalizeFingerprint([]string{"0006", "0018", "0359"})
	keyC, normC := NormalizeFingerprint([]string{"0018", "0359", "0006"})

	require.Equal(t, keyA, keyB)
	require.Equal(t, keyA, keyC)
	require.Equal(t, []string{"0006", "0018", "0359"}, normA)
	require.Equal(t, normA, normB)
	require.Equal(t, normA, normC)
}

// 重複スプライトが（先頭も含めて）除去されることを検証する。
func test_NormalizeFingerprint_DuplicatesAreRemoved(t *testing.T) {
	key, normalized := NormalizeFingerprint([]string{"0018", "0006", "0018", "0006"})
	require.Equal(t, []string{"0006", "0018"}, normalized)
	require.Equal(t, "0006,0018", key)
}

// 先頭が異なっていても、スプライトの集合が同じであれば同一キーになることを検証する
// （順番を考慮しないため、旧仕様の「先頭=大分類アンカー」は撤廃されている）。
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
