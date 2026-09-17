package entity

// OpponentDeckCandidate は相手デッキの入力候補。対戦結果から
// 「相手デッキの表記 × スプライト(1体目・2体目)」の組み合わせごとに出現回数を数えたもの。
//
// 集計値だけを持ち、誰の対戦かは含めない(全ユーザーを対象に数える場合があるため)。
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

// Key は候補を一意に識別する文字列。同じ表記でもスプライトが違えば別の候補として扱う
// (webapp が候補を突き合わせるときのキーと同じ組み立て方)。
func (c *OpponentDeckCandidate) Key() string {
	return c.OpponentsDeckInfo + "|" + c.spriteIdAt(1) + "|" + c.spriteIdAt(2)
}

func (c *OpponentDeckCandidate) spriteIdAt(position uint) string {
	for _, sprite := range c.PokemonSprites {
		if sprite.Position == position {
			return sprite.ID
		}
	}

	return ""
}
