package entity

import (
	"time"
)

type Match struct {
	ID                 string
	CreatedAt          time.Time
	RecordId           string
	DeckId             string
	DeckCodeId         string
	UserId             string
	OpponentsUserId    string
	BO3Flg             bool
	GroupMatchFlg      bool
	QualifyingRoundFlg bool
	FinalTournamentFlg bool
	DefaultVictoryFlg  bool
	DefaultDefeatFlg   bool
	VictoryFlg         bool
	// DrawFlg は両者引き分け(ダブルドロー)を表す。BO3で2本先取に届かず
	// 1勝1敗のまま決着した場合のみ true になる(BO1/チーム戦では常に false)。
	DrawFlg              bool
	GroupMatchVictoryFlg bool
	OpponentsDeckInfo    string
	Memo                 string
	Games                []*Game
	PokemonSprites       []*PokemonSprite
	// Tags は付与されたタグ。読み込み時にインフラ層が詰める。
	// 付与の書き込みは TagRepository.ReplaceMatchTags が担うため、
	// NewMatch のコンストラクタ引数には含めない(deck の Tags と同じ扱い)。
	Tags []*Tag
	// Position は record 内での表示順序。Reorder によってのみ更新されるため、
	// NewMatch のコンストラクタ引数には含めず、必要な箇所で個別に設定する。
	Position int
}

// MatchResult は対戦全体の結果(勝ち/負け/引き分け)を表す3値。
// VictoryFlg / DrawFlg の組み合わせを一箇所で解釈し、勝敗の考慮漏れを防ぐ。
type MatchResult string

const (
	MatchResultWin  MatchResult = "win"
	MatchResultLose MatchResult = "lose"
	MatchResultDraw MatchResult = "draw"
)

// Result は対戦全体の結果を返す。DrawFlg を最優先で判定する。
func (m *Match) Result() MatchResult {
	if m.DrawFlg {
		return MatchResultDraw
	}
	if m.VictoryFlg {
		return MatchResultWin
	}
	return MatchResultLose
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
	drawFlg bool,
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
		DrawFlg:              drawFlg,
		GroupMatchVictoryFlg: groupMatchVictoryFlg,
		OpponentsDeckInfo:    opponentsDeckInfo,
		Memo:                 memo,
		Games:                games,
		PokemonSprites:       pokemonSprites,
	}
}

// MatchSummary は1つの記録(record)に紐づく対戦の集計。
//
// 記録一覧のカードが必要としているのは勝敗数とチーム戦/BO3の有無だけで、
// 対戦一覧そのものではない。記録ごとに対戦一覧を取得すると1ページ(10件)で
// 10往復になるため、この形に落として1回でまとめて返せるようにしている。
type MatchSummary struct {
	RecordId string
	Total    int
	Wins     int
	// Losses は Total から Wins と Draws を引いた数。引き分けは負けに数えない。
	Losses int
	Draws  int
	// HasGroupMatch / HasBo3 は、その記録に該当する対戦が1件でもあるか。
	HasGroupMatch bool
	HasBo3        bool
}

func NewMatchSummary(
	recordId string,
	total int,
	wins int,
	losses int,
	draws int,
	hasGroupMatch bool,
	hasBo3 bool,
) *MatchSummary {
	return &MatchSummary{
		RecordId:      recordId,
		Total:         total,
		Wins:          wins,
		Losses:        losses,
		Draws:         draws,
		HasGroupMatch: hasGroupMatch,
		HasBo3:        hasBo3,
	}
}
