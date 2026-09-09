package model

// OfficialEventEnvironment は公式イベントの環境の例外(official_event_environments)。
// 開催日から引く環境と実際の対戦環境がズレるイベントだけが登録される。
type OfficialEventEnvironment struct {
	OfficialEventId uint `gorm:"primaryKey"`
	EnvironmentId   string
}

func NewOfficialEventEnvironment(
	officialEventId uint,
	environmentId string,
) *OfficialEventEnvironment {
	return &OfficialEventEnvironment{
		OfficialEventId: officialEventId,
		EnvironmentId:   environmentId,
	}
}
