package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// OpponentDeckCandidateFilter は相手デッキ候補の集計条件。
type OpponentDeckCandidateFilter struct {
	// UserId を指定すると、そのユーザーが作った対戦結果だけを集計する(自身の履歴からの候補)。
	// 空なら全ユーザーの対戦結果を対象にする。
	UserId string
	// Since はこの日時以降に作られた対戦結果だけを集計の対象にする。
	Since time.Time
	// Limit は返す候補の最大数。
	Limit int
}

type OpponentDeckCandidateInterface interface {
	// FindOpponentDeckCandidates は条件に合う対戦結果から、相手デッキの候補を
	// 出現回数の多い順に Limit 件返す。
	//
	// 記録の公開・非公開は問わない(集計値しか返さないため)が、集計対象外(ignore_stats_flg)の
	// 記録の対戦結果は他の集計と同じく数えない。
	FindOpponentDeckCandidates(
		ctx context.Context,
		filter *OpponentDeckCandidateFilter,
	) ([]*entity.OpponentDeckCandidate, error)
}
