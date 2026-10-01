package entity

// OpponentDeck はユーザー自身の対戦結果に付けた相手デッキの組み合わせ
// (表記 × スプライト(1体目・2体目))と、その組み合わせの対戦の数。
//
// 相手デッキの一括編集で、置き換える対象を選ばせるために使う。表記ゆれ(「ドラパ」「ドラパルト」)や
// スプライトの付け忘れがあると、相手デッキの分布で別のデッキとして数えられてしまうため、
// 同じ組み合わせの対戦をまとめて直せるようにしている。
type OpponentDeck struct {
	OpponentDeckSpec
	// Count はこの組み合わせの対戦の数。
	Count int
	// LastEventDate はこの組み合わせと最後に対戦した記録の開催日(YYYY-MM-DD)。
	LastEventDate string
}

func NewOpponentDeck(
	opponentsDeckInfo string,
	pokemonSprites []*PokemonSprite,
	count int,
	lastEventDate string,
) *OpponentDeck {
	return &OpponentDeck{
		OpponentDeckSpec: OpponentDeckSpec{
			OpponentsDeckInfo: opponentsDeckInfo,
			PokemonSprites:    pokemonSprites,
		},
		Count:         count,
		LastEventDate: lastEventDate,
	}
}

// OpponentDeckSpec は相手デッキの表記とスプライトの組み合わせ。
// 一括編集の「置き換え元」と「置き換え先」の指定に使う。
type OpponentDeckSpec struct {
	OpponentsDeckInfo string
	// PokemonSprites は position 1 / 2 のスプライト。未設定の枠は含めない。
	PokemonSprites []*PokemonSprite
}

func NewOpponentDeckSpec(
	opponentsDeckInfo string,
	pokemonSprites []*PokemonSprite,
) *OpponentDeckSpec {
	return &OpponentDeckSpec{
		OpponentsDeckInfo: opponentsDeckInfo,
		PokemonSprites:    pokemonSprites,
	}
}

// SpriteIdAt は position の枠のスプライト ID を返す。未設定なら空文字。
func (s *OpponentDeckSpec) SpriteIdAt(position uint) string {
	for _, sprite := range s.PokemonSprites {
		if sprite.Position == position {
			return sprite.ID
		}
	}

	return ""
}

// IsEmpty は表記もスプライトも無いか。不戦勝・不戦敗など相手デッキが無い対戦がこの形になる。
func (s *OpponentDeckSpec) IsEmpty() bool {
	return s.OpponentsDeckInfo == "" && len(s.PokemonSprites) == 0
}

// OpponentDeckMatch は相手デッキの組み合わせに当てはまる対戦 1 件と、その対戦を付けた記録の見出し。
//
// 一括編集で組み合わせを選んだときに、どの記録でどんな対戦結果に付けた表記なのかを見せるために使う
// (「ドラパ」が本当にドラパルトなのか、いつの対戦なのかを思い出してから直せるように)。
type OpponentDeckMatch struct {
	MatchId  string
	RecordId string
	// EventDate は記録の開催日(YYYY-MM-DD)。未設定の記録は空文字。
	EventDate string
	// EventType は記録のイベントの種類("official" / "tonamel" / "unofficial")。どれでもなければ空文字。
	EventType string
	// EventTitle はイベントのタイトル。取得できなければ空文字。
	EventTitle string
	// DeckName は記録に登録した自分のデッキの名前。未登録なら空文字。
	DeckName             string
	BO3Flg               bool
	GroupMatchFlg        bool
	GroupMatchVictoryFlg bool
	DefaultVictoryFlg    bool
	DefaultDefeatFlg     bool
	VictoryFlg           bool
	DrawFlg              bool
	// Games は対局(1本目から順)。不戦勝・不戦敗や対局を入力していない対戦は空。
	Games []*OpponentDeckMatchGame
}

// OpponentDeckMatchGame は対戦の中の対局 1 本の先攻・後攻と勝敗。
type OpponentDeckMatchGame struct {
	GoFirst    bool
	WinningFlg bool
}
