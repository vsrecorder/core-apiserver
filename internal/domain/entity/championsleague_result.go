package entity

import "time"

// ChampionsleagueEventResult は大型大会の入賞1件。
// シティリーグの EventResult と違い Point を持たない(championsleague_results に列が無い)。
type ChampionsleagueEventResult struct {
	PlayerId   string
	PlayerName string
	Rank       uint
	DeckCode   string
}

func NewChampionsleagueEventResult(
	playerId string,
	playerName string,
	rank uint,
	deckCode string,
) *ChampionsleagueEventResult {
	return &ChampionsleagueEventResult{
		PlayerId:   playerId,
		PlayerName: playerName,
		Rank:       rank,
		DeckCode:   deckCode,
	}
}

// ChampionsleagueResultEvent は結果が登録されているイベントを、入賞者を含めずに表す。
// 大会ハブの件数集計や sitemap のように、識別子と開催日だけを必要とする用途で使う。
//
// シティリーグ側(CityleagueResultEvent)と違い大会IDを持つのは、大型大会は
// 「1大会 = 複数イベント(リーグ区分 × Day)」で、イベントを大会単位にまとめ直す
// 必要があるため。開催日からの逆引きでは Day1 と Day2 をまたぐ大会を束ねられない。
type ChampionsleagueResultEvent struct {
	ChampionsleagueScheduleId string
	OfficialEventId           uint
	LeagueType                uint
	EventDate                 time.Time
}

func NewChampionsleagueResultEvent(
	championsleagueScheduleId string,
	officialEventId uint,
	leagueType uint,
	eventDate time.Time,
) *ChampionsleagueResultEvent {
	return &ChampionsleagueResultEvent{
		ChampionsleagueScheduleId: championsleagueScheduleId,
		OfficialEventId:           officialEventId,
		LeagueType:                leagueType,
		EventDate:                 eventDate,
	}
}

// ChampionsleagueResult は1イベント(リーグ区分 × Day)の入賞者をまとめたもの。
type ChampionsleagueResult struct {
	ChampionsleagueScheduleId string
	OfficialEventId           uint
	LeagueType                uint
	EventDate                 time.Time
	EventResults              []*ChampionsleagueEventResult
}

func NewChampionsleagueResult(
	championsleagueScheduleId string,
	officialEventId uint,
	leagueType uint,
	eventDate time.Time,
	eventResults []*ChampionsleagueEventResult,
) *ChampionsleagueResult {
	return &ChampionsleagueResult{
		ChampionsleagueScheduleId: championsleagueScheduleId,
		OfficialEventId:           officialEventId,
		LeagueType:                leagueType,
		EventDate:                 eventDate,
		EventResults:              eventResults,
	}
}
