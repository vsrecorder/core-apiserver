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
		return nil, err
	}

	environments, err := u.findEnvironmentsForMatches(ctx, rawMatches)
	if err != nil {
		return nil, err
	}

	// 表示対象は末尾 count 件。それより前の試合はローリング勝率計算の前情報としてのみ使う。
	displayStart := 0
	if len(rawMatches) > count {
		displayStart = len(rawMatches) - count
	}

	wins := 0
	matches := make([]*entity.RecentMatch, 0, len(rawMatches)-displayStart)
	for idx := displayStart; idx < len(rawMatches); idx++ {
		m := rawMatches[idx]

		windowStart := idx - windowSize + 1
		if windowStart < 0 {
			windowStart = 0
		}
		windowWins, windowTotal := 0, 0
		for j := windowStart; j <= idx; j++ {
			windowTotal++
			if rawMatches[j].VictoryFlg {
				windowWins++
			}
		}
		var rollingWinRate float64
		if windowTotal > 0 {
			rollingWinRate = float64(windowWins) / float64(windowTotal)
		}

		if m.VictoryFlg {
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
			rollingWinRate,
			environmentId,
			environmentTitle,
			m.PokemonSprites,
		))
	}

	totalMatches := len(matches)
	var winRate float64
	if totalMatches > 0 {
		winRate = float64(wins) / float64(totalMatches)
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
