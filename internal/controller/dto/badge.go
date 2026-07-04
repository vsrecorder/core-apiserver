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
	// AchievedAt はオンボーディング系では永続化された獲得記録の日時、マイルストーン系・
	// 週次ストリーク系では対象シーズン内で criteria_value 番目の条件を初めて満たした日時
	// (ライブ集計)を返す。シーズン内でまだ閾値に届いていない場合は nil(JSON上は省略)。
	// 週次ストリークはシーズン中に途切れて Achieved が false に戻ることがあるが、その場合も
	// シーズン内で最初に到達した日時は保持され続ける。
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
