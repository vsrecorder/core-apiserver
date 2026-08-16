package entity

import (
	"time"
)

type EventResult struct {
	PlayerId   string
	PlayerName string
	Rank       uint
	Point      uint
	DeckCode   string
}

func NewEventResult(
	playerId string,
	playerName string,
	rank uint,
	point uint,
	deckCode string,
) *EventResult {
	return &EventResult{
		PlayerId:   playerId,
		PlayerName: playerName,
		Rank:       rank,
		Point:      point,
		DeckCode:   deckCode,
	}
}

// CityleagueResultEvent は結果が登録されているイベントを、入賞者を含めずに表す。
// sitemap の生成のように、イベントの識別子と開催日だけを必要とする用途で使う。
type CityleagueResultEvent struct {
	OfficialEventId uint
	LeagueType      uint
	EventDate       time.Time
}

func NewCityleagueResultEvent(
	officialEventId uint,
	leagueType uint,
	eventDate time.Time,
) *CityleagueResultEvent {
	return &CityleagueResultEvent{
		OfficialEventId: officialEventId,
		LeagueType:      leagueType,
		EventDate:       eventDate,
	}
}

// PlayerCityleagueResult は「あるプレイヤーIDの入賞1件」を、開催イベントの情報込みで表す。
//
// イベント単位でまとめる CityleagueResult と違い、入賞1件を1要素として扱う。
// トレーナー情報ページの「入賞したシティリーグ」は、自分の入賞を1件=1枚のカードとして
// 並べるため、イベント単位の入れ子は不要な一方、大会名・店舗名は各カードに必要になる。
// 呼び出し側(webapp)がイベント情報を別途引くとカード数だけ往復が増えるので、
// リポジトリ層で official_events を結合して1回の取得で完結させる。
type PlayerCityleagueResult struct {
	CityleagueScheduleId string
	OfficialEventId      uint
	LeagueType           uint
	EventDate            time.Time
	Rank                 uint
	Point                uint
	DeckCode             string
	EventTitle           string
	ShopName             string
	PrefectureName       string
	EnvironmentTitle     string
}

func NewPlayerCityleagueResult(
	cityleagueScheduleId string,
	officialEventId uint,
	leagueType uint,
	eventDate time.Time,
	rank uint,
	point uint,
	deckCode string,
	eventTitle string,
	shopName string,
	prefectureName string,
	environmentTitle string,
) *PlayerCityleagueResult {
	return &PlayerCityleagueResult{
		CityleagueScheduleId: cityleagueScheduleId,
		OfficialEventId:      officialEventId,
		LeagueType:           leagueType,
		EventDate:            eventDate,
		Rank:                 rank,
		Point:                point,
		DeckCode:             deckCode,
		EventTitle:           eventTitle,
		ShopName:             shopName,
		PrefectureName:       prefectureName,
		EnvironmentTitle:     environmentTitle,
	}
}

type CityleagueResult struct {
	CityleagueScheduleId string
	OfficialEventId      uint
	LeagueType           uint
	EventDate            time.Time
	EventResults         []*EventResult
}

func NewCityleagueResult(
	cityleagueScheduleId string,
	officialEventId uint,
	leagueType uint,
	eventDate time.Time,
	eventResults []*EventResult,
) *CityleagueResult {
	return &CityleagueResult{
		CityleagueScheduleId: cityleagueScheduleId,
		OfficialEventId:      officialEventId,
		LeagueType:           leagueType,
		EventDate:            eventDate,
		EventResults:         eventResults,
	}
}
