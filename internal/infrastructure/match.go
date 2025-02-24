package infrastructure

import (
	"context"
	"slices"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"gorm.io/gorm"
)

type Match struct {
	db *gorm.DB
}

func NewMatch(
	db *gorm.DB,
) repository.MatchInterface {
	return &Match{db}
}

func (i *Match) FindById(
	ctx context.Context,
	id string,
) (*entity.Match, error) {
	var results []*model.MatchJoinGame

	tx := i.db.Table(
		"matches",
	).Select(
		"matches.id AS match_id,matches.created_at AS match_created_at,matches.updated_at AS match_updated_at,matches.deleted_at AS match_deleted_at,matches.record_id AS match_record_id,matches.deck_id AS match_deck_id,matches.user_id AS match_user_id,matches.opponents_user_id AS match_opponents_user_id,matches.bo3_flg AS match_bo3_flg,matches.qualifying_round_flg AS match_qualifying_round_flg,matches.final_tournament_flg AS match_final_tournament_flg,matches.default_victory_flg AS match_default_victory_flg,matches.default_defeat_flg AS match_default_defeat_flg,matches.victory_flg AS match_victory_flg,matches.opponents_deck_info AS match_opponents_deck_info,matches.memo AS match_memo,games.id AS game_id,games.created_at AS game_created_at,games.updated_at AS game_updated_at,games.deleted_at AS game_deleted_at,games.match_id AS game_match_id, games.user_id AS game_user_id, games.go_first AS game_go_first, games.winning_flg AS game_winning_flg,games.your_prize_cards AS game_your_prize_cards,games.opponents_prize_cards AS game_opponents_prize_cards,games.memo AS game_memo",
	).Joins(
		"INNER JOIN games on matches.id = games.match_id",
	).Where(
		"matches.id = ? AND matches.deleted_at IS NULL", id,
	).Order(
		"games.created_at ASC",
	).Scan(&results)

	if tx.Error != nil {
		return nil, tx.Error
	}

	var games []*entity.Game
	for _, result := range results {
		game := entity.NewGame(
			result.GameID,
			result.GameCreatedAt,
			result.MatchID,
			result.MatchUserId,
			result.GameGoFirst,
			result.GameWinningFlg,
			result.GameYourPrizeCards,
			result.GameOpponentsPrizeCards,
			result.GameMemo,
		)
		games = append(games, game)
	}

	match := entity.NewMatch(
		results[0].MatchID,
		results[0].MatchCreatedAt,
		results[0].MatchRecordId,
		results[0].MatchDeckId,
		results[0].MatchUserId,
		results[0].MatchOpponentsUserId,
		results[0].MatchBO3Flg,
		results[0].MatchQualifyingRoundFlg,
		results[0].MatchFinalTournamentFlg,
		results[0].MatchDefaultVictoryFlg,
		results[0].MatchDefaultDefeatFlg,
		results[0].MatchVictoryFlg,
		results[0].MatchOpponentsDeckInfo,
		results[0].MatchMemo,
		games,
	)

	return match, nil
}

func (i *Match) FindByRecordId(
	ctx context.Context,
	recordId string,
) ([]*entity.Match, error) {
	var results []*model.MatchJoinGame

	tx := i.db.Table(
		"records",
	).Select(
		"matches.id AS match_id,matches.created_at AS match_created_at,matches.updated_at AS match_updated_at,matches.deleted_at AS match_deleted_at,matches.record_id AS match_record_id,matches.deck_id AS match_deck_id,matches.user_id AS match_user_id,matches.opponents_user_id AS match_opponents_user_id,matches.bo3_flg AS match_bo3_flg,matches.qualifying_round_flg AS match_qualifying_round_flg,matches.final_tournament_flg AS match_final_tournament_flg,matches.default_victory_flg AS match_default_victory_flg,matches.default_defeat_flg AS match_default_defeat_flg,matches.victory_flg AS match_victory_flg,matches.opponents_deck_info AS match_opponents_deck_info,matches.memo AS match_memo,games.id AS game_id,games.created_at AS game_created_at,games.updated_at AS game_updated_at,games.deleted_at AS game_deleted_at,games.match_id AS game_match_id, games.user_id AS game_user_id, games.go_first AS game_go_first, games.winning_flg AS game_winning_flg,games.your_prize_cards AS game_your_prize_cards,games.opponents_prize_cards AS game_opponents_prize_cards,games.memo AS game_memo",
	).Joins(
		"INNER JOIN matches on records.id = matches.record_id INNER JOIN games on matches.id = games.match_id",
	).Where(
		"records.id = ? AND records.deleted_at IS NULL AND matches.deleted_at IS NULL", recordId,
	).Order(
		"matches.created_at, games.created_at ASC",
	).Scan(&results)

	if tx.Error != nil {
		return nil, tx.Error
	}

	v := make(map[string]*entity.Match)
	var keys []string

	for _, result := range results {
		match, ok := v[result.MatchID]

		if !ok {
			var games []*entity.Game

			game := entity.NewGame(
				result.GameID,
				result.GameCreatedAt,
				result.MatchID,
				result.MatchUserId,
				result.GameGoFirst,
				result.GameWinningFlg,
				result.GameYourPrizeCards,
				result.GameOpponentsPrizeCards,
				result.GameMemo,
			)
			games = append(games, game)

			match := entity.NewMatch(
				result.MatchID,
				result.MatchCreatedAt,
				result.MatchRecordId,
				result.MatchDeckId,
				result.MatchUserId,
				result.MatchOpponentsUserId,
				result.MatchBO3Flg,
				result.MatchQualifyingRoundFlg,
				result.MatchFinalTournamentFlg,
				result.MatchDefaultVictoryFlg,
				result.MatchDefaultDefeatFlg,
				result.MatchVictoryFlg,
				result.MatchOpponentsDeckInfo,
				result.MatchMemo,
				games,
			)

			v[result.MatchID] = match
			keys = append(keys, result.MatchID)
		} else {
			game := entity.NewGame(
				result.GameID,
				result.GameCreatedAt,
				result.MatchID,
				result.MatchUserId,
				result.GameGoFirst,
				result.GameWinningFlg,
				result.GameYourPrizeCards,
				result.GameOpponentsPrizeCards,
				result.GameMemo,
			)
			match.Games = append(match.Games, game)
		}
	}

	slices.Sort(keys)

	var matches []*entity.Match
	for _, key := range keys {
		matches = append(matches, v[key])
	}

	return matches, nil
}
