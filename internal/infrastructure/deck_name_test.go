package infrastructure

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

var deckNameAliasColumns = []string{"alias", "position", "pokemon_sprite_id"}
var pokemonSpriteColumns = []string{"id", "name"}

func TestNormalizeDeckName(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T){
		"半角カナは全角カナになる":     test_NormalizeDeckName_FoldsHalfWidthKana,
		"ひらがなはカタカナになる":     test_NormalizeDeckName_HiraganaToKatakana,
		"アルファベットは除去される":    test_NormalizeDeckName_StripsAlphabet,
		"漢字は除去される":         test_NormalizeDeckName_StripsKanji,
		"空白と記号は除去される":      test_NormalizeDeckName_StripsSpacesAndSymbols,
		"長音と数字は保持される":      test_NormalizeDeckName_KeepsChoonAndDigits,
		"空文字は空文字のまま":       test_NormalizeDeckName_Empty,
		"記号のみは空文字になる":      test_NormalizeDeckName_SymbolsOnly,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_NormalizeDeckName_FoldsHalfWidthKana(t *testing.T) {
	require.Equal(t, "リザードン", NormalizeDeckName("ﾘｻﾞｰﾄﾞﾝ"))
}

func test_NormalizeDeckName_HiraganaToKatakana(t *testing.T) {
	require.Equal(t, "リザードン", NormalizeDeckName("りざーどん"))
}

// 「ex」「EX」等の修飾語で同じデッキが別キーに割れないよう、アルファベットは
// 半角/全角・大文字/小文字を問わず除去する。英字のみの名前は空になる(集計対象外)。
func test_NormalizeDeckName_StripsAlphabet(t *testing.T) {
	require.Equal(t, "シロナノガブリアス", NormalizeDeckName("シロナノガブリアスex"))
	require.Equal(t, "リザードン", NormalizeDeckName("リザードンＥＸ"))
	require.Equal(t, "", NormalizeDeckName("Lost Box"))
}

// 「改」「型」「〜版」等の修飾語で同じデッキが別キーに割れないよう、漢字は除去する。
func test_NormalizeDeckName_StripsKanji(t *testing.T) {
	require.Equal(t, "バシャドラ", NormalizeDeckName("バシャドラ改"))
	require.Equal(t, "ゲコゾロ", NormalizeDeckName("ゲコゾロ宮古版"))
	require.Equal(t, "オロチン", NormalizeDeckName("前オロチン"))
	require.Equal(t, "", NormalizeDeckName("不明"))
}

func test_NormalizeDeckName_StripsSpacesAndSymbols(t *testing.T) {
	require.Equal(t, "ロストバレット", NormalizeDeckName("ロスト・バレット"))
	require.Equal(t, "リザードンピジョット", NormalizeDeckName("リザードンex／ピジョット!"))
}

func test_NormalizeDeckName_KeepsChoonAndDigits(t *testing.T) {
	require.Equal(t, "サーフゴー2", NormalizeDeckName("サーフゴー 2"))
}

func test_NormalizeDeckName_Empty(t *testing.T) {
	require.Equal(t, "", NormalizeDeckName(""))
}

func test_NormalizeDeckName_SymbolsOnly(t *testing.T) {
	require.Equal(t, "", NormalizeDeckName("!?・ 【】"))
}

func TestDeckNameMatcher(t *testing.T) {
	// loadMatcher は辞書2クエリ(deck_name_aliases → pokemon_sprites)の期待を積んで
	// マッチャを組み立てる共通ヘルパー。
	loadMatcher := func(
		t *testing.T,
		aliasRows *sqlmock.Rows,
		spriteRows *sqlmock.Rows,
	) (*deckNameMatcher, sqlmock.Sqlmock) {
		t.Helper()

		db, mock := setupSqlmockDB(t)
		mock.ExpectQuery(`SELECT \* FROM "deck_name_aliases" ORDER BY alias ASC, position ASC`).
			WillReturnRows(aliasRows)
		mock.ExpectQuery(`SELECT \* FROM "pokemon_sprites" ORDER BY id ASC`).
			WillReturnRows(spriteRows)

		m, err := loadDeckNameMatcher(context.Background(), db)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())

		return m, mock
	}

	t.Run("正常系_最長一致のエイリアスを採用する", func(t *testing.T) {
		// 「リザ」と「リザードン」の両方が引っかかる名前では長い方が勝つ
		m, _ := loadMatcher(t,
			sqlmock.NewRows(deckNameAliasColumns).
				AddRow("リザ", 1, "0004").
				AddRow("リザードン", 1, "0006"),
			sqlmock.NewRows(pokemonSpriteColumns),
		)

		sprites := m.guess("リザードンex")
		require.Len(t, sprites, 1)
		require.Equal(t, "0006", sprites[0].id)

		// 短い方しか含まない名前は短い方に落ちる
		sprites = m.guess("リザ軸")
		require.Len(t, sprites, 1)
		require.Equal(t, "0004", sprites[0].id)
	})

	t.Run("正常系_正式名の包含関係も最長一致で解決する", func(t *testing.T) {
		// マスタの「リザード」「リザードン」は前者が後者の接頭辞。長い方が先に評価される
		m, _ := loadMatcher(t,
			sqlmock.NewRows(deckNameAliasColumns),
			sqlmock.NewRows(pokemonSpriteColumns).
				AddRow("0005", "リザード").
				AddRow("0006", "リザードン"),
		)

		sprites := m.guess("リザードンピジョット")
		require.Len(t, sprites, 1)
		require.Equal(t, "0006", sprites[0].id)

		sprites = m.guess("リザードだけ")
		require.Len(t, sprites, 1)
		require.Equal(t, "0005", sprites[0].id)
	})

	t.Run("正常系_エイリアス辞書が正式名より優先される", func(t *testing.T) {
		// 正式名「リザードン」(1体)を辞書で代表2体に上書きする
		m, _ := loadMatcher(t,
			sqlmock.NewRows(deckNameAliasColumns).
				AddRow("リザードン", 1, "0006").
				AddRow("リザードン", 2, "0018"),
			sqlmock.NewRows(pokemonSpriteColumns).
				AddRow("0006", "リザードン"),
		)

		sprites := m.guess("リザードンex")
		require.Len(t, sprites, 2)
		require.Equal(t, "0006", sprites[0].id)
		require.Equal(t, uint(1), sprites[0].position)
		require.Equal(t, "0018", sprites[1].id)
		require.Equal(t, uint(2), sprites[1].position)
	})

	t.Run("正常系_表記ゆれした名前でもヒットする", func(t *testing.T) {
		m, _ := loadMatcher(t,
			sqlmock.NewRows(deckNameAliasColumns).
				AddRow("ロスバレ", 1, "0487_origin").
				AddRow("ロスバレ", 2, "0225"),
			sqlmock.NewRows(pokemonSpriteColumns),
		)

		for _, name := range []string{"ロスバレ", "ろすばれ", "ﾛｽﾊﾞﾚ", "ロス バレ改"} {
			sprites := m.guess(name)
			require.Len(t, sprites, 2, "name=%s", name)
			require.Equal(t, "0487_origin", sprites[0].id, "name=%s", name)
		}
	})

	t.Run("正常系_ヒットしない名前はnilを返す", func(t *testing.T) {
		m, _ := loadMatcher(t,
			sqlmock.NewRows(deckNameAliasColumns).AddRow("リザ", 1, "0006"),
			sqlmock.NewRows(pokemonSpriteColumns),
		)

		require.Nil(t, m.guess("未知のデッキ"))
		require.Nil(t, m.guess(""))
	})

	// 表示スロットは2枠のため、エイリアスの3体目以降(position>2)は読み込まない
	t.Run("正常系_エイリアスの3体目以降は無視される", func(t *testing.T) {
		m, _ := loadMatcher(t,
			sqlmock.NewRows(deckNameAliasColumns).
				AddRow("トリオデッキ", 1, "0006").
				AddRow("トリオデッキ", 2, "0018").
				AddRow("トリオデッキ", 3, "0400"),
			sqlmock.NewRows(pokemonSpriteColumns),
		)

		sprites := m.guess("トリオデッキ")
		require.Len(t, sprites, 2)
		require.Equal(t, "0006", sprites[0].id)
		require.Equal(t, "0018", sprites[1].id)
	})

	t.Run("正常系_正規化後2文字未満のエイリアスは無視される", func(t *testing.T) {
		// 1文字エイリアスが全名にマッチする事故を防ぐ
		m, _ := loadMatcher(t,
			sqlmock.NewRows(deckNameAliasColumns).AddRow("リ", 1, "0006"),
			sqlmock.NewRows(pokemonSpriteColumns),
		)

		require.Nil(t, m.guess("リザードン"))
	})

	t.Run("異常系_辞書クエリのエラーをそのまま返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		mock.ExpectQuery(`SELECT \* FROM "deck_name_aliases"`).WillReturnError(sql.ErrConnDone)

		m, err := loadDeckNameMatcher(context.Background(), db)
		require.Error(t, err)
		require.Nil(t, m)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestFindDeckNamesByDeckIds(t *testing.T) {
	t.Run("正常系_deck_idごとの名前を返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)

		mock.ExpectQuery(`SELECT id, name FROM "decks" WHERE id IN`).
			WithArgs("deck-1", "deck-2").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
				AddRow("deck-1", "リザードンex").
				AddRow("deck-2", "ロスバレ"))

		// 引数はソートされて渡ることも確認する(WithArgs は deck-1, deck-2 の順)
		names, err := findDeckNamesByDeckIds(context.Background(), db, []string{"deck-2", "deck-1"})

		require.NoError(t, err)
		require.Equal(t, map[string]string{"deck-1": "リザードンex", "deck-2": "ロスバレ"}, names)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_空のIDリストはクエリせず空mapを返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)

		names, err := findDeckNamesByDeckIds(context.Background(), db, nil)

		require.NoError(t, err)
		require.Empty(t, names)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
