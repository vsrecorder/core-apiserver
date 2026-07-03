package dto

type DesignationResponse struct {
	ID            string `json:"id"`
	Tier          int    `json:"tier"`
	Code          string `json:"code"`
	Emoji         string `json:"emoji"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	CriteriaType  string `json:"criteria_type"`
	CriteriaValue int    `json:"criteria_value"`
}

type DesignationsResponse struct {
	Designations []*DesignationResponse `json:"designations"`
}

type DesignationLadderItemResponse struct {
	DesignationResponse
	Achieved bool `json:"achieved"`
	// CurrentValue は CriteriaType に対応する、対象シーズンでの現在の集計値。
	// CriteriaType が "unimplemented" の場合は常に0。
	CurrentValue int `json:"current_value"`
}

type UserDesignationResponse struct {
	UserId string `json:"user_id"`
	// Season は判定に使った対象シーズン(終了年、例:"2026")。リクエストで season 未指定の
	// 場合は現在のシーズンが入る。
	Season  string                           `json:"season"`
	Current *DesignationResponse             `json:"current"`
	Ladder  []*DesignationLadderItemResponse `json:"ladder"`
}
