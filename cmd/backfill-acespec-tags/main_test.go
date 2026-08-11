package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// cleanAceSpecNames の純粋ロジック(目印除去・空/長すぎ除外・重複除去、順序保持)を検証する。
func TestCleanAceSpecNames(t *testing.T) {
	t.Run("正常系_目印を除きトリムする", func(t *testing.T) {
		got := cleanAceSpecNames([]string{"マスターボール(ACE SPEC)"})
		require.Equal(t, []string{"マスターボール"}, got)
	})

	t.Run("正常系_重複は入力順を保って1つにまとめる", func(t *testing.T) {
		got := cleanAceSpecNames([]string{
			"プライムキャッチャー(ACE SPEC)",
			"マスターボール(ACE SPEC)",
			"プライムキャッチャー(ACE SPEC)", // 別カードの同名再録を想定
		})
		require.Equal(t, []string{"プライムキャッチャー", "マスターボール"}, got)
	})

	t.Run("正常系_空になる名前と長すぎる名前は除外する", func(t *testing.T) {
		tooLong := strings.Repeat("あ", maxTagNameLength+1) + aceSpecSuffix
		got := cleanAceSpecNames([]string{
			aceSpecSuffix, // 目印を除くと空
			tooLong,       // 32文字超
			"ポケストップ(ACE SPEC)",
		})
		require.Equal(t, []string{"ポケストップ"}, got)
	})
}

// fetchAceSpecCardNames が cards の「名前ごとの最小 card id 昇順」で返すことを実DBで検証する。
// VSRECORDER_TEST_DATABASE_URL 未設定時はスキップ(make integration-test で実行される)。
func TestIntegrationFetchAceSpecCardNames(t *testing.T) {
	dsn := os.Getenv("VSRECORDER_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("VSRECORDER_TEST_DATABASE_URL が未設定のためスキップ(make integration-test で実行できます)")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.Exec("TRUNCATE TABLE cards CASCADE").Error)

	// card id と名前順が一致しないように投入する。
	// - id=300 "アカマツ" (名前は後方)
	// - id=100 "ボスの指令" (名前は中間)  ← 最小idはこれ
	// - id=200 "ネオラントV" (名前は前方)
	// さらに id=250 は id=100 と同名(再録)で、名前ごとの最小 id 判定を確認する。
	insert := `INSERT INTO cards
		(id, collection_code, card_name, card_category, card_sub_category, rare_code,
		 card_image_filename, publish_status, block_code, group_id, pokemon_level,
		 pokemon_hp, pokemon_type, run_away_cost, evolution_number, great_pokemon_code,
		 regulation, regulation_mark)
		VALUES (?, '', ?, 0,0,0, '', 0, '', 0, 0, 0, 0, '', 0, 0, '', ?)`

	rows := []struct {
		id   int
		name string
		mark string
	}{
		{300, "アカマツ(ACE SPEC)", "H"},
		{100, "ボスの指令(ACE SPEC)", "H"},
		{200, "ネオラントV(ACE SPEC)", "H"},
		{250, "ボスの指令(ACE SPEC)", "H"}, // id=100 と同名(最小idは100)
		{50, "ふしぎなアメ", "H"},           // ACE SPEC でない → 対象外
		{10, "アカマツ(ACE SPEC)", "I"},   // 別レギュレーション → 対象外
	}
	for _, r := range rows {
		require.NoError(t, db.Exec(insert, r.id, r.name, r.mark).Error)
	}

	// 取得: レギュレーション H の ACE SPEC を最小 card id 昇順で。
	rawNames, err := fetchAceSpecCardNames(db, "H")
	require.NoError(t, err)

	// 目印付きのまま、最小 id 昇順で返る(id=100 ボス, id=200 ネオラント, id=300 アカマツ)。
	require.Equal(t, []string{
		"ボスの指令(ACE SPEC)",
		"ネオラントV(ACE SPEC)",
		"アカマツ(ACE SPEC)",
	}, rawNames)

	// クリーニング後も同じ順(=card id 昇順)を保つ。
	require.Equal(t, []string{"ボスの指令", "ネオラントV", "アカマツ"}, cleanAceSpecNames(rawNames))
}
