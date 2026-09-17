package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type OpponentDeckCandidateInterface interface {
	// FindOpponentDeckCandidates は since 以降に作られた全ユーザーの対戦結果から、
	// 相手デッキの候補を出現回数の多い順に limit 件返す。
	// 記録の公開・非公開は問わない(集計値しか返さないため)が、集計対象外(ignore_stats_flg)の
	// 記録の対戦結果は他の集計と同じく数えない。
	FindOpponentDeckCandidates(
		ctx context.Context,
		since time.Time,
		limit int,
	) ([]*entity.OpponentDeckCandidate, error)
}
