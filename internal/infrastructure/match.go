package infrastructure

import (
	"context"
	"database/sql"
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
	).Select(`
		matches.id AS match_id,
		matches.created_at AS match_created_at,
		matches.updated_at AS match_updated_at,
		matches.deleted_at AS match_deleted_at,
		matches.record_id AS match_record_id,
		matches.deck_id AS match_deck_id,
		matches.deck_code_id AS match_deck_code_id,
		matches.user_id AS match_user_id,
		matches.opponents_user_id AS match_opponents_user_id,
		matches.bo3_flg AS match_bo3_flg,
		matches.qualifying_round_flg AS match_qualifying_round_flg,
		matches.final_tournament_flg AS match_final_tournament_flg,
		matches.default_victory_flg AS match_default_victory_flg,
		matches.default_defeat_flg AS match_default_defeat_flg,
		matches.victory_flg AS match_victory_flg,
		matches.opponents_deck_info AS match_opponents_deck_info,
		matches.memo AS match_memo,
		games.id AS game_id,
		games.created_at AS game_created_at,
		games.updated_at AS game_updated_at,
		games.deleted_at AS game_deleted_at,
		games.match_id AS game_match_id,
		games.user_id AS game_user_id,
		games.go_first AS game_go_first,
		games.winning_flg AS game_winning_flg,
		games.your_prize_cards AS game_your_prize_cards,
		games.opponents_prize_cards AS game_opponents_prize_cards,
		games.memo AS game_memo`,
	).Joins(
		"LEFT JOIN games ON matches.id = games.match_id",
	).Where(
		"matches.id = ? AND matches.deleted_at IS NULL", id,
	).Order(
		"games.created_at ASC",
	).Scan(&results)

	if tx.Error != nil {
		return nil, tx.Error
	}

	if len(results) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var games []*entity.Game
	for _, result := range results {
		// Gameが存在しない場合はスキップ
		// 不戦勝/不戦敗の場合はGameが存在しないため
		if result.GameID == "" {
			continue
		}

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
		results[0].MatchDeckCodeId,
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
	).Select(`
		matches.id AS match_id,
		matches.created_at AS match_created_at,
		matches.updated_at AS match_updated_at,
		matches.deleted_at AS match_deleted_at,
		matches.record_id AS match_record_id,
		matches.deck_id AS match_deck_id,
		matches.deck_code_id AS match_deck_code_id,
		matches.user_id AS match_user_id,
		matches.opponents_user_id AS match_opponents_user_id,
		matches.bo3_flg AS match_bo3_flg,
		matches.qualifying_round_flg AS match_qualifying_round_flg,
		matches.final_tournament_flg AS match_final_tournament_flg,
		matches.default_victory_flg AS match_default_victory_flg,
		matches.default_defeat_flg AS match_default_defeat_flg,
		matches.victory_flg AS match_victory_flg,
		matches.opponents_deck_info AS match_opponents_deck_info,
		matches.memo AS match_memo,
		games.id AS game_id,
		games.created_at AS game_created_at,
		games.updated_at AS game_updated_at,
		games.deleted_at AS game_deleted_at,
		games.match_id AS game_match_id,
		games.user_id AS game_user_id,
		games.go_first AS game_go_first,
		games.winning_flg AS game_winning_flg,
		games.your_prize_cards AS game_your_prize_cards,
		games.opponents_prize_cards AS game_opponents_prize_cards,
		games.memo AS game_memo`,
	).Joins(`
		INNER JOIN matches 
		ON records.id = matches.record_id
		LEFT JOIN games 
		ON matches.id = games.match_id`,
	).Where(
		"records.id = ? AND records.deleted_at IS NULL AND matches.deleted_at IS NULL",
		recordId,
	).Order(
		"matches.created_at ASC, games.created_at ASC",
	).Scan(&results)

	if tx.Error != nil {
		return nil, tx.Error
	}

	if len(results) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	v := make(map[string]*entity.Match)
	var keys []string

	for _, result := range results {
		match, ok := v[result.MatchID]

		if !ok {
			var games []*entity.Game

			// Gameが存在しない場合はスキップ
			// 不戦勝/不戦敗の場合はGameが存在しないため
			if result.GameID != "" {
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
				result.MatchID,
				result.MatchCreatedAt,
				result.MatchRecordId,
				result.MatchDeckId,
				result.MatchDeckCodeId,
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
			// Gameが存在しない場合はスキップ
			// 不戦勝/不戦敗の場合はGameが存在しないため
			if result.GameID != "" {
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
	}

	slices.Sort(keys)

	var matches []*entity.Match
	for _, key := range keys {
		matches = append(matches, v[key])
	}

	return matches, nil
}

func (i *Match) Create(
	ctx context.Context,
	entity *entity.Match,
) error {
	matchModel := model.NewMatch(
		entity.ID,
		entity.CreatedAt,
		entity.RecordId,
		entity.DeckId,
		entity.DeckCodeId,
		entity.UserId,
		entity.OpponentsUserId,
		entity.BO3Flg,
		entity.QualifyingRoundFlg,
		entity.FinalTournamentFlg,
		entity.DefaultVictoryFlg,
		entity.DefaultDefeatFlg,
		entity.VictoryFlg,
		entity.OpponentsDeckInfo,
		entity.Memo,
	)

	var gameModels []*model.Game
	for _, game := range entity.Games {
		gameModels = append(
			gameModels,
			model.NewGame(
				game.ID,
				game.CreatedAt,
				game.MatchId,
				game.UserId,
				game.GoFirst,
				game.WinningFlg,
				game.YourPrizeCards,
				game.OpponentsPrizeCards,
				game.Memo,
			),
		)
	}

	return i.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(matchModel).Error; err != nil {
			return err
		}

		for _, gameModel := range gameModels {
			if err := tx.Save(gameModel).Error; err != nil {
				return err
			}
		}

		return nil
	}, &sql.TxOptions{Isolation: sql.LevelDefault})
}

