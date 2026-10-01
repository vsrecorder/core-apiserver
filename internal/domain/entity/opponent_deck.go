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
