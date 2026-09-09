package infrastructure

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// statPeriodSQL は applyStatPeriod が組み立てるWHERE句を、DryRunでSQL文字列として取り出す。
func statPeriodSQL(t *testing.T, period repository.StatPeriod) (string, []any) {
	t.Helper()

	db, _ := setupSqlmockDB(t)

	var rows []struct{}
	query := db.Session(&gorm.Session{DryRun: true}).Table("records").Select("records.id")
	query = applyStatPeriod(query, period)

	stmt := query.Find(&rows).Statement

	return stmt.SQL.String(), stmt.Vars
}

func TestApplyStatPeriod(t *testing.T) {
	from := time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 9, 16, 0, 0, 0, 0, time.Local)

	t.Run("正常系_期間がゼロ値なら絞り込みを付けない", func(t *testing.T) {
		sql, vars := statPeriodSQL(t, repository.StatPeriod{})

		require.NotContains(t, sql, "WHERE")
		require.Empty(t, vars)
	})

	t.Run("正常系_例外イベントが無ければevent_dateの範囲だけで絞る", func(t *testing.T) {
		sql, vars := statPeriodSQL(t, repository.StatPeriod{From: from, To: to, BaseFrom: from, BaseTo: to})

		require.Contains(t, sql, "records.event_date >= $1 AND records.event_date < $2")
		require.NotContains(t, sql, "official_event_id")
		require.Equal(t, []any{from, to}, vars)
	})

	// 開催日は環境の期間内でも、別の環境として登録されているイベントは集計から外す。
	// 公式イベントに紐づかない記録まで落とさないよう、NULL は明示的に許可する
	// (NOT IN は NULL に対して不定になるため)。
	t.Run("正常系_別環境として登録されたイベントを除外する", func(t *testing.T) {
		sql, vars := statPeriodSQL(t, repository.StatPeriod{
			From: from, To: to, BaseFrom: from, BaseTo: to,
			ExcludeOfficialEventIds: []uint{1113193, 1113194},
		})

		require.Contains(t, sql, "records.official_event_id IS NULL OR records.official_event_id NOT IN ($3,$4)")
		require.Equal(t, []any{from, to, uint(1113193), uint(1113194)}, vars)
	})

	// 開催日が環境の期間外でも、その環境として登録されたイベントは拾い直す。
	t.Run("正常系_期間外の例外イベントをORで拾い直す", func(t *testing.T) {
		sql, vars := statPeriodSQL(t, repository.StatPeriod{
			From: from, To: to, BaseFrom: from, BaseTo: to,
			IncludeOfficialEventIds: []uint{1113193},
		})

		require.Contains(t, sql, "OR (records.official_event_id IN ($3)")
		require.Contains(t, sql, "records.event_date >= $4 AND records.event_date < $5")
		require.Equal(t, []any{from, to, uint(1113193), from, to}, vars)
	})

	// 環境以外の条件(week / year_month / season / standard_regulation)は例外イベントにも
	// 効かせる。ここを付け忘れると、期間の交差を指定しているのに環境の全期間から
	// 例外イベントを拾ってしまう。
	t.Run("正常系_拾い直す範囲には環境以外の条件の期間を使う", func(t *testing.T) {
		baseFrom := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
		baseTo := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)

		_, vars := statPeriodSQL(t, repository.StatPeriod{
			From: from, To: to, BaseFrom: baseFrom, BaseTo: baseTo,
			IncludeOfficialEventIds: []uint{1113193},
		})

		require.Equal(t, []any{from, to, uint(1113193), baseFrom, baseTo}, vars)
	})

	// 期間の指定が無いときは全期間が対象なので、例外イベントも当然含まれる。
	// ここでORを足すと「例外イベントだけ」に絞られてしまう。
	t.Run("正常系_期間がゼロ値なら例外イベントがあっても絞り込みを付けない", func(t *testing.T) {
		sql, vars := statPeriodSQL(t, repository.StatPeriod{
			IncludeOfficialEventIds: []uint{1113193},
		})

		require.NotContains(t, sql, "WHERE")
		require.Empty(t, vars)
	})
}
