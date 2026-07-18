package infrastructure

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

func setupMock4MatchInfrastructure() (*gorm.DB, sqlmock.Sqlmock, error) {
	mockDB, mock, err := sqlmock.New()

	if err != nil {
		return nil, nil, err
	}

	db, err := gorm.Open(
		postgres.New(postgres.Config{
			Conn: mockDB,
		}),
		&gorm.Config{},
	)

	return db, mock, err
}

func setup4MatchInfrastructure() (repository.MatchInterface, sqlmock.Sqlmock, error) {
	db, mock, err := setupMock4MatchInfrastructure()

	if err != nil {
		return nil, nil, err
	}

	r := NewMatch(db)

	return r, mock, err
}

// matchJoinGameColumns は matches と games を JOIN した結果のカラム
// (model.MatchJoinGame に対応)。
var matchJoinGameColumns = []string{
	"match_id",
	"match_created_at",
	"match_updated_at",
	"match_deleted_at",
	"match_record_id",
	"match_deck_id",
	"match_deck_code_id",
	"match_user_id",
	"match_opponents_user_id",
	"match_bo3_flg",
	"match_group_match_flg",
	"match_qualifying_round_flg",
	"match_final_tournament_flg",
	"match_default_victory_flg",
	"match_default_defeat_flg",
	"match_victory_flg",
	"match_group_match_victory_flg",
	"match_opponents_deck_info",
	"match_memo",
	"match_position",
	"game_id",
	"game_created_at",
	"game_updated_at",
	"game_deleted_at",
	"game_match_id",
	"game_user_id",
	"game_go_first",
	"game_winning_flg",
	"game_your_prize_cards",
	"game_opponents_prize_cards",
	"game_memo",
}

var matchPokemonSpriteColumns = []string{"match_id", "position", "pokemon_sprite_id"}

// matchJoinGameQuery は matches と games を JOIN するクエリにマッチする正規表現を組み立てる。
// SELECT句・JOIN句はGoのソース上の改行やインデントをそのままSQLに含むため、
// 検証したいWHERE以降(絞り込み条件・並び順)だけを完全一致で見る。
func matchJoinGameQuery(from string, tail string) string {
	return `(?s)SELECT.*FROM "` + from + `".*LEFT JOIN games.*games\.deleted_at IS NULL.*` + regexp.QuoteMeta(tail)
}

// matchJoinGameRow は1行分の値を組み立てる。gameIdが空の場合は不戦勝/不戦敗
// (=対局が存在しない)を表す行になる。
type matchJoinGameRow struct {
	matchId    string
	position   int
	victoryFlg bool
	gameId     string
	goFirst    bool
	winningFlg bool
}

func addMatchJoinGameRow(rows *sqlmock.Rows, datetime time.Time, r matchJoinGameRow) *sqlmock.Rows {
	return rows.AddRow(
		r.matchId,
		datetime,
		datetime,
		gorm.DeletedAt{},
		"01HD7Y3K8D6FDHMHTZ2GT41TR1",
		"01HD7Y3K8D6FDHMHTZ2GT41TD1",
		"01HD7Y3K8D6FDHMHTZ2GT41TC1",
		"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
		"",
		false,
		false,
		false,
		false,
		false,
		false,
		r.victoryFlg,
		false,
		"対戦相手のデッキ情報",
		"メモ",
		r.position,
		r.gameId,
		datetime,
		datetime,
		gorm.DeletedAt{},
		r.matchId,
		"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
		r.goFirst,
		r.winningFlg,
		uint(0),
		uint(6),
		"",
	)
}

func TestMatchInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"FindById":       test_MatchInfrastructure_FindById,
		"FindByRecordId": test_MatchInfrastructure_FindByRecordId,
		"FindByUserId":   test_MatchInfrastructure_FindByUserId,
		"FindLatest":     test_MatchInfrastructure_FindLatest,
		"Create":         test_MatchInfrastructure_Create,
		"Update":         test_MatchInfrastructure_Update,
		"Delete":         test_MatchInfrastructure_Delete,
		"Reorder":        test_MatchInfrastructure_Reorder,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_MatchInfrastructure_FindById(t *testing.T) {
	matchId := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	// 対局が複数ある場合、1つのMatchにまとめられる
	t.Run("正常系_複数の対局が1つのMatchにまとめられる", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()

		rows := sqlmock.NewRows(matchJoinGameColumns)
		rows = addMatchJoinGameRow(rows, datetime, matchJoinGameRow{
			matchId: matchId, position: 1, victoryFlg: true,
			gameId: "01HD7Y3K8D6FDHMHTZ2GT41TG1", goFirst: true, winningFlg: true,
		})
		rows = addMatchJoinGameRow(rows, datetime, matchJoinGameRow{
			matchId: matchId, position: 1, victoryFlg: true,
			gameId: "01HD7Y3K8D6FDHMHTZ2GT41TG2", goFirst: false, winningFlg: true,
		})

		mock.ExpectQuery(matchJoinGameQuery("matches",
			`WHERE matches.id = $1 AND matches.deleted_at IS NULL ORDER BY games.created_at ASC`,
		)).WithArgs(matchId).WillReturnRows(rows)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "match_pokemon_sprites" WHERE match_id = $1`,
		)).WithArgs(matchId).WillReturnRows(
			sqlmock.NewRows(matchPokemonSpriteColumns).AddRow(matchId, 1, "pikachu"),
		)

		match, err := r.FindById(context.Background(), matchId)

		require.NoError(t, err)
		require.Equal(t, matchId, match.ID)
		require.Equal(t, 1, match.Position)
		require.True(t, match.VictoryFlg)
		require.Equal(t, "対戦相手のデッキ情報", match.OpponentsDeckInfo)
		require.Len(t, match.Games, 2)
		require.Equal(t, "01HD7Y3K8D6FDHMHTZ2GT41TG1", match.Games[0].ID)
		require.True(t, match.Games[0].GoFirst)
		require.Equal(t, "01HD7Y3K8D6FDHMHTZ2GT41TG2", match.Games[1].ID)
		require.False(t, match.Games[1].GoFirst)
		require.Len(t, match.PokemonSprites, 1)
		require.Equal(t, "pikachu", match.PokemonSprites[0].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 不戦勝/不戦敗の場合、対局が存在しないためgamesは空になる
	t.Run("正常系_不戦勝や不戦敗ならGamesは空になる", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()

		rows := addMatchJoinGameRow(sqlmock.NewRows(matchJoinGameColumns), datetime, matchJoinGameRow{
			matchId: matchId, position: 1, victoryFlg: true, gameId: "",
		})

		mock.ExpectQuery(matchJoinGameQuery("matches",
			`WHERE matches.id = $1 AND matches.deleted_at IS NULL ORDER BY games.created_at ASC`,
		)).WithArgs(matchId).WillReturnRows(rows)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "match_pokemon_sprites" WHERE match_id = $1`,
		)).WithArgs(matchId).WillReturnRows(sqlmock.NewRows(matchPokemonSpriteColumns))

		match, err := r.FindById(context.Background(), matchId)

		require.NoError(t, err)
		require.Equal(t, matchId, match.ID)
		require.Empty(t, match.Games)
		require.Empty(t, match.PokemonSprites)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_存在しないIDはErrRecordNotFoundへ変換する", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		mock.ExpectQuery(matchJoinGameQuery("matches",
			`WHERE matches.id = $1 AND matches.deleted_at IS NULL ORDER BY games.created_at ASC`,
		)).WithArgs(matchId).WillReturnRows(sqlmock.NewRows(matchJoinGameColumns))

		match, err := r.FindById(context.Background(), matchId)

		require.Equal(t, apperror.ErrRecordNotFound, err)
		require.Nil(t, match)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func test_MatchInfrastructure_FindByRecordId(t *testing.T) {
	recordId := "01HD7Y3K8D6FDHMHTZ2GT41TR1"
	matchId1 := "01HD7Y3K8D6FDHMHTZ2GT41TN1"
	matchId2 := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	// 複数のMatchがposition順に返り、同一Matchの対局はまとめられる
	t.Run("正常系_複数Matchがposition順に返り対局がまとめられる", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()

		rows := sqlmock.NewRows(matchJoinGameColumns)
		rows = addMatchJoinGameRow(rows, datetime, matchJoinGameRow{
			matchId: matchId1, position: 1, victoryFlg: true,
			gameId: "01HD7Y3K8D6FDHMHTZ2GT41TG1", goFirst: true, winningFlg: true,
		})
		rows = addMatchJoinGameRow(rows, datetime, matchJoinGameRow{
			matchId: matchId1, position: 1, victoryFlg: true,
			gameId: "01HD7Y3K8D6FDHMHTZ2GT41TG2", goFirst: false, winningFlg: true,
		})
		rows = addMatchJoinGameRow(rows, datetime, matchJoinGameRow{
			matchId: matchId2, position: 2, victoryFlg: false,
			gameId: "01HD7Y3K8D6FDHMHTZ2GT41TG3", goFirst: true, winningFlg: false,
		})

		mock.ExpectQuery(`(?s)SELECT.*FROM "records".*INNER JOIN matches.*LEFT JOIN games.*` + regexp.QuoteMeta(
			`WHERE records.id = $1 AND records.deleted_at IS NULL AND matches.deleted_at IS NULL ORDER BY matches.position ASC, games.created_at ASC`,
		)).WithArgs(recordId).WillReturnRows(rows)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "match_pokemon_sprites" WHERE match_id IN ($1,$2) ORDER BY position ASC`,
		)).WithArgs(matchId1, matchId2).WillReturnRows(
			sqlmock.NewRows(matchPokemonSpriteColumns).AddRow(matchId2, 1, "raichu"),
		)

		matches, err := r.FindByRecordId(context.Background(), recordId)

		require.NoError(t, err)
		require.Len(t, matches, 2)
		require.Equal(t, matchId1, matches[0].ID)
		require.Len(t, matches[0].Games, 2)
		require.Empty(t, matches[0].PokemonSprites)
		require.Equal(t, matchId2, matches[1].ID)
		require.Len(t, matches[1].Games, 1)
		require.Len(t, matches[1].PokemonSprites, 1)
		require.Equal(t, "raichu", matches[1].PokemonSprites[0].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_該当なしはErrRecordNotFoundを返す", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		mock.ExpectQuery(`(?s)SELECT.*FROM "records".*INNER JOIN matches.*LEFT JOIN games.*`).
			WithArgs(recordId).WillReturnRows(sqlmock.NewRows(matchJoinGameColumns))

		matches, err := r.FindByRecordId(context.Background(), recordId)

		require.Equal(t, apperror.ErrRecordNotFound, err)
		require.Nil(t, matches)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func test_MatchInfrastructure_FindByUserId(t *testing.T) {
	uid := "CeQ0Oa9g9uRThL11lj4l45VAg8p1"
	matchId1 := "01HD7Y3K8D6FDHMHTZ2GT41TN1"
	matchId2 := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	// 副問い合わせで対象ユーザの最新Matchを絞り込んだ上で、作成日時の降順に返る
	t.Run("正常系_指定ユーザの最新Matchを作成日時降順で返す", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()
		limit := 10

		rows := sqlmock.NewRows(matchJoinGameColumns)
		rows = addMatchJoinGameRow(rows, datetime, matchJoinGameRow{
			matchId: matchId2, position: 2, victoryFlg: true,
			gameId: "01HD7Y3K8D6FDHMHTZ2GT41TG3", goFirst: true, winningFlg: true,
		})
		rows = addMatchJoinGameRow(rows, datetime, matchJoinGameRow{
			matchId: matchId1, position: 1, victoryFlg: false,
			gameId: "01HD7Y3K8D6FDHMHTZ2GT41TG1", goFirst: false, winningFlg: false,
		})

		mock.ExpectQuery(matchJoinGameQuery("matches",
			`WHERE matches.id IN (SELECT id FROM "matches" WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2) ORDER BY matches.created_at DESC, games.created_at ASC`,
		)).WithArgs(uid, limit).WillReturnRows(rows)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "match_pokemon_sprites" WHERE match_id IN ($1,$2) ORDER BY position ASC`,
		)).WithArgs(matchId2, matchId1).WillReturnRows(sqlmock.NewRows(matchPokemonSpriteColumns))

		matches, err := r.FindByUserId(context.Background(), uid, limit)

		require.NoError(t, err)
		require.Len(t, matches, 2)
		require.Equal(t, matchId2, matches[0].ID)
		require.Equal(t, matchId1, matches[1].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_該当なしはErrRecordNotFoundを返す", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		limit := 10

		mock.ExpectQuery(matchJoinGameQuery("matches", ``)).
			WithArgs(uid, limit).WillReturnRows(sqlmock.NewRows(matchJoinGameColumns))

		matches, err := r.FindByUserId(context.Background(), uid, limit)

		require.Equal(t, apperror.ErrRecordNotFound, err)
		require.Nil(t, matches)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func test_MatchInfrastructure_FindLatest(t *testing.T) {
	matchId := "01HD7Y3K8D6FDHMHTZ2GT41TN1"

	// ユーザを問わず最新のMatchが返る
	t.Run("正常系_全ユーザの最新Matchを返す", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()
		limit := 10

		rows := addMatchJoinGameRow(sqlmock.NewRows(matchJoinGameColumns), datetime, matchJoinGameRow{
			matchId: matchId, position: 1, victoryFlg: true,
			gameId: "01HD7Y3K8D6FDHMHTZ2GT41TG1", goFirst: true, winningFlg: true,
		})

		mock.ExpectQuery(matchJoinGameQuery("matches",
			`WHERE matches.id IN (SELECT id FROM "matches" WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1) ORDER BY matches.created_at DESC, games.created_at ASC`,
		)).WithArgs(limit).WillReturnRows(rows)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "match_pokemon_sprites" WHERE match_id IN ($1) ORDER BY position ASC`,
		)).WithArgs(matchId).WillReturnRows(sqlmock.NewRows(matchPokemonSpriteColumns))

		matches, err := r.FindLatest(context.Background(), limit)

		require.NoError(t, err)
		require.Len(t, matches, 1)
		require.Equal(t, matchId, matches[0].ID)
		require.Len(t, matches[0].Games, 1)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_該当なしはErrRecordNotFoundを返す", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		limit := 10

		mock.ExpectQuery(matchJoinGameQuery("matches", ``)).
			WithArgs(limit).WillReturnRows(sqlmock.NewRows(matchJoinGameColumns))

		matches, err := r.FindLatest(context.Background(), limit)

		require.Equal(t, apperror.ErrRecordNotFound, err)
		require.Nil(t, matches)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// matchUpdateArgs は newTestMatch が保存される際の UPDATE "matches" の実引数を、
// model.Match のフィールド順(=GORMがSET句を組み立てる順)で返す。
// 末尾のpositionとidが、採番結果の検証対象になる。
func matchUpdateArgs(datetime time.Time, position int) []driver.Value {
	return []driver.Value{
		datetime,
		AnyTime{},
		gorm.DeletedAt{},
		"01HD7Y3K8D6FDHMHTZ2GT41TR1",
		"01HD7Y3K8D6FDHMHTZ2GT41TD1",
		"01HD7Y3K8D6FDHMHTZ2GT41TC1",
		"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
		"",
		false,
		false,
		false,
		false,
		false,
		false,
		true,
		false,
		"対戦相手のデッキ情報",
		"メモ",
		position,
		"01HD7Y3K8D6FDHMHTZ2GT41TN1",
	}
}

func newTestMatch(matchId string, datetime time.Time, games []*entity.Game, sprites []*entity.PokemonSprite) *entity.Match {
	return entity.NewMatch(
		matchId,
		datetime,
		"01HD7Y3K8D6FDHMHTZ2GT41TR1",
		"01HD7Y3K8D6FDHMHTZ2GT41TD1",
		"01HD7Y3K8D6FDHMHTZ2GT41TC1",
		"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
		"",
		false,
		false,
		false,
		false,
		false,
		false,
		true,
		false,
		"対戦相手のデッキ情報",
		"メモ",
		games,
		sprites,
	)
}

func test_MatchInfrastructure_Create(t *testing.T) {
	matchId := "01HD7Y3K8D6FDHMHTZ2GT41TN1"
	recordId := "01HD7Y3K8D6FDHMHTZ2GT41TR1"

	// 同一record内の最大position+1が採番され、match・game・スプライトが保存される
	t.Run("正常系_最大position加算で採番しゲームとスプライトも保存する", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT MAX(position) FROM "matches" WHERE (record_id = $1 AND deleted_at IS NULL) AND "matches"."deleted_at" IS NULL`,
		)).WithArgs(recordId).WillReturnRows(
			sqlmock.NewRows([]string{"max"}).AddRow(2),
		)

		mock.ExpectBegin()
		// 既存の最大positionが2のため、3が採番される
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "matches" SET`)).
			WithArgs(matchUpdateArgs(datetime, 3)...).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "games" SET`)).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "match_pokemon_sprites" SET "pokemon_sprite_id"=$1 WHERE "match_id" = $2 AND "position" = $3`,
		)).WithArgs("pikachu", matchId, 1).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		game := entity.NewGame("01HD7Y3K8D6FDHMHTZ2GT41TG1", datetime, matchId, "CeQ0Oa9g9uRThL11lj4l45VAg8p1", true, true, 0, 6, "")
		match := newTestMatch(matchId, datetime, []*entity.Game{game}, []*entity.PokemonSprite{entity.NewPokemonSprite("pikachu")})

		require.NoError(t, r.Create(context.Background(), match))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// record内に既存のmatchが無い場合、positionは1から始まる
	t.Run("正常系_既存Matchがなければpositionは1から始まる", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT MAX(position) FROM "matches" WHERE (record_id = $1 AND deleted_at IS NULL) AND "matches"."deleted_at" IS NULL`,
		)).WithArgs(recordId).WillReturnRows(
			sqlmock.NewRows([]string{"max"}).AddRow(nil),
		)

		mock.ExpectBegin()
		// MAX(position)がNULL(=既存のmatchが無い)のため、1から始まる
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "matches" SET`)).
			WithArgs(matchUpdateArgs(datetime, 1)...).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		match := newTestMatch(matchId, datetime, nil, nil)

		require.NoError(t, r.Create(context.Background(), match))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// position採番に失敗した場合はトランザクションを開始せずに終了する
	t.Run("異常系_position採番失敗時はトランザクションを開始しない", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT MAX(position) FROM "matches" WHERE (record_id = $1 AND deleted_at IS NULL) AND "matches"."deleted_at" IS NULL`,
		)).WithArgs(recordId).WillReturnError(sql.ErrConnDone)

		match := newTestMatch(matchId, datetime, nil, nil)

		require.Error(t, r.Create(context.Background(), match))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func test_MatchInfrastructure_Update(t *testing.T) {
	matchId := "01HD7Y3K8D6FDHMHTZ2GT41TN1"
	uid := "CeQ0Oa9g9uRThL11lj4l45VAg8p1"

	// 既存と同数の対局を更新する場合、既存のGameのID・作成日時を維持して上書きする
	t.Run("正常系_同数の対局は既存GameのIDと作成日時を維持して上書きする", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "games" WHERE match_id = $1 AND "games"."deleted_at" IS NULL ORDER BY created_at ASC`,
		)).WithArgs(matchId).WillReturnRows(
			sqlmock.NewRows([]string{"id", "created_at", "match_id", "user_id"}).
				AddRow("01HD7Y3K8D6FDHMHTZ2GT41TG1", datetime, matchId, uid),
		)

		mock.ExpectBegin()
		// positionはReorderでのみ変更されるため、通常の更新では現在値がそのまま保存される
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "matches" SET`)).
			WithArgs(matchUpdateArgs(datetime, 0)...).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(
			`DELETE FROM "match_pokemon_sprites" WHERE match_id = $1`,
		)).WithArgs(matchId).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "match_pokemon_sprites" SET "pokemon_sprite_id"=$1 WHERE "match_id" = $2 AND "position" = $3`,
		)).WithArgs("pikachu", matchId, 1).WillReturnResult(sqlmock.NewResult(0, 1))
		// 既存のGameのIDで上書きされる(リクエストで渡されたIDは使われない)
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "games" SET`)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			matchId,
			uid,
			false,
			false,
			uint(3),
			uint(6),
			"更新後のメモ",
			"01HD7Y3K8D6FDHMHTZ2GT41TG1",
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		game := entity.NewGame("01HD7Y3K8D6FDHMHTZ2GT41TGX", datetime, matchId, uid, false, false, 3, 6, "更新後のメモ")
		match := newTestMatch(matchId, datetime, []*entity.Game{game}, []*entity.PokemonSprite{entity.NewPokemonSprite("pikachu")})

		require.NoError(t, r.Update(context.Background(), match))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 対局が増えた場合、既存分は上書きし、超過分は新規に追加する
	t.Run("正常系_対局が増えた分は新規Gameとして追加する", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "games" WHERE match_id = $1 AND "games"."deleted_at" IS NULL ORDER BY created_at ASC`,
		)).WithArgs(matchId).WillReturnRows(
			sqlmock.NewRows([]string{"id", "created_at", "match_id", "user_id"}).
				AddRow("01HD7Y3K8D6FDHMHTZ2GT41TG1", datetime, matchId, uid),
		)

		mock.ExpectBegin()
		// positionはReorderでのみ変更されるため、通常の更新では現在値がそのまま保存される
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "matches" SET`)).
			WithArgs(matchUpdateArgs(datetime, 0)...).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(
			`DELETE FROM "match_pokemon_sprites" WHERE match_id = $1`,
		)).WithArgs(matchId).WillReturnResult(sqlmock.NewResult(0, 1))
		// 1件目は既存のGameを上書き
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "games" SET`)).WithArgs(
			datetime, AnyTime{}, gorm.DeletedAt{}, matchId, uid, true, true, uint(0), uint(6), "",
			"01HD7Y3K8D6FDHMHTZ2GT41TG1",
		).WillReturnResult(sqlmock.NewResult(0, 1))
		// 2件目は新規追加
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "games" SET`)).WithArgs(
			datetime, AnyTime{}, gorm.DeletedAt{}, matchId, uid, false, true, uint(0), uint(6), "",
			"01HD7Y3K8D6FDHMHTZ2GT41TG2",
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		games := []*entity.Game{
			entity.NewGame("01HD7Y3K8D6FDHMHTZ2GT41TG1", datetime, matchId, uid, true, true, 0, 6, ""),
			entity.NewGame("01HD7Y3K8D6FDHMHTZ2GT41TG2", datetime, matchId, uid, false, true, 0, 6, ""),
		}
		match := newTestMatch(matchId, datetime, games, nil)

		require.NoError(t, r.Update(context.Background(), match))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 対局が減った場合、余った既存のGameは削除される
	t.Run("正常系_対局が減った分の既存Gameは削除する", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "games" WHERE match_id = $1 AND "games"."deleted_at" IS NULL ORDER BY created_at ASC`,
		)).WithArgs(matchId).WillReturnRows(
			sqlmock.NewRows([]string{"id", "created_at", "match_id", "user_id"}).
				AddRow("01HD7Y3K8D6FDHMHTZ2GT41TG1", datetime, matchId, uid).
				AddRow("01HD7Y3K8D6FDHMHTZ2GT41TG2", datetime, matchId, uid),
		)

		mock.ExpectBegin()
		// positionはReorderでのみ変更されるため、通常の更新では現在値がそのまま保存される
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "matches" SET`)).
			WithArgs(matchUpdateArgs(datetime, 0)...).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(
			`DELETE FROM "match_pokemon_sprites" WHERE match_id = $1`,
		)).WithArgs(matchId).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "games" SET`)).WithArgs(
			datetime, AnyTime{}, gorm.DeletedAt{}, matchId, uid, true, true, uint(0), uint(6), "",
			"01HD7Y3K8D6FDHMHTZ2GT41TG1",
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "games" SET "deleted_at"=$1 WHERE id = $2 AND "games"."deleted_at" IS NULL`,
		)).WithArgs(
			AnyTime{},
			"01HD7Y3K8D6FDHMHTZ2GT41TG2",
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		games := []*entity.Game{
			entity.NewGame("01HD7Y3K8D6FDHMHTZ2GT41TG1", datetime, matchId, uid, true, true, 0, 6, ""),
		}
		match := newTestMatch(matchId, datetime, games, nil)

		require.NoError(t, r.Update(context.Background(), match))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_既存Game取得失敗時はエラーを返す", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "games" WHERE match_id = $1 AND "games"."deleted_at" IS NULL ORDER BY created_at ASC`,
		)).WithArgs(matchId).WillReturnError(sql.ErrConnDone)

		match := newTestMatch(matchId, datetime, nil, nil)

		require.Error(t, r.Update(context.Background(), match))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func test_MatchInfrastructure_Delete(t *testing.T) {
	matchId := "01HD7Y3K8D6FDHMHTZ2GT41TN1"

	// 紐づくgameも併せて論理削除される
	t.Run("正常系_紐づくGameも併せて論理削除する", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "games" SET "deleted_at"=$1 WHERE match_id = $2 AND "games"."deleted_at" IS NULL`,
		)).WithArgs(
			AnyTime{},
			matchId,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "matches" SET "deleted_at"=$1 WHERE id = $2 AND "matches"."deleted_at" IS NULL`,
		)).WithArgs(
			AnyTime{},
			matchId,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		require.NoError(t, r.Delete(context.Background(), matchId))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// gameの削除に失敗した場合、matchは削除されずロールバックされる
	t.Run("異常系_Game削除失敗時はロールバックする", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "games" SET "deleted_at"=$1`)).WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		require.Error(t, r.Delete(context.Background(), matchId))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func test_MatchInfrastructure_Reorder(t *testing.T) {
	recordId := "01HD7Y3K8D6FDHMHTZ2GT41TR1"
	matchId1 := "01HD7Y3K8D6FDHMHTZ2GT41TN1"
	matchId2 := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	// 指定された順に position が 0 から振り直される
	t.Run("正常系_指定順にpositionを0から振り直す", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT count(*) FROM "matches" WHERE (record_id = $1 AND deleted_at IS NULL) AND "matches"."deleted_at" IS NULL`,
		)).WithArgs(recordId).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "matches" SET`)).WithArgs(
			false,
			0,
			true,
			AnyTime{},
			matchId2,
			recordId,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "matches" SET`)).WithArgs(
			true,
			1,
			false,
			AnyTime{},
			matchId1,
			recordId,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		orders := []*entity.MatchOrder{
			{ID: matchId2, QualifyingRoundFlg: true, FinalTournamentFlg: false},
			{ID: matchId1, QualifyingRoundFlg: false, FinalTournamentFlg: true},
		}

		require.NoError(t, r.Reorder(context.Background(), recordId, orders))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// リクエストの件数がrecord内のmatch数と一致しない場合は不正な並び順として扱う
	t.Run("異常系_件数がrecord内のMatch数と不一致ならErrInvalidMatchOrder", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT count(*) FROM "matches" WHERE (record_id = $1 AND deleted_at IS NULL) AND "matches"."deleted_at" IS NULL`,
		)).WithArgs(recordId).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectRollback()

		orders := []*entity.MatchOrder{
			{ID: matchId1},
		}

		require.Equal(t, apperror.ErrInvalidMatchOrder, r.Reorder(context.Background(), recordId, orders))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 他のrecordのmatchが混ざっている場合など、更新対象が存在しない場合も不正として扱う
	t.Run("異常系_更新対象が存在しない場合もErrInvalidMatchOrder", func(t *testing.T) {
		r, mock, err := setup4MatchInfrastructure()
		require.NoError(t, err)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT count(*) FROM "matches" WHERE (record_id = $1 AND deleted_at IS NULL) AND "matches"."deleted_at" IS NULL`,
		)).WithArgs(recordId).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "matches" SET`)).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		orders := []*entity.MatchOrder{
			{ID: matchId1},
		}

		require.Equal(t, apperror.ErrInvalidMatchOrder, r.Reorder(context.Background(), recordId, orders))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
