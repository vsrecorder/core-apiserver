package infrastructure

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeFingerprint(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T){
		"EmptyReturnsEmpty":                    test_NormalizeFingerprint_EmptyReturnsEmpty,
		"SingleSprite":                         test_NormalizeFingerprint_SingleSprite,
		"OrderOfNonPrimaryIsNormalized":        test_NormalizeFingerprint_OrderOfNonPrimaryIsNormalized,
		"DuplicatesAreRemoved":                 test_NormalizeFingerprint_DuplicatesAreRemoved,
		"PrimaryIsPreservedAsAnchor":           test_NormalizeFingerprint_PrimaryIsPreservedAsAnchor,
		"TextIsNotPartOfKey":                   test_NormalizeFingerprint_TextIsNotPartOfKey,
		"DifferentPrimaryProducesDifferentKey": test_NormalizeFingerprint_DifferentPrimaryProducesDifferentKey,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_NormalizeFingerprint_EmptyReturnsEmpty(t *testing.T) {
	key, primary, normalized := NormalizeFingerprint([]string{})
	require.Equal(t, "", key)
	require.Equal(t, "", primary)
	require.Nil(t, normalized)
}

func test_NormalizeFingerprint_SingleSprite(t *testing.T) {
	key, primary, normalized := NormalizeFingerprint([]string{"0006"})
	require.Equal(t, "0006|", key)
	require.Equal(t, "0006", primary)
	require.Equal(t, []string{"0006"}, normalized)
}

// 先頭(代表)以外の並び順が違っても同一キーに正規化されることを検証する。
func test_NormalizeFingerprint_OrderOfNonPrimaryIsNormalized(t *testing.T) {
	keyA, _, normA := NormalizeFingerprint([]string{"0006", "0018", "0359"})
	keyB, _, normB := NormalizeFingerprint([]string{"0006", "0359", "0018"})

	require.Equal(t, keyA, keyB)
	require.Equal(t, normA, normB)
	require.Equal(t, []string{"0006", "0018", "0359"}, normA)
}

// 重複スプライトが除去されることを検証する。
func test_NormalizeFingerprint_DuplicatesAreRemoved(t *testing.T) {
	key, primary, normalized := NormalizeFingerprint([]string{"0006", "0018", "0018", "0006"})
	require.Equal(t, "0006", primary)
	require.Equal(t, []string{"0006", "0018"}, normalized)
	require.Equal(t, "0006|0018", key)
}

// 先頭(Position1)は大分類のアンカーとして保持されることを検証する。
func test_NormalizeFingerprint_PrimaryIsPreservedAsAnchor(t *testing.T) {
	_, primary, normalized := NormalizeFingerprint([]string{"0359", "0006", "0018"})
	require.Equal(t, "0359", primary)
	require.Equal(t, "0359", normalized[0])
	// 残りはソートされる
	require.Equal(t, []string{"0359", "0006", "0018"}, normalized)
}

// フリーテキストはキーに含まれない（この関数はスプライトのみを受け取る）ため、
// 同一指紋であればキーが一致することを検証する。
func test_NormalizeFingerprint_TextIsNotPartOfKey(t *testing.T) {
	keyA, _, _ := NormalizeFingerprint([]string{"0006", "0018"})
	keyB, _, _ := NormalizeFingerprint([]string{"0006", "0018"})
	require.Equal(t, keyA, keyB)
}

// 代表(先頭)が異なる場合は別デッキ（別キー）として扱われることを検証する。
func test_NormalizeFingerprint_DifferentPrimaryProducesDifferentKey(t *testing.T) {
	keyA, _, _ := NormalizeFingerprint([]string{"0006", "0018"})
	keyB, _, _ := NormalizeFingerprint([]string{"0018", "0006"})
	require.NotEqual(t, keyA, keyB)
}
