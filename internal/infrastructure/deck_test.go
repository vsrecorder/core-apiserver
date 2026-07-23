package infrastructure

import (
	"context"
	"database/sql"
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

func setupMock4DeckInfrastructure() (*gorm.DB, sqlmock.Sqlmock, error) {
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

func setup4DeckInfrastructure() (repository.DeckInterface, sqlmock.Sqlmock, error) {
	db, mock, err := setupMock4DeckInfrastructure()

	if err != nil {
		return nil, nil, err
	}

	r := NewDeck(db)

	return r, mock, err
}

// deckJoinDeckCodeColumns は decks と最新の deck_codes を JOIN した結果のカラム
// (model.DeckJoinDeckCode に対応)。
var deckJoinDeckCodeColumns = []string{
	"deck_id",
	"deck_created_at",
	"deck_updated_at",
	"deck_deleted_at",
	"deck_archived_at",
	"deck_user_id",
	"deck_name",
	"deck_private_flg",
	"deck_code_id",
	"deck_code_created_at",
	"deck_code_updated_at",
	"deck_code_deleted_at",
	"deck_code_user_id",
	"deck_code_deck_id",
	"deck_code_code",
	"deck_code_private_code_flg",
	"deck_code_memo",
}

var deckPokemonSpriteColumns = []string{"deck_id", "position", "pokemon_sprite_id"}

// deckJoinDeckCodeQuery は decks と deck_codes を JOIN するクエリにマッチする正規表現を組み立てる。
// SELECT句・JOIN句はGoのソース上の改行やインデントをそのままSQLに含むため、
// 検証したいWHERE以降(絞り込み条件・並び順・件数制限)だけを完全一致で見る。
func deckJoinDeckCodeQuery(tail string) string {
	return `(?s)SELECT.*FROM "decks".*LEFT JOIN.*AS deck_codes ON decks\.id = deck_codes\.deck_id.*` + regexp.QuoteMeta(tail)
}

func TestDeckInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"Find":                 test_DeckInfrastructure_Find,
		"FindAll":              test_DeckInfrastructure_FindAll,
		"FindOnCursor":         test_DeckInfrastructure_FindOnCursor,
		"FindById":             test_DeckInfrastructure_FindById,
		"FindByUserId":         test_DeckInfrastructure_FindByUserId,
		"FindByUserIdOnCursor": test_DeckInfrastructure_FindByUserIdOnCursor,
		"DeleteByUserId":       test_DeckInfrastructure_DeleteByUserId,
		"Save":                 test_DeckInfrastructure_Save,
		"Delete":               test_DeckInfrastructure_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_DeckInfrastructure_Find(t *testing.T) {
	// 公開・未アーカイブのデッキが、最新のデッキコードとスプライトを伴って返る
	t.Run("正常系_公開かつ未アーカイブのデッキをデッキコードとスプライト付きで返す", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()
		limit := 10
		offset := 10

		rows := sqlmock.NewRows(deckJoinDeckCodeColumns).AddRow(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
			datetime,
			datetime,
			gorm.DeletedAt{},
			sql.NullTime{},
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"テストデッキ",
			false,
			"01HD7Y3K8D6FDHMHTZ2GT41TN3",
			datetime,
			datetime,
			gorm.DeletedAt{},
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
			"5dbFbk-uBwjqP-VVk5Vv",
			false,
			"",
		)

		mock.ExpectQuery(deckJoinDeckCodeQuery(
			`WHERE decks.private_flg = false AND decks.archived_at IS NULL AND decks.deleted_at IS NULL ORDER BY decks.created_at DESC LIMIT $1 OFFSET $2`,
		)).WithArgs(
			limit,
			offset,
		).WillReturnRows(rows)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id IN ($1) ORDER BY position ASC`,
		)).WithArgs(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		).WillReturnRows(sqlmock.NewRows(deckPokemonSpriteColumns).AddRow(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2", 1, "pikachu",
		))

		decks, err := r.Find(context.Background(), limit, offset)

		require.NoError(t, err)
		require.Equal(t, 1, len(decks))
		require.Equal(t, "01HD7Y3K8D6FDHMHTZ2GT41TN2", decks[0].ID)
		require.Equal(t, "テストデッキ", decks[0].Name)
		require.Empty(t, decks[0].ArchivedAt)
		require.Equal(t, "5dbFbk-uBwjqP-VVk5Vv", decks[0].LatestDeckCode.Code)
		require.Len(t, decks[0].PokemonSprites, 1)
		require.Equal(t, "pikachu", decks[0].PokemonSprites[0].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 該当デッキが無い場合、スプライトの取得は行わず空のスライスを返す
	t.Run("正常系_該当なしならスプライト取得を行わず空スライスを返す", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		limit := 10
		offset := 10

		mock.ExpectQuery(deckJoinDeckCodeQuery(
			`WHERE decks.private_flg = false AND decks.archived_at IS NULL AND decks.deleted_at IS NULL ORDER BY decks.created_at DESC LIMIT $1 OFFSET $2`,
		)).WithArgs(
			limit,
			offset,
		).WillReturnRows(sqlmock.NewRows(deckJoinDeckCodeColumns))

		decks, err := r.Find(context.Background(), limit, offset)

		require.NoError(t, err)
		require.Equal(t, []*entity.Deck{}, decks)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func test_DeckInfrastructure_FindAll(t *testing.T) {
	// 指定ユーザの未アーカイブのデッキが、非公開のものも含めて全件返る
	t.Run("正常系_指定ユーザの未アーカイブデッキを非公開含め全件返す", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()
		uid := "CeQ0Oa9g9uRThL11lj4l45VAg8p1"

		rows := sqlmock.NewRows(deckJoinDeckCodeColumns).AddRow(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
			datetime,
			datetime,
			gorm.DeletedAt{},
			sql.NullTime{},
			uid,
			"テストデッキ",
			true,
			"",
			time.Time{},
			time.Time{},
			gorm.DeletedAt{},
			"",
			"",
			"",
			false,
			"",
		)

		mock.ExpectQuery(deckJoinDeckCodeQuery(
			`WHERE decks.user_id = $2 AND decks.archived_at IS NULL AND decks.deleted_at IS NULL ORDER BY decks.created_at DESC`,
		)).WithArgs(
			uid,
			uid,
		).WillReturnRows(rows)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id IN ($1) ORDER BY position ASC`,
		)).WithArgs(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		).WillReturnRows(sqlmock.NewRows(deckPokemonSpriteColumns))

		decks, err := r.FindAll(context.Background(), uid)

		require.NoError(t, err)
		require.Equal(t, 1, len(decks))
		require.Equal(t, uid, decks[0].UserId)
		require.True(t, decks[0].PrivateFlg)
		require.Empty(t, decks[0].PokemonSprites)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_該当なしの場合は空スライスを返す", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		uid := "CeQ0Oa9g9uRThL11lj4l45VAg8p1"

		mock.ExpectQuery(deckJoinDeckCodeQuery(
			`WHERE decks.user_id = $2 AND decks.archived_at IS NULL AND decks.deleted_at IS NULL ORDER BY decks.created_at DESC`,
		)).WithArgs(
			uid,
			uid,
		).WillReturnRows(sqlmock.NewRows(deckJoinDeckCodeColumns))

		decks, err := r.FindAll(context.Background(), uid)

		require.NoError(t, err)
		require.Equal(t, []*entity.Deck{}, decks)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func test_DeckInfrastructure_FindOnCursor(t *testing.T) {
	// カーソル(作成日時)より古いデッキのみが返る
	t.Run("正常系_カーソルより古いデッキのみ返す", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		cursor := time.Now().Local()
		datetime := time.Now().Local()
		limit := 10

		rows := sqlmock.NewRows(deckJoinDeckCodeColumns).AddRow(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
			datetime,
			datetime,
			gorm.DeletedAt{},
			sql.NullTime{},
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"テストデッキ",
			false,
			"",
			time.Time{},
			time.Time{},
			gorm.DeletedAt{},
			"",
			"",
			"",
			false,
			"",
		)

		mock.ExpectQuery(deckJoinDeckCodeQuery(
			`WHERE decks.created_at < $1 AND decks.private_flg = false AND decks.archived_at IS NULL AND decks.deleted_at IS NULL ORDER BY decks.created_at DESC LIMIT $2`,
		)).WithArgs(
			cursor,
			limit,
		).WillReturnRows(rows)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id IN ($1) ORDER BY position ASC`,
		)).WithArgs(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		).WillReturnRows(sqlmock.NewRows(deckPokemonSpriteColumns))

		decks, err := r.FindOnCursor(context.Background(), limit, cursor)

		require.NoError(t, err)
		require.Equal(t, 1, len(decks))
		require.Equal(t, "01HD7Y3K8D6FDHMHTZ2GT41TN2", decks[0].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_該当なしの場合は空スライスを返す", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		cursor := time.Now().Local()
		limit := 10

		mock.ExpectQuery(deckJoinDeckCodeQuery(
			`WHERE decks.created_at < $1 AND decks.private_flg = false AND decks.archived_at IS NULL AND decks.deleted_at IS NULL ORDER BY decks.created_at DESC LIMIT $2`,
		)).WithArgs(
			cursor,
			limit,
		).WillReturnRows(sqlmock.NewRows(deckJoinDeckCodeColumns))

		decks, err := r.FindOnCursor(context.Background(), limit, cursor)

		require.NoError(t, err)
		require.Equal(t, []*entity.Deck{}, decks)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func test_DeckInfrastructure_FindById(t *testing.T) {
	t.Run("正常系_指定IDのデッキをデッキコードとスプライト付きで返す", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()
		id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

		// idの存在確認
		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "decks" WHERE id = $1 AND "decks"."deleted_at" IS NULL ORDER BY "decks"."id" LIMIT $2`,
		)).WithArgs(
			id,
			1,
		).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))

		rows := sqlmock.NewRows(deckJoinDeckCodeColumns).AddRow(
			id,
			datetime,
			datetime,
			gorm.DeletedAt{},
			sql.NullTime{},
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"テストデッキ",
			false,
			"01HD7Y3K8D6FDHMHTZ2GT41TN3",
			datetime,
			datetime,
			gorm.DeletedAt{},
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			id,
			"5dbFbk-uBwjqP-VVk5Vv",
			true,
			"メモ",
		)

		mock.ExpectQuery(deckJoinDeckCodeQuery(
			`WHERE decks.id = $2 AND decks.deleted_at IS NULL`,
		)).WithArgs(
			id,
			id,
		).WillReturnRows(rows)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id = $1`,
		)).WithArgs(
			id,
		).WillReturnRows(sqlmock.NewRows(deckPokemonSpriteColumns).AddRow(
			id, 1, "pikachu",
		).AddRow(
			id, 2, "raichu",
		))

		deck, err := r.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, id, deck.ID)
		require.Equal(t, "テストデッキ", deck.Name)
		require.Equal(t, "5dbFbk-uBwjqP-VVk5Vv", deck.LatestDeckCode.Code)
		require.True(t, deck.LatestDeckCode.PrivateCodeFlg)
		require.Equal(t, "メモ", deck.LatestDeckCode.Memo)
		require.Len(t, deck.PokemonSprites, 2)
		require.Equal(t, "pikachu", deck.PokemonSprites[0].ID)
		require.Equal(t, "raichu", deck.PokemonSprites[1].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 存在しないidの場合、gormのエラーはドメインエラーへ変換される
	t.Run("異常系_存在しないIDはErrRecordNotFoundへ変換する", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "decks" WHERE id = $1 AND "decks"."deleted_at" IS NULL ORDER BY "decks"."id" LIMIT $2`,
		)).WithArgs(
			id,
			1,
		).WillReturnRows(sqlmock.NewRows([]string{"id"}))

		deck, err := r.FindById(context.Background(), id)

		require.ErrorIs(t, err, apperror.ErrRecordNotFound)
		require.Nil(t, deck)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func test_DeckInfrastructure_FindByUserId(t *testing.T) {
	uid := "CeQ0Oa9g9uRThL11lj4l45VAg8p1"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	// archivedFlg = false の場合、未アーカイブのデッキが返る
	t.Run("正常系_未アーカイブ指定でarchived_atがNULLの条件になる", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()
		limit := 10
		offset := 10

		rows := sqlmock.NewRows(deckJoinDeckCodeColumns).AddRow(
			id, datetime, datetime, gorm.DeletedAt{}, sql.NullTime{}, uid, "テストデッキ", false,
			"", time.Time{}, time.Time{}, gorm.DeletedAt{}, "", "", "", false, "",
		)

		mock.ExpectQuery(deckJoinDeckCodeQuery(
			`WHERE decks.user_id = $2 AND decks.archived_at IS NULL AND decks.deleted_at IS NULL ORDER BY decks.created_at DESC LIMIT $3 OFFSET $4`,
		)).WithArgs(
			uid,
			uid,
			limit,
			offset,
		).WillReturnRows(rows)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id IN ($1) ORDER BY position ASC`,
		)).WithArgs(id).WillReturnRows(sqlmock.NewRows(deckPokemonSpriteColumns))

		decks, err := r.FindByUserId(context.Background(), uid, false, limit, offset)

		require.NoError(t, err)
		require.Equal(t, 1, len(decks))
		require.Equal(t, uid, decks[0].UserId)
		require.Empty(t, decks[0].ArchivedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// archivedFlg = true の場合、アーカイブ済みのデッキが返る
	t.Run("正常系_アーカイブ済み指定でarchived_atがNOTNULLの条件になる", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()
		archivedAt := sql.NullTime{Time: time.Now().Local(), Valid: true}
		limit := 10
		offset := 10

		rows := sqlmock.NewRows(deckJoinDeckCodeColumns).AddRow(
			id, datetime, datetime, gorm.DeletedAt{}, archivedAt, uid, "テストデッキ", false,
			"", time.Time{}, time.Time{}, gorm.DeletedAt{}, "", "", "", false, "",
		)

		mock.ExpectQuery(deckJoinDeckCodeQuery(
			`WHERE decks.user_id = $2 AND decks.archived_at IS NOT NULL AND decks.deleted_at IS NULL ORDER BY decks.created_at DESC LIMIT $3 OFFSET $4`,
		)).WithArgs(
			uid,
			uid,
			limit,
			offset,
		).WillReturnRows(rows)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id IN ($1) ORDER BY position ASC`,
		)).WithArgs(id).WillReturnRows(sqlmock.NewRows(deckPokemonSpriteColumns))

		decks, err := r.FindByUserId(context.Background(), uid, true, limit, offset)

		require.NoError(t, err)
		require.Equal(t, 1, len(decks))
		require.Equal(t, archivedAt.Time, decks[0].ArchivedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_該当なしの場合は空スライスを返す", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		limit := 10
		offset := 10

		mock.ExpectQuery(deckJoinDeckCodeQuery(
			`WHERE decks.user_id = $2 AND decks.archived_at IS NULL AND decks.deleted_at IS NULL ORDER BY decks.created_at DESC LIMIT $3 OFFSET $4`,
		)).WithArgs(
			uid,
			uid,
			limit,
			offset,
		).WillReturnRows(sqlmock.NewRows(deckJoinDeckCodeColumns))

		decks, err := r.FindByUserId(context.Background(), uid, false, limit, offset)

		require.NoError(t, err)
		require.Equal(t, []*entity.Deck{}, decks)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func test_DeckInfrastructure_FindByUserIdOnCursor(t *testing.T) {
	uid := "CeQ0Oa9g9uRThL11lj4l45VAg8p1"
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	t.Run("正常系_カーソルと未アーカイブ条件で絞り込む", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		cursor := time.Now().Local()
		datetime := time.Now().Local()
		limit := 10

		rows := sqlmock.NewRows(deckJoinDeckCodeColumns).AddRow(
			id, datetime, datetime, gorm.DeletedAt{}, sql.NullTime{}, uid, "テストデッキ", false,
			"", time.Time{}, time.Time{}, gorm.DeletedAt{}, "", "", "", false, "",
		)

		mock.ExpectQuery(deckJoinDeckCodeQuery(
			`WHERE decks.created_at < $2 AND decks.user_id = $3 AND decks.archived_at IS NULL AND decks.deleted_at IS NULL ORDER BY decks.created_at DESC LIMIT $4`,
		)).WithArgs(
			uid,
			cursor,
			uid,
			limit,
		).WillReturnRows(rows)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id IN ($1) ORDER BY position ASC`,
		)).WithArgs(id).WillReturnRows(sqlmock.NewRows(deckPokemonSpriteColumns))

		decks, err := r.FindByUserIdOnCursor(context.Background(), uid, false, limit, cursor)

		require.NoError(t, err)
		require.Equal(t, 1, len(decks))
		require.Equal(t, uid, decks[0].UserId)
		require.Empty(t, decks[0].ArchivedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_カーソルとアーカイブ済み条件で絞り込む", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		cursor := time.Now().Local()
		datetime := time.Now().Local()
		archivedAt := sql.NullTime{Time: time.Now().Local(), Valid: true}
		limit := 10

		rows := sqlmock.NewRows(deckJoinDeckCodeColumns).AddRow(
			id, datetime, datetime, gorm.DeletedAt{}, archivedAt, uid, "テストデッキ", false,
			"", time.Time{}, time.Time{}, gorm.DeletedAt{}, "", "", "", false, "",
		)

		mock.ExpectQuery(deckJoinDeckCodeQuery(
			`WHERE decks.created_at < $2 AND decks.user_id = $3 AND decks.archived_at IS NOT NULL AND decks.deleted_at IS NULL ORDER BY decks.created_at DESC LIMIT $4`,
		)).WithArgs(
			uid,
			cursor,
			uid,
			limit,
		).WillReturnRows(rows)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id IN ($1) ORDER BY position ASC`,
		)).WithArgs(id).WillReturnRows(sqlmock.NewRows(deckPokemonSpriteColumns))

		decks, err := r.FindByUserIdOnCursor(context.Background(), uid, true, limit, cursor)

		require.NoError(t, err)
		require.Equal(t, 1, len(decks))
		require.Equal(t, archivedAt.Time, decks[0].ArchivedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_該当なしの場合は空スライスを返す", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		cursor := time.Now().Local()
		limit := 10

		mock.ExpectQuery(deckJoinDeckCodeQuery(
			`WHERE decks.created_at < $2 AND decks.user_id = $3 AND decks.archived_at IS NULL AND decks.deleted_at IS NULL ORDER BY decks.created_at DESC LIMIT $4`,
		)).WithArgs(
			uid,
			cursor,
			uid,
			limit,
		).WillReturnRows(sqlmock.NewRows(deckJoinDeckCodeColumns))

		decks, err := r.FindByUserIdOnCursor(context.Background(), uid, false, limit, cursor)

		require.NoError(t, err)
		require.Equal(t, []*entity.Deck{}, decks)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func test_DeckInfrastructure_DeleteByUserId(t *testing.T) {
	// 退会時の一括削除。デッキの件数によらずクエリは2文で、
	// アーカイブ済みのデッキも対象になる(archived_at を条件に入れない)。
	t.Run("正常系_デッキコードとデッキをまとめて削除する", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		uid := "CeQ0Oa9g9uRThL11lj4l45VAg8p1"

		mock.ExpectBegin()

		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "deck_codes" SET "deleted_at"=$1 WHERE deck_id IN (SELECT "id" FROM "decks" WHERE user_id = $2 AND "decks"."deleted_at" IS NULL) AND "deck_codes"."deleted_at" IS NULL`,
		)).WithArgs(
			AnyTime{},
			uid,
		).WillReturnResult(sqlmock.NewResult(0, 2))

		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "decks" SET "deleted_at"=$1 WHERE user_id = $2 AND "decks"."deleted_at" IS NULL`,
		)).WithArgs(
			AnyTime{},
			uid,
		).WillReturnResult(sqlmock.NewResult(0, 2))

		mock.ExpectCommit()

		err = r.DeleteByUserId(context.Background(), uid)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func test_DeckInfrastructure_Save(t *testing.T) {
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"
	uid := "CeQ0Oa9g9uRThL11lj4l45VAg8p1"

	// デッキコードを持たない場合、deck_codes への保存は行わない。
	// スプライトは入れ替えのため、一度削除してから保存する。
	t.Run("正常系_デッキコードなしならdecksとスプライト削除のみ保存する", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "decks" SET`)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			sql.NullTime{},
			uid,
			"テストデッキ",
			false,
			id,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(
			`DELETE FROM "deck_pokemon_sprites" WHERE deck_id = $1`,
		)).WithArgs(
			id,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		deck := entity.NewDeck(
			id,
			datetime,
			time.Time{},
			uid,
			"テストデッキ",
			false,
			nil,
			nil,
		)

		require.NoError(t, r.Save(context.Background(), deck))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// アーカイブ日時を持つ場合、archived_at が有効値として保存される
	t.Run("正常系_アーカイブ日時が有効値として保存される", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()
		now := time.Now().Local()
		archivedAt := sql.NullTime{Time: now, Valid: true}

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "decks" SET`)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			archivedAt,
			uid,
			"テストデッキ",
			false,
			id,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(
			`DELETE FROM "deck_pokemon_sprites" WHERE deck_id = $1`,
		)).WithArgs(
			id,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		deck := entity.NewDeck(
			id,
			datetime,
			now,
			uid,
			"テストデッキ",
			false,
			nil,
			nil,
		)

		require.NoError(t, r.Save(context.Background(), deck))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// FK制約を満たすため、decks → deck_pokemon_sprites → deck_codes の順に保存する
	t.Run("正常系_decksスプライトdeck_codesの順に保存する", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()
		deckCodeId := "01HD7Y3K8D6FDHMHTZ2GT41TN3"

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "decks" SET`)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			sql.NullTime{},
			uid,
			"テストデッキ",
			false,
			id,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(
			`DELETE FROM "deck_pokemon_sprites" WHERE deck_id = $1`,
		)).WithArgs(
			id,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "deck_pokemon_sprites" SET "pokemon_sprite_id"=$1 WHERE "deck_id" = $2 AND "position" = $3`,
		)).WithArgs(
			"pikachu",
			id,
			1,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "deck_codes" SET`)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			uid,
			id,
			"5dbFbk-uBwjqP-VVk5Vv",
			false,
			"",
			deckCodeId,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		deck := entity.NewDeck(
			id,
			datetime,
			time.Time{},
			uid,
			"テストデッキ",
			false,
			entity.NewDeckCode(deckCodeId, datetime, uid, id, "5dbFbk-uBwjqP-VVk5Vv", false, ""),
			[]*entity.PokemonSprite{entity.NewPokemonSprite("pikachu")},
		)

		require.NoError(t, r.Save(context.Background(), deck))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 2枠目のみのスプライトは position=2 のまま保存され、1枠目へ詰められない
	t.Run("正常系_position指定のスプライトは指定スロットで保存する", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "decks" SET`)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			sql.NullTime{},
			uid,
			"テストデッキ",
			false,
			id,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(
			`DELETE FROM "deck_pokemon_sprites" WHERE deck_id = $1`,
		)).WithArgs(
			id,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "deck_pokemon_sprites" SET "pokemon_sprite_id"=$1 WHERE "deck_id" = $2 AND "position" = $3`,
		)).WithArgs(
			"raichu",
			id,
			2,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		deck := entity.NewDeck(
			id,
			datetime,
			time.Time{},
			uid,
			"テストデッキ",
			false,
			nil,
			[]*entity.PokemonSprite{entity.NewPokemonSpriteWithPosition("raichu", 2)},
		)

		require.NoError(t, r.Save(context.Background(), deck))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// 読み取りでは DB の position が entity へ引き継がれ、2枠目のみでも枠が保たれる
	t.Run("正常系_スプライトのpositionが読み取りでentityへ引き継がれる", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()
		id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

		// idの存在確認
		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "decks" WHERE id = $1 AND "decks"."deleted_at" IS NULL ORDER BY "decks"."id" LIMIT $2`,
		)).WithArgs(
			id,
			1,
		).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))

		rows := sqlmock.NewRows(deckJoinDeckCodeColumns).AddRow(
			id,
			datetime,
			datetime,
			gorm.DeletedAt{},
			sql.NullTime{},
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"テストデッキ",
			false,
			"01HD7Y3K8D6FDHMHTZ2GT41TN3",
			datetime,
			datetime,
			gorm.DeletedAt{},
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			id,
			"5dbFbk-uBwjqP-VVk5Vv",
			true,
			"メモ",
		)

		mock.ExpectQuery(deckJoinDeckCodeQuery(
			`WHERE decks.id = $2 AND decks.deleted_at IS NULL`,
		)).WithArgs(
			id,
			id,
		).WillReturnRows(rows)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "deck_pokemon_sprites" WHERE deck_id = $1`,
		)).WithArgs(
			id,
		).WillReturnRows(sqlmock.NewRows(deckPokemonSpriteColumns).AddRow(
			id, 2, "raichu",
		))

		deck, err := r.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Len(t, deck.PokemonSprites, 1)
		require.Equal(t, "raichu", deck.PokemonSprites[0].ID)
		require.Equal(t, uint(2), deck.PokemonSprites[0].Position)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// デッキコードのIDが空の場合(=デッキコード未登録)、deck_codes への保存は行わない
	t.Run("正常系_デッキコードID空ならdeck_codesへは保存しない", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "decks" SET`)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			sql.NullTime{},
			uid,
			"テストデッキ",
			false,
			id,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(
			`DELETE FROM "deck_pokemon_sprites" WHERE deck_id = $1`,
		)).WithArgs(
			id,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		deck := entity.NewDeck(
			id,
			datetime,
			time.Time{},
			uid,
			"テストデッキ",
			false,
			entity.NewDeckCode("", time.Time{}, "", "", "", false, ""),
			nil,
		)

		require.NoError(t, r.Save(context.Background(), deck))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// decks の保存に失敗した場合はロールバックされる
	t.Run("異常系_decks保存失敗時はロールバックする", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		datetime := time.Now().Local()

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "decks" SET`)).WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		deck := entity.NewDeck(id, datetime, time.Time{}, uid, "テストデッキ", false, nil, nil)

		require.Error(t, r.Save(context.Background(), deck))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func test_DeckInfrastructure_Delete(t *testing.T) {
	id := "01HD7Y3K8D6FDHMHTZ2GT41TN2"

	// デッキに紐づくデッキコードも併せて削除される
	t.Run("正常系_紐づくデッキコードも併せて論理削除する", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		deckCodeId := "01HD7Y3K8D6FDHMHTZ2GT41TN3"

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "deck_codes" WHERE deck_id = $1 AND "deck_codes"."deleted_at" IS NULL ORDER BY created_at ASC`,
		)).WithArgs(
			id,
		).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(deckCodeId))

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "deck_codes" SET "deleted_at"=$1 WHERE id = $2 AND "deck_codes"."deleted_at" IS NULL`,
		)).WithArgs(
			AnyTime{},
			deckCodeId,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "decks" SET "deleted_at"=$1 WHERE id = $2 AND "decks"."deleted_at" IS NULL`,
		)).WithArgs(
			AnyTime{},
			id,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		require.NoError(t, r.Delete(context.Background(), id))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// デッキコードが無い場合はデッキのみ削除される
	t.Run("正常系_デッキコードがなければデッキのみ論理削除する", func(t *testing.T) {
		r, mock, err := setup4DeckInfrastructure()
		require.NoError(t, err)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "deck_codes" WHERE deck_id = $1 AND "deck_codes"."deleted_at" IS NULL ORDER BY created_at ASC`,
		)).WithArgs(
			id,
		).WillReturnRows(sqlmock.NewRows([]string{"id"}))

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "decks" SET "deleted_at"=$1 WHERE id = $2 AND "decks"."deleted_at" IS NULL`,
		)).WithArgs(
			AnyTime{},
			id,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		require.NoError(t, r.Delete(context.Background(), id))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
