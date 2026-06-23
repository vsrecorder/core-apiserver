package entity

type UserStat struct {
	UserId       string
	TotalMatches int
	Wins         int
	Losses       int
	WinRate      float64
}

func NewUserStat(
	userId string,
	totalMatches int,
	wins int,
	losses int,
	winRate float64,
) *UserStat {
	return &UserStat{
		UserId:       userId,
		TotalMatches: totalMatches,
		Wins:         wins,
		Losses:       losses,
		WinRate:      winRate,
	}
}
