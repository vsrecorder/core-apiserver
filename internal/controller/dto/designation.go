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

type DesignationTierStatResponse struct {
	Tier int `json:"tier"`
	// UserCount は対象シーズンで、ちょうどこの tier を現在の称号として持つユーザー数。
	UserCount int `json:"user_count"`
}

type DesignationRankStatsResponse struct {
	// Season は集計に使った対象シーズン(終了年、例:"2026")。リクエストで season 未指定の
	// 場合は現在のシーズンが入る。
	Season string `json:"season"`
	// TotalUsers はいずれかの tier に到達した(=称号未達成を除く)ユーザーの合計数。
	// 称号ランク一覧モーダルでの「モンスターボール級以上」の分母にあたる。
	TotalUsers int                            `json:"total_users"`
	Tiers      []*DesignationTierStatResponse `json:"tiers"`
}
