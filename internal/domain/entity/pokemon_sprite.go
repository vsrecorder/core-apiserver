package entity

type PokemonSprite struct {
	ID string
	// 表示枠の位置(1 or 2)。position を扱わない経路では 0。
	Position uint
}

func NewPokemonSprite(
	id string,
) *PokemonSprite {
	return &PokemonSprite{
		ID: id,
	}
}

// NewPokemonSpriteWithPosition は表示枠の位置(position)付きで生成する。
// デッキのスプライトのようにスロットを固定して往復させる経路で使う。
func NewPokemonSpriteWithPosition(
	id string,
	position uint,
) *PokemonSprite {
	return &PokemonSprite{
		ID:       id,
		Position: position,
	}
}
