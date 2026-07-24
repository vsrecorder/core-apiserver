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
	// StandaloneThreshold は CriteriaType が "official_city_league_record"(レギュラーの
	// 継続条件)の場合のみ、前シーズンの実績を問わず今シーズン単独で達成とみなす閾値。
	// それ以外は常に0。
	StandaloneThreshold int `json:"standalone_threshold"`
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
	// PreviousValue は CriteriaType が "official_city_league_record"(レギュラーの継続条件)の
	// 場合のみ、前シーズンの集計値。それ以外は常に0。
	PreviousValue int `json:"previous_value"`
	// MissingOfficialEventRecord は CriteriaType が "official_city_league_placement"(ベテラン)
	// または "official_city_league_playoff"(熟練)かつ Achieved が false の場合のみ、
	// 未達成の原因が「公式サイトの結果はあるが、対応する大会の記録をまだ作成していないこと」
	// であるかを表す。それ以外は常にfalse。
	MissingOfficialEventRecord bool `json:"missing_official_event_record"`
	// CityLeagueRecordWithoutPlayerLink は CriteriaType が "official_city_league_placement"
	// (ベテラン)または "official_city_league_playoff"(熟練)の場合のみ、プレイヤーズ
	// クラブ未連携であるにもかかわらず、対象シーズン内にシティリーグの記録を既に作成済み
	// であるかを表す。それ以外、またはプレイヤーズクラブ連携済みの場合は常にfalse。
	CityLeagueRecordWithoutPlayerLink bool `json:"city_league_record_without_player_link"`
	// CityLeagueWinCount / CityLeaguePlacementCount / CityLeagueParticipationCount は CriteriaType が
	// "official_city_league_grandmaster"(名人)の場合のみ設定される、称号詳細モーダルの
	// 「優勝 N/1」「入賞 N/参加数」プログレスバー用の集計値。それ以外は常に0。
	// ベテラン〜名人はいずれも「記録」と「公式結果」の両方が必須のため、いずれも記録ベースで数える。
	//   - CityLeagueWinCount: 今シーズンの優勝(rank1)の記録数(優勝バーの分子。分母は CriteriaValue=1)
	//   - CityLeaguePlacementCount: 今シーズンの、記録と公式結果がそろった入賞大会数(入賞バーの分子)
	//   - CityLeagueParticipationCount: 今シーズンの参加大会数=シティリーグ記録数(入賞バーの分母)
	CityLeagueWinCount           int `json:"city_league_win_count"`
	CityLeaguePlacementCount     int `json:"city_league_placement_count"`
	CityLeagueParticipationCount int `json:"city_league_participation_count"`
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
