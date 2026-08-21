package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type UserStatRecentInterface interface {
	GetRecentMatches(
		ctx context.Context,
		userId string,
		count int,
		deckId string,
	) (*entity.RecentMatchStat, error)
}

type UserStatRecent struct {
	repo            repository.UserStatRecentInterface
	environmentRepo repository.EnvironmentInterface
}

func NewUserStatRecent(
	repo repository.UserStatRecentInterface,
	environmentRepo repository.EnvironmentInterface,
) UserStatRecentInterface {
	return &UserStatRecent{repo, environmentRepo}
}

func (u *UserStatRecent) GetRecentMatches(
	ctx context.Context,
	userId string,
	count int,
	deckId string,
) (*entity.RecentMatchStat, error) {
	// 1試合目が必ず0%/100%になる「先頭からの通算勝率」を避けるため、
	// 表示件数の半分を移動平均のウィンドウ幅とし、表示区間より前の試合も
	// ウィンドウ幅-1件だけ余分に取得して各点の直近K戦勝率を計算する。
	windowSize := count / 2
	if windowSize < 1 {
		windowSize = 1
	}
	fetchCount := count + windowSize - 1

	rawMatches, err := u.repo.FindRecentMatches(ctx, userId, fetchCount, deckId)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	environments, err := u.findEnvironmentsForMatches(ctx, rawMatches)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	// 表示対象は末尾 count 件。それより前の試合はローリング勝率計算の前情報としてのみ使う。
	displayStart := 0
	if len(rawMatches) > count {
		displayStart = len(rawMatches) - count
	}

	// 各点のローリング勝率は、そのつどウィンドウ内を数え直すと O(count^2) になる。
	// 勝利数・引き分け数の累積和を先に作り、任意区間の値を差分で引けるようにする。
	// prefixWins[k] は rawMatches[0..k-1] の勝利数（prefixWins[0]=0）。
	// 引き分けは勝率の分母から除外するため、引き分け数も累積しておく。
	prefixWins := make([]int, len(rawMatches)+1)
	prefixDraws := make([]int, len(rawMatches)+1)
	for i, rm := range rawMatches {
		prefixWins[i+1] = prefixWins[i]
		prefixDraws[i+1] = prefixDraws[i]
		if rm.DrawFlg {
			prefixDraws[i+1]++
		} else if rm.VictoryFlg {
			prefixWins[i+1]++
		}
	}

	wins := 0
	draws := 0
	matches := make([]*entity.RecentMatch, 0, len(rawMatches)-displayStart)
	for idx := displayStart; idx < len(rawMatches); idx++ {
		m := rawMatches[idx]

		windowStart := idx - windowSize + 1
		if windowStart < 0 {
			windowStart = 0
		}
		// [windowStart, idx] の勝利数・引き分け数を累積和の差分で求める（O(1)）。
		// 引き分けは分母から除外する(勝ち/(勝ち+負け))。
		windowTotal := idx - windowStart + 1
		windowWins := prefixWins[idx+1] - prefixWins[windowStart]
		windowDraws := prefixDraws[idx+1] - prefixDraws[windowStart]
		var rollingWinRate float64
		if windowDecided := windowTotal - windowDraws; windowDecided > 0 {
			rollingWinRate = float64(windowWins) / float64(windowDecided)
		}

		if m.DrawFlg {
			draws++
		} else if m.VictoryFlg {
			wins++
		}

		sequence := idx - displayStart + 1

		var environmentId, environmentTitle string
		if env := findEnvironmentForDate(environments, m.EventDate); env != nil {
			environmentId = env.ID
			environmentTitle = env.Title
		}

		matches = append(matches, entity.NewRecentMatch(
			sequence,
			m.EventDate,
			m.DeckId,
			m.OpponentsDeckInfo,
			m.VictoryFlg,
			m.DrawFlg,
			rollingWinRate,
			environmentId,
			environmentTitle,
			m.PokemonSprites,
		))
	}

	totalMatches := len(matches)
	// 引き分けは勝率の分母から除外する(勝ち/(勝ち+負け))。
	var winRate float64
	if decided := totalMatches - draws; decided > 0 {
		winRate = float64(wins) / float64(decided)
	}

	return entity.NewRecentMatchStat(userId, count, totalMatches, wins, winRate, matches), nil
}

// findEnvironmentsForMatches は対象試合の対戦日を包含する期間の環境（レギュレーション）一覧を取得する。
func (u *UserStatRecent) findEnvironmentsForMatches(
	ctx context.Context,
	matches []*entity.RecentMatch,
) ([]*entity.Environment, error) {
	if len(matches) == 0 {
		return nil, nil
	}

	minDate, maxDate := matches[0].EventDate, matches[0].EventDate
	for _, m := range matches {
		if m.EventDate.Before(minDate) {
			minDate = m.EventDate
		}
		if m.EventDate.After(maxDate) {
			maxDate = m.EventDate
		}
	}

	return u.environmentRepo.FindByTerm(ctx, minDate, maxDate)
}

// findEnvironmentForDate は指定日を含む環境を返す。
// environments は from_date の降順（EnvironmentInterface.FindByTerm の返却順）を前提とし、
// from_date <= date を満たす最初（＝最新）の環境を採用する（FindByDate と同じ判定基準）。
func findEnvironmentForDate(environments []*entity.Environment, date time.Time) *entity.Environment {
	for _, env := range environments {
		if !env.FromDate.After(date) {
			return env
		}
	}
	return nil
}
