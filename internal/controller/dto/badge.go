package dto

import (
	"time"
)

type BadgeDefinitionResponse struct {
	ID            string `json:"id"`
	Code          string `json:"code"`
	Category      string `json:"category"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	IconKey       string `json:"icon_key"`
	CriteriaType  string `json:"criteria_type"`
	CriteriaValue int    `json:"criteria_value"`
}

type BadgeDefinitionsResponse struct {
	Badges []*BadgeDefinitionResponse `json:"badges"`
}

type UserBadgeResponse struct {
	BadgeDefinitionResponse
	Achieved bool `json:"achieved"`
	// AchievedAt はオンボーディング系(永続化された獲得記録を持つ)のみ設定される。
	// マイルストーン系・週次ストリーク系はシーズンの集計値からその場でライブ判定するため
	// 実際の獲得日時を持たず、nil(JSON上は省略)になる。
	AchievedAt   *time.Time `json:"achieved_at,omitempty"`
	CurrentValue int        `json:"current_value"`
}

type UserBadgesResponse struct {
	UserId string `json:"user_id"`
	// Season はマイルストーン系・週次ストリーク系の判定に使った対象シーズン(終了年、例:"2026")。
	// リクエストで season 未指定の場合は現在のシーズンが入る。オンボーディング系の判定には
	// 影響しない(常に全期間で判定される)。
	Season string               `json:"season"`
	Badges []*UserBadgeResponse `json:"badges"`
}
