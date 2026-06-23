package entity

type UserStat struct {
	UserId              string
	TotalRecords        int
	OfficialEventCount  int
	TonamelEventCount   int
	TotalMatches        int
	Wins                int
	Losses              int
	WinRate             float64
}

func NewUserStat(
	userId string,
	totalRecords int,
	officialEventCount int,
	tonamelEventCount int,
	totalMatches int,
	wins int,
	losses int,
	winRate float64,
) *UserStat {
	return &UserStat{
		UserId:             userId,
		TotalRecords:       totalRecords,
		OfficialEventCount: officialEventCount,
		TonamelEventCount:  tonamelEventCount,
		TotalMatches:       totalMatches,
		Wins:               wins,
		Losses:             losses,
		WinRate:            winRate,
	}
}