func (i *Match) Update(
	ctx context.Context,
	entity *entity.Match,
) error {
	var models []*model.Game

	if tx := i.db.Where("match_id = ?", entity.ID).Order("created_at ASC").Find(&models); tx.Error != nil {
		return tx.Error
	}

	matchModel := model.NewMatch(
		entity.ID,
		entity.CreatedAt,
		entity.RecordId,
		entity.DeckId,
		entity.DeckCodeId,
		entity.UserId,
		entity.OpponentsUserId,
		entity.BO3Flg,
		entity.QualifyingRoundFlg,
		entity.FinalTournamentFlg,
		entity.DefaultVictoryFlg,
		entity.DefaultDefeatFlg,
		entity.VictoryFlg,
		entity.OpponentsDeckInfo,
		entity.Memo,
	)

	return i.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(matchModel).Error; err != nil {
			return err
		}

		// len(models) <= len(entity.Games) の場合は新しくGameが追加されている可能性がある
		if len(models) <= len(entity.Games) {
			for i, game := range entity.Games {
				if i < len(models) { // 既存のGameを上書き
					gameModel := model.NewGame(
						models[i].ID,
						models[i].CreatedAt,
						game.MatchId,
						game.UserId,
						game.GoFirst,
						game.WinningFlg,
						game.YourPrizeCards,
						game.OpponentsPrizeCards,
						game.Memo,
					)

					if err := tx.Save(gameModel).Error; err != nil {
						return err
					}
				} else { // 新しくGameを追加
					gameModel := model.NewGame(
						game.ID,
						game.CreatedAt,
						game.MatchId,
						game.UserId,
						game.GoFirst,
						game.WinningFlg,
						game.YourPrizeCards,
						game.OpponentsPrizeCards,
						game.Memo,
					)

					if err := tx.Save(gameModel).Error; err != nil {
						return err
					}
				}
			}
		} else { // それ以外(len(models) > len(entity.Games))はGameが削除されている
			for i, game := range models {
				if i < len(entity.Games) { // 既存のGameを上書き
					gameModel := model.NewGame(
						models[i].ID,
						models[i].CreatedAt,
						entity.Games[i].MatchId,
						entity.Games[i].UserId,
						entity.Games[i].GoFirst,
						entity.Games[i].WinningFlg,
						entity.Games[i].YourPrizeCards,
						entity.Games[i].OpponentsPrizeCards,
						entity.Games[i].Memo,
					)

					if err := tx.Save(gameModel).Error; err != nil {
						return err
					}
				} else { // 既存のGameを削除
					if tx := tx.Where("id = ?", game.ID).Delete(&model.Game{}); tx.Error != nil {
						return tx.Error
					}
				}
			}
		}

		return nil
	}, &sql.TxOptions{Isolation: sql.LevelDefault})
}

func (i *Match) Delete(
	ctx context.Context,
	id string,
) error {
	return i.db.Transaction(func(tx *gorm.DB) error {
		if tx := tx.Where("match_id = ?", id).Delete(&model.Game{}); tx.Error != nil {
			return tx.Error
		}

		if tx := tx.Where("id = ?", id).Delete(&model.Match{}); tx.Error != nil {
			return tx.Error
		}

		return nil
	}, &sql.TxOptions{Isolation: sql.LevelDefault})

}
