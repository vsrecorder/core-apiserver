package dto

type UserStatResponse struct {
	UserId               string `json:"user_id"`
	Week                 string `json:"week,omitempty"`
	YearMonth            string `json:"year_month,omitempty"`
	EnvironmentId        string `json:"environment_id,omitempty"`
	Season               string `json:"season,omitempty"`
	StandardRegulationId string `json:"standard_regulation_id,omitempty"`
	RegulationId         uint   `json:"regulation_id,omitempty"`
	// 不戦勝/不戦敗を集計から外したかどうか。
	// 他の絞り込みと違って omitempty を付けない。false は「絞り込まなかった」ではなく
	// 「不戦も含めて数えた」という結果そのもので、落とすと受け取り側が区別できないため。
	ExcludeDefaultMatches bool    `json:"exclude_default_matches"`
	TotalRecords          int     `json:"total_records"`
	OfficialEventCount    int     `json:"official_event_count"`
	TonamelEventCount     int     `json:"tonamel_event_count"`
	UnofficialEventCount  int     `json:"unofficial_event_count"`
	TotalMatches          int     `json:"total_matches"`
	Wins                  int     `json:"wins"`
	Losses                int     `json:"losses"`
	WinRate               float64 `json:"win_rate"`
}
