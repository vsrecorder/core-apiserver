package entity

import (
	"time"
)

type Match struct {
	ID                   string
	CreatedAt            time.Time
	RecordId             string
	DeckId               string
	DeckCodeId           string
	UserId               string
	OpponentsUserId      string
	BO3Flg               bool
	GroupMatchFlg        bool
	QualifyingRoundFlg   bool
	FinalTournamentFlg   bool
	DefaultVictoryFlg    bool
	DefaultDefeatFlg     bool
	VictoryFlg           bool
	GroupMatchVictoryFlg bool
	OpponentsDeckInfo    string
	Memo                 string
	Games                []*Game
	PokemonSprites       []*PokemonSprite
	// Position は record 内での表示順序。Reorder によってのみ更新されるため、
	// NewMatch のコンストラクタ引数には含めず、必要な箇所で個別に設定する。
	Position int
}

// MatchOrder は Reorder で1件の match に適用する並び順とセクション分類を表す。
type MatchOrder struct {
	ID                 string
	QualifyingRoundFlg bool
	FinalTournamentFlg bool
}

func NewMatch(
	id string,
	createdAt time.Time,
	recordId string,
	deckId string,
	deckCodeId string,
	userId string,
	opponentsUserId string,
	bo3Flg bool,
	groupMatchFlg bool,
	qualifyingRoundFlg bool,
	finalTournamentFlg bool,
	defaultVictoryFlg bool,
	defaultDefeatFlg bool,
	victoryFlg bool,
	groupMatchVictoryFlg bool,
	opponentsDeckInfo string,
	memo string,
	games []*Game,
	pokemonSprites []*PokemonSprite,
) *Match {
	return &Match{
		ID:                   id,
		CreatedAt:            createdAt,
		RecordId:             recordId,
		DeckId:               deckId,
		DeckCodeId:           deckCodeId,
		UserId:               userId,
		OpponentsUserId:      opponentsUserId,
		BO3Flg:               bo3Flg,
		GroupMatchFlg:        groupMatchFlg,
		QualifyingRoundFlg:   qualifyingRoundFlg,
		FinalTournamentFlg:   finalTournamentFlg,
		DefaultVictoryFlg:    defaultVictoryFlg,
		DefaultDefeatFlg:     defaultDefeatFlg,
		VictoryFlg:           victoryFlg,
		GroupMatchVictoryFlg: groupMatchVictoryFlg,
		OpponentsDeckInfo:    opponentsDeckInfo,
		Memo:                 memo,
		Games:                games,
		PokemonSprites:       pokemonSprites,
	}
}
