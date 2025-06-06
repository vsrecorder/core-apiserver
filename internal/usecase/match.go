package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/gorm"
)

type GameParam struct {
	GoFirst             bool
	WinningFlg          bool
	YourPrizeCards      uint
	OpponentsPrizeCards uint
	Memo                string
}

type MatchParam struct {
	RecordId           string
	DeckId             string
	UserId             string
	OpponentsUserId    string
	BO3Flg             bool
	QualifyingRoundFlg bool
	FinalTournamentFlg bool
	DefaultVictoryFlg  bool
	DefaultDefeatFlg   bool
	VictoryFlg         bool
	OpponentsDeckInfo  string
	Memo               string
	Games              []*GameParam
}

func NewGameParam(
	goFirst bool,
	winningFlg bool,
	yourPrizeCards uint,
	opponentsPrizeCards uint,
	memo string,
) *GameParam {
	return &GameParam{
		GoFirst:             goFirst,
		WinningFlg:          winningFlg,
		YourPrizeCards:      yourPrizeCards,
		OpponentsPrizeCards: opponentsPrizeCards,
		Memo:                memo,
	}
}

func NewMatchParam(
	recordId string,
	deckId string,
	userId string,
	opponentsUserId string,
	bo3Flg bool,
	qualifyingRoundFlg bool,
	finalTournamentFlg bool,
	defaultVictoryFlg bool,
	defaultDefeatFlg bool,
	victoryFlg bool,
	opponentsDeckInfo string,
	memo string,
	games []*GameParam,
) *MatchParam {
	return &MatchParam{
		RecordId:           recordId,
		DeckId:             deckId,
		UserId:             userId,
		OpponentsUserId:    opponentsUserId,
		BO3Flg:             bo3Flg,
		QualifyingRoundFlg: qualifyingRoundFlg,
		FinalTournamentFlg: finalTournamentFlg,
		DefaultVictoryFlg:  defaultVictoryFlg,
		DefaultDefeatFlg:   defaultDefeatFlg,
		VictoryFlg:         victoryFlg,
		OpponentsDeckInfo:  opponentsDeckInfo,
		Memo:               memo,
		Games:              games,
	}
}

type MatchInterface interface {
	FindById(
		ctx context.Context,
		id string,
	) (*entity.Match, error)

	FindByRecordId(
		ctx context.Context,
		recordId string,
	) ([]*entity.Match, error)

	Create(
		ctx context.Context,
		param *MatchParam,
	) (*entity.Match, error)

	Update(
		ctx context.Context,
		id string,
		param *MatchParam,
	) (*entity.Match, error)

	Delete(
		ctx context.Context,
		id string,
	) error
}

type Match struct {
	repository repository.MatchInterface
}

func NewMatch(
	repository repository.MatchInterface,
) MatchInterface {
	return &Match{repository}
}

func (u *Match) FindById(
	ctx context.Context,
	id string,
) (*entity.Match, error) {
	match, err := u.repository.FindById(ctx, id)

	if err != nil {
		return nil, err
	}

	return match, nil
}

func (u *Match) FindByRecordId(
	ctx context.Context,
	recordId string,
) ([]*entity.Match, error) {
	matches, err := u.repository.FindByRecordId(ctx, recordId)

	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (u *Match) Create(
	ctx context.Context,
	param *MatchParam,
) (*entity.Match, error) {
	matchId, err := generateId()
	if err != nil {
		return nil, err
	}

	createdAt := time.Now().UTC().Truncate(0)

	var games []*entity.Game
	for _, game := range param.Games {
		gameId, err := generateId()
		if err != nil {
			return nil, err
		}

		createdAt := time.Now().UTC().Truncate(0)

		games = append(
			games,
			entity.NewGame(
				gameId,
				createdAt,
				matchId,
				param.UserId,
				game.GoFirst,
				game.WinningFlg,
				game.YourPrizeCards,
				game.OpponentsPrizeCards,
				game.Memo,
			),
		)
	}

	match := entity.NewMatch(
		matchId,
		createdAt,
		param.RecordId,
		param.DeckId,
		param.UserId,
		param.OpponentsUserId,
		param.BO3Flg,
		param.QualifyingRoundFlg,
		param.FinalTournamentFlg,
		param.DefaultVictoryFlg,
		param.DefaultDefeatFlg,
		param.VictoryFlg,
		param.OpponentsDeckInfo,
		param.Memo,
		games,
	)

	if err := u.repository.Create(ctx, match); err != nil {
		return nil, err
	}

	return match, nil
}

func (u *Match) Update(
	ctx context.Context,
	id string,
	param *MatchParam,
) (*entity.Match, error) {
	// 指定されたidのMatchを取得
	ret, err := u.repository.FindById(ctx, id)
	if err == gorm.ErrRecordNotFound {
		return nil, err
	} else if err != nil {
		return nil, err
	}

	var games []*entity.Game

	// len(ret.Games) <= len(param.Games) の場合は新しくGameが追加されている可能性がある
	if len(ret.Games) <= len(param.Games) {
		for i, game := range param.Games {
			if i < len(ret.Games) { // 既存のGameを上書き
				games = append(
					games,
					entity.NewGame(
						ret.Games[i].ID,
						ret.Games[i].CreatedAt,
						id,
						param.UserId,
						game.GoFirst,
						game.WinningFlg,
						game.YourPrizeCards,
						game.OpponentsPrizeCards,
						game.Memo,
					),
				)
			} else { // 新しくGameを追加
				gameId, err := generateId()
				if err != nil {
					return nil, err
				}

				createdAt := time.Now().UTC().Truncate(0)

				games = append(
					games,
					entity.NewGame(
						gameId,
						createdAt,
						id,
						param.UserId,
						game.GoFirst,
						game.WinningFlg,
						game.YourPrizeCards,
						game.OpponentsPrizeCards,
						game.Memo,
					),
				)
			}
		}
	} else { // それ以外(len(ret.Games) > len(param.Games))はGameが削除されている
		for i, game := range param.Games {
			games = append(
				games,
				entity.NewGame(
					ret.Games[i].ID,
					ret.Games[i].CreatedAt,
					id,
					param.UserId,
					game.GoFirst,
					game.WinningFlg,
					game.YourPrizeCards,
					game.OpponentsPrizeCards,
					game.Memo,
				),
			)
		}
	}

	match := entity.NewMatch(
		ret.ID,
		ret.CreatedAt,
		param.RecordId,
		param.DeckId,
		param.UserId,
		param.OpponentsUserId,
		param.BO3Flg,
		param.QualifyingRoundFlg,
		param.FinalTournamentFlg,
		param.DefaultVictoryFlg,
		param.DefaultDefeatFlg,
		param.VictoryFlg,
		param.OpponentsDeckInfo,
		param.Memo,
		games,
	)

	if err := u.repository.Update(ctx, match); err != nil {
		return nil, err
	}

	return match, nil
}

func (u *Match) Delete(
	ctx context.Context,
	id string,
) error {
	err := u.repository.Delete(ctx, id)

	if err != nil {
		return err
	}

	return nil
}
