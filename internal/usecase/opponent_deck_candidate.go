package usecase

import (
	"context"
	"sync"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	// OwnOpponentDeckCandidateWindowMonths は自身の履歴を遡る月数。
	// 環境は数か月で入れ替わるため、古い対戦まで数えると、いま当たらないデッキが
	// 出現回数の多い順で上位に残り続ける。
	OwnOpponentDeckCandidateWindowMonths = 6

	// OpponentDeckCandidateWindowMonths は全体の候補を遡る月数。
	// 自身の履歴より短くするのは、全体候補は「不足分の穴埋め」で、より今の環境に
	// 寄っているほうが役に立つため。
	OpponentDeckCandidateWindowMonths = 3

	// opponentDeckCandidateCacheTTL は全体の候補を保持する時間。全ユーザーの対戦結果を
	// GROUP BY する集計で、対戦結果の入力フォームを開くたびに走らせる必要はない。
	// 自身の履歴はこの対象にしない(対戦を記録した直後にその相手デッキが候補へ出るように)。
	opponentDeckCandidateCacheTTL = 10 * time.Minute

	// opponentDeckCandidateCacheSize は保持する全体候補の数。一覧の limit の上限
	// (helper.MaxLimit)と同じにし、リクエストの limit はこの中から切り出す。
	opponentDeckCandidateCacheSize = 100
)

type OpponentDeckCandidateInterface interface {
	// FindOpponentDeckCandidates は uid 向けの相手デッキ入力候補を limit 件まで返す。
	//
	// uid 自身の履歴からの候補を先頭に置き、limit に満たない分だけ全ユーザーの候補で
	// 埋める(重複する組み合わせは除く)。候補が尽きれば limit より少なくなる。
	FindOpponentDeckCandidates(
		ctx context.Context,
		uid string,
		limit int,
	) ([]*entity.OpponentDeckCandidate, error)
}

type OpponentDeckCandidate struct {
	repository repository.OpponentDeckCandidateInterface

	mu sync.Mutex
	// cached は直近の全体候補(出現回数順・最大 opponentDeckCandidateCacheSize 件)。
	cached    []*entity.OpponentDeckCandidate
	expiresAt time.Time
}

func NewOpponentDeckCandidate(
	repository repository.OpponentDeckCandidateInterface,
) OpponentDeckCandidateInterface {
	return &OpponentDeckCandidate{repository: repository}
}

func (u *OpponentDeckCandidate) FindOpponentDeckCandidates(
	ctx context.Context,
	uid string,
	limit int,
) ([]*entity.OpponentDeckCandidate, error) {
	if limit <= 0 {
		return []*entity.OpponentDeckCandidate{}, nil
	}

	now := timeNow()

	// 自身の履歴は毎回集計する。対戦を記録した直後に、その相手デッキが候補へ出るようにするため
	// (uid で絞った集計なので、全体の集計より軽い)。
	own, err := u.repository.FindOpponentDeckCandidates(ctx, &repository.OpponentDeckCandidateFilter{
		UserId: uid,
		Since:  now.AddDate(0, -OwnOpponentDeckCandidateWindowMonths, 0),
		Limit:  limit,
	})
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	if len(own) >= limit {
		return own[:limit], nil
	}

	global, err := u.globalCandidates(ctx, now)
	if err != nil {
		return nil, err
	}

	// 自身の履歴を先頭に、不足分を全体候補で埋める(同じ組み合わせは自身のぶんを残す)。
	ret := make([]*entity.OpponentDeckCandidate, 0, limit)
	seen := make(map[string]struct{}, limit)
	for _, candidate := range own {
		ret = append(ret, candidate)
		seen[candidate.Key()] = struct{}{}
	}
	for _, candidate := range global {
		if len(ret) >= limit {
			break
		}
		key := candidate.Key()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		ret = append(ret, candidate)
	}

	return ret, nil
}

// globalCandidates は全ユーザーの候補を返す。TTLのあいだは前回の集計結果を使い回す。
func (u *OpponentDeckCandidate) globalCandidates(
	ctx context.Context,
	now time.Time,
) ([]*entity.OpponentDeckCandidate, error) {
	// 集計中に同じ問い合わせが重ならないよう、更新はロックの中で行う
	// (数ミリ秒の集計で、TTLごとに1回しか走らない)。
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.cached != nil && now.Before(u.expiresAt) {
		return u.cached, nil
	}

	candidates, err := u.repository.FindOpponentDeckCandidates(ctx, &repository.OpponentDeckCandidateFilter{
		Since: now.AddDate(0, -OpponentDeckCandidateWindowMonths, 0),
		Limit: opponentDeckCandidateCacheSize,
	})
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	u.cached = candidates
	u.expiresAt = now.Add(opponentDeckCandidateCacheTTL)

	return u.cached, nil
}
