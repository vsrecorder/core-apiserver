package entity

// CalendarRecord は1件の記録と、それに紐づく対戦結果をまとめたもの。
type CalendarRecord struct {
	Record  *Record
	Matches []*Match
}

func NewCalendarRecord(
	record *Record,
	matches []*Match,
) *CalendarRecord {
	return &CalendarRecord{
		Record:  record,
		Matches: matches,
	}
}

// CalendarDeck は1件のデッキと、それに紐づくデッキコード(バージョン)をまとめたもの。
type CalendarDeck struct {
	Deck      *Deck
	DeckCodes []*DeckCode
}

func NewCalendarDeck(
	deck *Deck,
	deckCodes []*DeckCode,
) *CalendarDeck {
	return &CalendarDeck{
		Deck:      deck,
		DeckCodes: deckCodes,
	}
}

// Calendar は活動ログのカレンダーを組み立てるのに必要な、あるユーザの全データを表す。
//
// 呼び出し側(webapp)が記録・対戦・デッキ・デッキコードの登録日時から
// カレンダーを構築できるよう、参照先のイベント情報まで含めて一度に返す。
// これにより、以前のように記録1件ごと・デッキ1件ごとにAPIを呼ぶ必要がなくなる。
//
// 表示上の色やラベルは持たない。それらはUIの関心事であり、呼び出し側で決める。
type Calendar struct {
	Records          []*CalendarRecord
	Decks            []*CalendarDeck
	OfficialEvents   []*OfficialEvent
	UnofficialEvents []*UnofficialEvent
	TonamelEvents    []*TonamelEvent
}

func NewCalendar(
	records []*CalendarRecord,
	decks []*CalendarDeck,
	officialEvents []*OfficialEvent,
	unofficialEvents []*UnofficialEvent,
	tonamelEvents []*TonamelEvent,
) *Calendar {
	return &Calendar{
		Records:          records,
		Decks:            decks,
		OfficialEvents:   officialEvents,
		UnofficialEvents: unofficialEvents,
		TonamelEvents:    tonamelEvents,
	}
}
