package infrastructure

import (
	"strings"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// applyStatPeriod は records に対する期間の絞り込みを query に適用する。
// 呼び出し側は records を(JOINでも直接でも)引いている必要がある。
//
// 生成する条件は次の形。例外イベントが無ければ後半は付かず、従来どおり event_date の
// 範囲だけになる。
//
//	(event_date >= From AND event_date < To AND official_event_id NOT IN (Exclude))
//	OR (official_event_id IN (Include) AND event_date >= BaseFrom AND event_date < BaseTo)
func applyStatPeriod(query *gorm.DB, period repository.StatPeriod) *gorm.DB {
	inPeriodCond, inPeriodArgs := statPeriodInRangeClause(period)

	// 期間による絞り込みが無い(全期間)なら、例外イベントも当然含まれるので何もしない。
	if inPeriodCond == "" {
		return query
	}

	includeCond, includeArgs := statPeriodIncludeClause(period)
	if includeCond == "" {
		return query.Where(inPeriodCond, inPeriodArgs...)
	}

	return query.Where(
		"("+inPeriodCond+") OR ("+includeCond+")",
		append(inPeriodArgs, includeArgs...)...,
	)
}

// statPeriodInRangeClause は「期間内(ただし別環境として扱うイベントを除く)」の条件を返す。
func statPeriodInRangeClause(period repository.StatPeriod) (string, []any) {
	conds := make([]string, 0, 3)
	args := make([]any, 0, 3)

	if !period.From.IsZero() {
		conds = append(conds, "records.event_date >= ?")
		args = append(args, period.From)
	}
	if !period.To.IsZero() {
		conds = append(conds, "records.event_date < ?")
		args = append(args, period.To)
	}

	// 公式イベントに紐づかない記録(official_event_id が NULL / 0)は除外の対象にならない。
	// NOT IN は NULL に対して不定になるため、明示的に許可する必要がある。
	if len(period.ExcludeOfficialEventIds) > 0 {
		conds = append(conds, "(records.official_event_id IS NULL OR records.official_event_id NOT IN (?))")
		args = append(args, period.ExcludeOfficialEventIds)
	}

	if len(conds) == 0 {
		return "", nil
	}

	return strings.Join(conds, " AND "), args
}

// statPeriodIncludeClause は「期間外だがこの環境として拾い直すイベント」の条件を返す。
// 拾い直す対象が無ければ空文字を返す。
func statPeriodIncludeClause(period repository.StatPeriod) (string, []any) {
	if len(period.IncludeOfficialEventIds) == 0 {
		return "", nil
	}

	conds := []string{"records.official_event_id IN (?)"}
	args := []any{period.IncludeOfficialEventIds}

	// 環境以外の条件は例外イベントにも効かせる(環境と他の条件は交差を取るため)。
	if !period.BaseFrom.IsZero() {
		conds = append(conds, "records.event_date >= ?")
		args = append(args, period.BaseFrom)
	}
	if !period.BaseTo.IsZero() {
		conds = append(conds, "records.event_date < ?")
		args = append(args, period.BaseTo)
	}

	return strings.Join(conds, " AND "), args
}
