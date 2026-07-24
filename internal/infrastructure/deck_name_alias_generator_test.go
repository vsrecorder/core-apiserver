package infrastructure

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

var (
	deckNameSupplyColumns = []string{"name", "user_id", "layout"}
	deckNameDemandColumns = []string{"name", "count"}
)

func TestParseDeckNameLayout(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T){
		"1体":            test_parseDeckNameLayout_Single,
		"2体はposition順": test_parseDeckNameLayout_SortedByPosition,
		"空文字はnil":       test_parseDeckNameLayout_Empty,
		"壊れた要素は読み飛ばす":   test_parseDeckNameLayout_SkipsBroken,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_parseDeckNameLayout_Single(t *testing.T) {
	sprites := parseDeckNameLayout("1:0006")
	require.Equal(t, []DeckNameAliasSprite{{PokemonSpriteId: "0006", Position: 1}}, sprites)
}

func test_parseDeckNameLayout_SortedByPosition(t *testing.T) {
	sprites := parseDeckNameLayout("2:0018,1:0006")
	require.Equal(t, []DeckNameAliasSprite{
		{PokemonSpriteId: "0006", Position: 1},
		{PokemonSpriteId: "0018", Position: 2},
	}, sprites)
}

func test_parseDeckNameLayout_Empty(t *testing.T) {
	require.Nil(t, parseDeckNameLayout(""))
}

func test_parseDeckNameLayout_SkipsBroken(t *testing.T) {
	// 区切り無し・position が数値でない・position が 0 の要素は落とす
	sprites := parseDeckNameLayout("0006,x:0018,0:0025,1:0006")
	require.Equal(t, []DeckNameAliasSprite{{PokemonSpriteId: "0006", Position: 1}}, sprites)
}

func TestGenerateDeckNameAliasCandidates(t *testing.T) {
	from := time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 7, 24, 0, 0, 0, 0, time.Local)

	cfg := func() DeckNameAliasGeneratorConfig {
		c := DefaultDeckNameAliasGeneratorConfig()
		c.SupplyFrom, c.SupplyTo = from, to
		c.DemandFrom, c.DemandTo = from, to
		return c
	}

	// expectQueries は 供給(相手/自分) → 需要(相手/自分) → 手動辞書 の順にクエリ期待を積む。
	expectQueries := func(
		mock sqlmock.Sqlmock,
		opponentSupply *sqlmock.Rows,
		ownSupply *sqlmock.Rows,
		opponentDemand *sqlmock.Rows,
		ownDemand *sqlmock.Rows,
		manualAliases *sqlmock.Rows,
	) {
		mock.ExpectQuery(`string_agg\(match_pokemon_sprites`).WillReturnRows(opponentSupply)
		mock.ExpectQuery(`string_agg\(deck_pokemon_sprites`).WillReturnRows(ownSupply)
		mock.ExpectQuery(`COUNT\(\*\).*match_pokemon_sprites\.match_id IS NULL`).WillReturnRows(opponentDemand)
		mock.ExpectQuery(`COUNT\(\*\).*deck_pokemon_sprites\.deck_id IS NULL`).WillReturnRows(ownDemand)
		mock.ExpectQuery(`SELECT \* FROM "deck_name_aliases" WHERE source = \$1`).
			WithArgs("manual").
			WillReturnRows(manualAliases)
	}

	// supplyRows は同じ名前・同じ構成の教師データを users 人ぶん・count 件生成する。
	supplyRows := func(name, layout string, count int, users int) *sqlmock.Rows {
		rows := sqlmock.NewRows(deckNameSupplyColumns)
		for i := 0; i < count; i++ {
			rows = rows.AddRow(name, "user-"+string(rune('a'+i%users)), layout)
		}
		return rows
	}

	t.Run("正常系_しきい値を満たす候補を生成する", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)

		expectQueries(mock,
			// 教師データ: 「ロスバレ」= 0487_origin+0225 が 12件/4人
			supplyRows("ロスバレ", "1:0487_origin,2:0225", 12, 4),
			sqlmock.NewRows(deckNameSupplyColumns),
			// 需要: スプライト未設定の「ロスバレ」が 30票
			sqlmock.NewRows(deckNameDemandColumns).AddRow("ロスバレ", 30),
			sqlmock.NewRows(deckNameDemandColumns),
			sqlmock.NewRows(deckNameAliasColumns),
		)

		candidates, _, err := GenerateDeckNameAliasCandidates(context.Background(), db, cfg())

		require.NoError(t, err)
		require.Len(t, candidates, 1)

		c := candidates[0]
		require.Equal(t, "ロスバレ", c.Alias)
		require.Equal(t, []DeckNameAliasSprite{
			{PokemonSpriteId: "0487_origin", Position: 1},
			{PokemonSpriteId: "0225", Position: 2},
		}, c.Sprites)
		require.Equal(t, 12, c.Support)
		require.Equal(t, 12, c.TotalSupply)
		require.InDelta(t, 1.0, c.Ratio, 1e-9)
		require.Equal(t, 4, c.Contributors)
		require.Equal(t, 30, c.DemandVotes)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_表記ゆれは正規化して同じ名前に集約する", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)

		// 教師データ側は半角カナ・ひらがな、需要側は全角カナで登録されていても同一視する
		supply := sqlmock.NewRows(deckNameSupplyColumns)
		for i := 0; i < 6; i++ {
			supply = supply.AddRow("ﾛｽﾊﾞﾚ", "user-"+string(rune('a'+i%3)), "1:0487_origin,2:0225")
		}
		for i := 0; i < 6; i++ {
			supply = supply.AddRow("ろすばれ", "user-"+string(rune('d'+i%3)), "1:0487_origin,2:0225")
		}

		expectQueries(mock,
			supply,
			sqlmock.NewRows(deckNameSupplyColumns),
			sqlmock.NewRows(deckNameDemandColumns).AddRow("ロスバレ", 10),
			sqlmock.NewRows(deckNameDemandColumns),
			sqlmock.NewRows(deckNameAliasColumns),
		)

		candidates, _, err := GenerateDeckNameAliasCandidates(context.Background(), db, cfg())

		require.NoError(t, err)
		require.Len(t, candidates, 1)
		require.Equal(t, "ロスバレ", candidates[0].Alias)
		require.Equal(t, 12, candidates[0].Support)
		require.Equal(t, 6, candidates[0].Contributors)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_構成が割れる名前は占有率の下限で保留する", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)

		// 同じ「ミライドン」で 2 構成が半々(占有率 50% < 60%)。
		// 名前は下限(4文字)以上にして、占有率より先に too_short で落ちないようにする。
		supply := sqlmock.NewRows(deckNameSupplyColumns)
		for i := 0; i < 10; i++ {
			supply = supply.AddRow("ミライドン", "user-"+string(rune('a'+i%5)), "1:1008,2:0025")
		}
		for i := 0; i < 10; i++ {
			supply = supply.AddRow("ミライドン", "user-"+string(rune('a'+i%5)), "1:1008,2:0145")
		}

		expectQueries(mock,
			supply,
			sqlmock.NewRows(deckNameSupplyColumns),
			sqlmock.NewRows(deckNameDemandColumns).AddRow("ミライドン", 50),
			sqlmock.NewRows(deckNameDemandColumns),
			sqlmock.NewRows(deckNameAliasColumns),
		)

		candidates, rejected, err := GenerateDeckNameAliasCandidates(context.Background(), db, cfg())

		require.NoError(t, err)
		require.Empty(t, candidates)
		// 占有率不足で落ちたことを理由つきで確認する
		require.Len(t, rejected, 1)
		require.Equal(t, "ミライドン", rejected[0].Alias)
		require.Equal(t, DeckNameAliasRejectLowRatio, rejected[0].Reason)
		require.Equal(t, 50, rejected[0].DemandVotes)
		require.Equal(t, 10, rejected[0].Support)
		require.Equal(t, 20, rejected[0].TotalSupply)
		require.InDelta(t, 0.5, rejected[0].Ratio, 1e-9)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_支持件数と人数の下限を満たさない候補は落とす", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)

		expectQueries(mock,
			// 支持は 12 件あるが 2 人しかいない(下限3人)
			supplyRows("パオジアン", "1:0999,2:0143", 12, 2),
			// 支持が 5 件しかない(下限10件)
			supplyRows("ドラパルト", "1:0887", 5, 5),
			sqlmock.NewRows(deckNameDemandColumns).AddRow("パオジアン", 40).AddRow("ドラパルト", 30),
			sqlmock.NewRows(deckNameDemandColumns),
			sqlmock.NewRows(deckNameAliasColumns),
		)

		candidates, rejected, err := GenerateDeckNameAliasCandidates(context.Background(), db, cfg())

		require.NoError(t, err)
		require.Empty(t, candidates)
		// 需要の多い順(パオジアン40 → ドラパルト30)に、それぞれ別の理由で落ちる
		require.Len(t, rejected, 2)
		require.Equal(t, "パオジアン", rejected[0].Alias)
		require.Equal(t, DeckNameAliasRejectFewContributors, rejected[0].Reason)
		require.Equal(t, 2, rejected[0].Contributors)
		require.Equal(t, "ドラパルト", rejected[1].Alias)
		require.Equal(t, DeckNameAliasRejectLowSupport, rejected[1].Reason)
		require.Equal(t, 5, rejected[1].Support)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_手動エントリで解決できる名前は生成しない", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)

		expectQueries(mock,
			supplyRows("ロスバレ", "1:0487_origin,2:0225", 12, 4),
			sqlmock.NewRows(deckNameSupplyColumns),
			sqlmock.NewRows(deckNameDemandColumns).AddRow("ロスバレ", 30),
			sqlmock.NewRows(deckNameDemandColumns),
			// 手動で「ロスバレ」が登録済み
			sqlmock.NewRows(deckNameAliasColumns).AddRow("ロスバレ", 1, "0487_origin"),
		)

		candidates, _, err := GenerateDeckNameAliasCandidates(context.Background(), db, cfg())

		require.NoError(t, err)
		require.Empty(t, candidates)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_短すぎるエイリアスは生成しない", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)

		expectQueries(mock,
			// 「サナ」は3文字で下限(4文字)未満
			supplyRows("サナ", "1:0282", 20, 5),
			sqlmock.NewRows(deckNameSupplyColumns),
			sqlmock.NewRows(deckNameDemandColumns).AddRow("サナ", 40),
			sqlmock.NewRows(deckNameDemandColumns),
			sqlmock.NewRows(deckNameAliasColumns),
		)

		candidates, _, err := GenerateDeckNameAliasCandidates(context.Background(), db, cfg())

		require.NoError(t, err)
		require.Empty(t, candidates)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_教師データが無い名前は生成しない", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)

		expectQueries(mock,
			sqlmock.NewRows(deckNameSupplyColumns),
			sqlmock.NewRows(deckNameSupplyColumns),
			sqlmock.NewRows(deckNameDemandColumns).AddRow("謎のデッキ", 100),
			sqlmock.NewRows(deckNameDemandColumns),
			sqlmock.NewRows(deckNameAliasColumns),
		)

		candidates, _, err := GenerateDeckNameAliasCandidates(context.Background(), db, cfg())

		require.NoError(t, err)
		require.Empty(t, candidates)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_需要の多い順に並ぶ", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)

		supply := sqlmock.NewRows(deckNameSupplyColumns)
		for i := 0; i < 12; i++ {
			supply = supply.AddRow("ロストバレット", "user-"+string(rune('a'+i%4)), "1:0487_origin,2:0225")
		}
		for i := 0; i < 12; i++ {
			supply = supply.AddRow("リザードンピジョット", "user-"+string(rune('a'+i%4)), "1:0006,2:0018")
		}

		expectQueries(mock,
			supply,
			sqlmock.NewRows(deckNameSupplyColumns),
			sqlmock.NewRows(deckNameDemandColumns).
				AddRow("ロストバレット", 10).
				AddRow("リザードンピジョット", 80),
			sqlmock.NewRows(deckNameDemandColumns),
			sqlmock.NewRows(deckNameAliasColumns),
		)

		candidates, _, err := GenerateDeckNameAliasCandidates(context.Background(), db, cfg())

		require.NoError(t, err)
		require.Len(t, candidates, 2)
		require.Equal(t, "リザードンピジョット", candidates[0].Alias)
		require.Equal(t, "ロストバレット", candidates[1].Alias)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_教師データ取得のエラーをそのまま返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)

		mock.ExpectQuery(`string_agg\(match_pokemon_sprites`).WillReturnError(sql.ErrConnDone)

		candidates, rejected, err := GenerateDeckNameAliasCandidates(context.Background(), db, cfg())

		require.Error(t, err)
		require.Nil(t, candidates)
		require.Nil(t, rejected)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_落選理由を票の多い順に返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)

		expectQueries(mock,
			// 「ミライドン」は教師データはあるが占有率不足で落ちる(50%)
			func() *sqlmock.Rows {
				rows := sqlmock.NewRows(deckNameSupplyColumns)
				for i := 0; i < 10; i++ {
					rows = rows.AddRow("ミライドン", "user-"+string(rune('a'+i%5)), "1:1008,2:0025")
				}
				for i := 0; i < 10; i++ {
					rows = rows.AddRow("ミライドン", "user-"+string(rune('a'+i%5)), "1:1008,2:0145")
				}
				return rows
			}(),
			sqlmock.NewRows(deckNameSupplyColumns),
			// 需要: 教師データ無し「謎のデッキ」80票 / 占有率不足「ミライドン」50票 / 短すぎ「サナ」20票 /
			//       手動解決済み「リザードン」10票
			sqlmock.NewRows(deckNameDemandColumns).
				AddRow("謎のデッキ", 80).
				AddRow("ミライドン", 50).
				AddRow("サナ", 20).
				AddRow("リザードン", 10),
			sqlmock.NewRows(deckNameDemandColumns),
			// 手動で「リザードン」が登録済み
			sqlmock.NewRows(deckNameAliasColumns).AddRow("リザードン", 1, "0006"),
		)

		candidates, rejected, err := GenerateDeckNameAliasCandidates(context.Background(), db, cfg())

		require.NoError(t, err)
		require.Empty(t, candidates)
		require.Len(t, rejected, 4)

		// 票の多い順に、それぞれ想定どおりの理由で並ぶ。
		// Alias は正規化済みの名前(「謎のデッキ」→「謎ノデッキ」)である点に注意。
		require.Equal(t, "謎ノデッキ", rejected[0].Alias)
		require.Equal(t, DeckNameAliasRejectNoSupply, rejected[0].Reason)
		require.Equal(t, 80, rejected[0].DemandVotes)
		require.Zero(t, rejected[0].TotalSupply)

		require.Equal(t, "ミライドン", rejected[1].Alias)
		require.Equal(t, DeckNameAliasRejectLowRatio, rejected[1].Reason)

		require.Equal(t, "サナ", rejected[2].Alias)
		require.Equal(t, DeckNameAliasRejectTooShort, rejected[2].Reason)

		require.Equal(t, "リザードン", rejected[3].Alias)
		require.Equal(t, DeckNameAliasRejectManualExists, rejected[3].Reason)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReplaceAutoDeckNameAliases(t *testing.T) {
	t.Run("正常系_autoを全削除してから候補を書き込む", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)

		candidates := []*DeckNameAliasCandidate{
			{
				Alias: "ロストバレット",
				Sprites: []DeckNameAliasSprite{
					{PokemonSpriteId: "0487_origin", Position: 1},
					{PokemonSpriteId: "0225", Position: 2},
				},
			},
		}

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "deck_name_aliases" WHERE source = \$1`).
			WithArgs("auto").
			WillReturnResult(sqlmock.NewResult(0, 3))
		mock.ExpectQuery(`SELECT \* FROM "deck_name_aliases"`).
			WillReturnRows(sqlmock.NewRows(deckNameAliasColumns))
		mock.ExpectExec(`INSERT INTO "deck_name_aliases"`).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		saved, err := ReplaceAutoDeckNameAliases(context.Background(), db, candidates)

		require.NoError(t, err)
		require.Equal(t, 2, saved)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_手動エントリと同名の候補は取り込まない", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)

		candidates := []*DeckNameAliasCandidate{
			{Alias: "ロスバレ", Sprites: []DeckNameAliasSprite{{PokemonSpriteId: "0487_origin", Position: 1}}},
		}

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "deck_name_aliases" WHERE source = \$1`).
			WithArgs("auto").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(`SELECT \* FROM "deck_name_aliases"`).
			WillReturnRows(sqlmock.NewRows(deckNameAliasColumns).AddRow("ロスバレ", 1, "0487_origin"))
		mock.ExpectCommit()

		saved, err := ReplaceAutoDeckNameAliases(context.Background(), db, candidates)

		require.NoError(t, err)
		require.Zero(t, saved)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_候補が空でもautoの全削除は行う", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "deck_name_aliases" WHERE source = \$1`).
			WithArgs("auto").
			WillReturnResult(sqlmock.NewResult(0, 5))
		mock.ExpectQuery(`SELECT \* FROM "deck_name_aliases"`).
			WillReturnRows(sqlmock.NewRows(deckNameAliasColumns))
		mock.ExpectCommit()

		saved, err := ReplaceAutoDeckNameAliases(context.Background(), db, nil)

		require.NoError(t, err)
		require.Zero(t, saved)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_書き込みに失敗したらロールバックする", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)

		candidates := []*DeckNameAliasCandidate{
			{Alias: "ロストバレット", Sprites: []DeckNameAliasSprite{{PokemonSpriteId: "0487_origin", Position: 1}}},
		}

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "deck_name_aliases" WHERE source = \$1`).
			WithArgs("auto").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(`SELECT \* FROM "deck_name_aliases"`).
			WillReturnRows(sqlmock.NewRows(deckNameAliasColumns))
		mock.ExpectExec(`INSERT INTO "deck_name_aliases"`).WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		saved, err := ReplaceAutoDeckNameAliases(context.Background(), db, candidates)

		require.Error(t, err)
		require.Zero(t, saved)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
