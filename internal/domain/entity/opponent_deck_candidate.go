package entity

// OpponentDeckCandidate は相手デッキの入力候補。全ユーザーの対戦結果から
// 「相手デッキの表記 × スプライト(1体目・2体目)」の組み合わせごとに出現回数を数えたもの。
//
// 自分の対戦がまだ無いユーザーにも候補を出すために使う。集計値だけを持ち、
// 誰の対戦かは含まない(記録の公開・非公開を問わず集計するため、個人に結びつく項目は持たせない)。
type OpponentDeckCandidate struct {
	OpponentsDeckInfo string
	// PokemonSprites は候補に付くスプライト(position 1 / 2)。未設定なら空。
	PokemonSprites []*PokemonSprite
	// Count はこの組み合わせが対戦結果に現れた回数。
	Count int
}

func NewOpponentDeckCandidate(
	opponentsDeckInfo string,
	pokemonSprites []*PokemonSprite,
	count int,
) *OpponentDeckCandidate {
	return &OpponentDeckCandidate{
		OpponentsDeckInfo: opponentsDeckInfo,
		PokemonSprites:    pokemonSprites,
		Count:             count,
	}
}
