package entity

type UserStatMonthly struct {
	YearMonth    string
	TotalMatches int
	Wins         int
	Losses       int
	WinRate      float64
}

func NewUserStatMonthly(yearMonth string, totalMatches int, wins int, losses int, winRate float64) *UserStatMonthly {
	return &UserStatMonthly{
		YearMonth:    yearMonth,
		TotalMatches: totalMatches,
		Wins:         wins,
		Losses:       losses,
		WinRate:      winRate,
	}
}
