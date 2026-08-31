package infrastructure

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
)

// shopColumns は shops に prefectures を JOIN した結果のカラム(model.Shop に対応)。
var shopColumns = []string{
	"id",
	"name",
	"zip_code",
	"prefecture_id",
	"prefecture_name",
	"address",
	"tel",
	"business_hours",
	"url",
}

// shopQuery は店舗を引くクエリにマッチする正規表現を組み立てる。
// SELECT句はGoのソース上の改行をそのままSQLに含むため、WHERE以降だけを完全一致で見る。
func shopQuery(tail string) string {
	return `(?s)SELECT.*FROM "shops".*LEFT JOIN prefectures ON prefectures\.id = shops\.prefecture_id.*` +
		regexp.QuoteMeta(tail)
}

func shopRows() *sqlmock.Rows {
	return sqlmock.NewRows(shopColumns).AddRow(
		uint(10317),
		"古本市場新琴似店",
		"001-0907",
		uint(1),
		"北海道",
		"札幌市北区新琴似七条１７−１",
		"011-000-0000",
		"10:00~20:00",
		"https://example.com",
	)
}

func TestShopInfrastructure(t *testing.T) {
	t.Run("Find", func(t *testing.T) {
		t.Run("正常系_キーワードは店舗名と住所の部分一致で絞り込む", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewShop(db)

			mock.ExpectQuery(shopQuery(
				`WHERE shops.id > 0 AND ((shops.name ILIKE $1 OR shops.address ILIKE $2)) `+
					`ORDER BY shops.prefecture_id ASC, shops.name ASC, shops.id ASC LIMIT $3`,
			)).WithArgs("%古本市場%", "%古本市場%", 50).WillReturnRows(shopRows())

			shops, err := r.Find(context.Background(), "古本市場", 50)

			require.NoError(t, err)
			require.Len(t, shops, 1)
			require.Equal(t, uint(10317), shops[0].ID)
			require.Equal(t, "北海道", shops[0].PrefectureName)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("正常系_LIKEのワイルドカードはエスケープして渡す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewShop(db)

			// "%" をそのまま渡すと全件一致になるため、パターンとしての意味を消す
			mock.ExpectQuery(shopQuery(`WHERE shops.id > 0 AND ((shops.name ILIKE $1 OR shops.address ILIKE $2))`)).
				WithArgs(`%\%%`, `%\%%`, 50).
				WillReturnRows(sqlmock.NewRows(shopColumns))

			shops, err := r.Find(context.Background(), "%", 50)

			require.NoError(t, err)
			require.Empty(t, shops)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("FindById", func(t *testing.T) {
		t.Run("正常系_指定IDの店舗を返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewShop(db)

			mock.ExpectQuery(shopQuery(`WHERE shops.id = $1`)).
				WithArgs(uint(10317), 1).
				WillReturnRows(shopRows())

			shop, err := r.FindById(context.Background(), 10317)

			require.NoError(t, err)
			require.Equal(t, uint(10317), shop.ID)
			require.Equal(t, "古本市場新琴似店", shop.Name)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("異常系_存在しなければErrRecordNotFoundへ変換して返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewShop(db)

			mock.ExpectQuery(shopQuery(`WHERE shops.id = $1`)).
				WithArgs(uint(99999999), 1).
				WillReturnRows(sqlmock.NewRows(shopColumns))

			shop, err := r.FindById(context.Background(), 99999999)

			require.Nil(t, shop)
			require.ErrorIs(t, err, apperror.ErrRecordNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
