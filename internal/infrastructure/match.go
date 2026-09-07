package infrastructure

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
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
		matches.group_match_flg AS match_group_match_flg,
		matches.qualifying_round_flg AS match_qualifying_round_flg,
		matches.final_tournament_flg AS match_final_tournament_flg,
		matches.default_victory_flg AS match_default_victory_flg,
		matches.default_defeat_flg AS match_default_defeat_flg,
		matches.victory_flg AS match_victory_flg,
		matches.draw_flg AS match_draw_flg,
		matches.group_match_victory_flg AS match_group_match_victory_flg,
		matches.opponents_deck_info AS match_opponents_deck_info,
		matches.memo AS match_memo,
		matches.position AS match_position,
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
		// 論理削除済みのgamesを取得しないよう、JOIN条件でdeleted_atを除外する。
		// (Table()+Scan()のためgormのソフトデリートは自動適用されない)
		"LEFT JOIN games ON matches.id = games.match_id AND games.deleted_at IS NULL",
	).Where(
		"matches.id = ? AND matches.deleted_at IS NULL", id,
	).Order(
		"games.created_at ASC",
	).Scan(&results)

	if tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	if len(results) == 0 {
		return nil, apperror.ErrRecordNotFound
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

	var matchPokemonSpriteModels []*model.MatchPokemonSprite
	if tx := i.db.Where("match_id = ?", id).Order("position ASC").Find(&matchPokemonSpriteModels); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	var pokemonSprites []*entity.PokemonSprite
	for _, matchPokemonSpriteModel := range matchPokemonSpriteModels {
		entity := entity.NewPokemonSpriteWithPosition(matchPokemonSpriteModel.PokemonSpriteId, matchPokemonSpriteModel.Position)
		pokemonSprites = append(pokemonSprites, entity)
	}

	tagsByMatchId, err := findTagsByMatchIds(ctx, i.db, []string{id})
	if err != nil {
		logError(ctx, err)
		return nil, err
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
		results[0].MatchGroupMatchFlg,
		results[0].MatchQualifyingRoundFlg,
		results[0].MatchFinalTournamentFlg,
		results[0].MatchDefaultVictoryFlg,
		results[0].MatchDefaultDefeatFlg,
		results[0].MatchVictoryFlg,
		results[0].MatchDrawFlg,
		results[0].MatchGroupMatchVictoryFlg,
		results[0].MatchOpponentsDeckInfo,
		results[0].MatchMemo,
		games,
		pokemonSprites,
	)
	match.Position = results[0].MatchPosition
	match.Tags = tagsByMatchId[id]

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
		matches.group_match_victory_flg AS match_group_match_victory_flg,
		matches.qualifying_round_flg AS match_qualifying_round_flg,
		matches.final_tournament_flg AS match_final_tournament_flg,
		matches.default_victory_flg AS match_default_victory_flg,
		matches.default_defeat_flg AS match_default_defeat_flg,
		matches.victory_flg AS match_victory_flg,
		matches.draw_flg AS match_draw_flg,
		matches.group_match_flg AS match_group_match_flg,
		matches.opponents_deck_info AS match_opponents_deck_info,
		matches.memo AS match_memo,
		matches.position AS match_position,
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
		ON matches.id = games.match_id AND games.deleted_at IS NULL`,
	).Where(
		"records.id = ? AND records.deleted_at IS NULL AND matches.deleted_at IS NULL",
		recordId,
	).Order(
		"matches.position ASC, games.created_at ASC",
	).Scan(&results)

	if tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	if len(results) == 0 {
		return nil, apperror.ErrRecordNotFound
	}

	spritesByMatchId, err := findMatchPokemonSpritesByMatchIds(ctx, i.db, matchIdsOf(results))
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	tagsByMatchId, err := findTagsByMatchIds(ctx, i.db, matchIdsOf(results))
	if err != nil {
		logError(ctx, err)
		return nil, err
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
				result.MatchGroupMatchFlg,
				result.MatchQualifyingRoundFlg,
				result.MatchFinalTournamentFlg,
				result.MatchDefaultVictoryFlg,
				result.MatchDefaultDefeatFlg,
				result.MatchVictoryFlg,
				result.MatchDrawFlg,
				result.MatchGroupMatchVictoryFlg,
				result.MatchOpponentsDeckInfo,
				result.MatchMemo,
				games,
				spritesByMatchId[result.MatchID],
			)
			match.Position = result.MatchPosition
			match.Tags = tagsByMatchId[result.MatchID]

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

	// keys は position ASC のクエリ順で先着順に積まれているため、そのまま順序として使う
	var matches []*entity.Match
	for _, key := range keys {
		matches = append(matches, v[key])
	}

	return matches, nil
}

func (i *Match) FindByUserId(
	ctx context.Context,
	userId string,
	limit int,
) ([]*entity.Match, error) {
	var results []*model.MatchJoinGame

	subQuery := i.db.Table("matches").
		Select("id").
		Where("user_id = ? AND deleted_at IS NULL", userId).
		Order("created_at DESC").
		Limit(limit)

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
		matches.group_match_flg AS match_group_match_flg,
		matches.qualifying_round_flg AS match_qualifying_round_flg,
		matches.final_tournament_flg AS match_final_tournament_flg,
		matches.default_victory_flg AS match_default_victory_flg,
		matches.default_defeat_flg AS match_default_defeat_flg,
		matches.victory_flg AS match_victory_flg,
		matches.draw_flg AS match_draw_flg,
		matches.group_match_victory_flg AS match_group_match_victory_flg,
		matches.opponents_deck_info AS match_opponents_deck_info,
		matches.memo AS match_memo,
		matches.position AS match_position,
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
		// 論理削除済みのgamesを取得しないよう、JOIN条件でdeleted_atを除外する。
		// (Table()+Scan()のためgormのソフトデリートは自動適用されない)
		"LEFT JOIN games ON matches.id = games.match_id AND games.deleted_at IS NULL",
	).Where(
		"matches.id IN (?)", subQuery,
	).Order(
		"matches.created_at DESC, games.created_at ASC",
	).Scan(&results)

	if tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	if len(results) == 0 {
		return nil, apperror.ErrRecordNotFound
	}

	spritesByMatchId, err := findMatchPokemonSpritesByMatchIds(ctx, i.db, matchIdsOf(results))
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	tagsByMatchId, err := findTagsByMatchIds(ctx, i.db, matchIdsOf(results))
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	v := make(map[string]*entity.Match)
	var keys []string

	for _, result := range results {
		match, ok := v[result.MatchID]

		if !ok {
			var games []*entity.Game

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

			pokemonSprites := spritesByMatchId[result.MatchID]

			match := entity.NewMatch(
				result.MatchID,
				result.MatchCreatedAt,
				result.MatchRecordId,
				result.MatchDeckId,
				result.MatchDeckCodeId,
				result.MatchUserId,
				result.MatchOpponentsUserId,
				result.MatchBO3Flg,
				result.MatchGroupMatchFlg,
				result.MatchQualifyingRoundFlg,
				result.MatchFinalTournamentFlg,
				result.MatchDefaultVictoryFlg,
				result.MatchDefaultDefeatFlg,
				result.MatchVictoryFlg,
				result.MatchDrawFlg,
				result.MatchGroupMatchVictoryFlg,
				result.MatchOpponentsDeckInfo,
				result.MatchMemo,
				games,
				pokemonSprites,
			)
			match.Position = result.MatchPosition
			match.Tags = tagsByMatchId[result.MatchID]

			v[result.MatchID] = match
			keys = append(keys, result.MatchID)
		} else {
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

	// created_at DESC 順を維持するため keys を逆順ソートしない
	// subquery により既に DESC 順で取得済み
	var matches []*entity.Match
	for _, key := range keys {
		matches = append(matches, v[key])
	}

	return matches, nil
}

// matchSummaryResult は記録ごとの対戦集計クエリ(FindSummariesByRecordIds)の1行。
//
// has_group_match / has_bo3 は真偽値だが、LEFT JOIN で1件も対戦が無い行では
// 集約結果がNULLになりboolへScanできないため、件数として受け取ってGo側で
// 0件かどうかに落とす。
type matchSummaryResult struct {
	RecordId        string
	Total           int
	Wins            int
	Draws           int
	GroupMatchCount int
	Bo3Count        int
}

func (i *Match) FindSummariesByRecordIds(
	ctx context.Context,
	userId string,
	recordIds []string,
) ([]*entity.MatchSummary, error) {
	// 空指定は集計対象が無いだけでエラーではないため、クエリを投げずに空で返す。
	if len(recordIds) == 0 {
		return []*entity.MatchSummary{}, nil
	}

	var results []matchSummaryResult

	// 認可はこのクエリのWHERE(records.user_id)で完結させる。呼び出し側で
	// 記録の所有者を突き合わせる作りにすると、突き合わせ漏れがそのまま
	// 他人の集計の漏洩になるため、他人の記録はそもそも結果に現れないようにする。
	//
	// matches ではなく records を起点にLEFT JOINするのは、対戦がまだ1件も
	// 紐づいていない自分の記録も total=0 の行として返すため。ここで欠けると
	// webappが記録ごとの取得へフォールバックし、往復を減らす目的を果たせない。
	//
	// 論理削除は FindByRecordId と同じく records / matches の双方で除外する
	// (Table()+Scan() のためgormのソフトデリートは自動適用されない)。
	tx := i.db.Table(
		"records",
	).Select(`
		records.id AS record_id,
		COUNT(matches.id) AS total,
		COUNT(CASE WHEN matches.victory_flg THEN 1 END) AS wins,
		COUNT(CASE WHEN matches.draw_flg THEN 1 END) AS draws,
		COUNT(CASE WHEN matches.group_match_flg THEN 1 END) AS group_match_count,
		COUNT(CASE WHEN matches.bo3_flg THEN 1 END) AS bo3_count`,
	).Joins(
		"LEFT JOIN matches ON records.id = matches.record_id AND matches.deleted_at IS NULL",
	).Where(
		"records.id IN ? AND records.user_id = ? AND records.deleted_at IS NULL",
		recordIds, userId,
	).Group(
		"records.id",
	).Scan(&results)

	if tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	summaryByRecordId := make(map[string]*entity.MatchSummary, len(results))
	for _, result := range results {
		// 引き分けは負けに数えない(webappの集計と定義を揃える)。
		losses := result.Total - result.Wins - result.Draws

		summaryByRecordId[result.RecordId] = entity.NewMatchSummary(
			result.RecordId,
			result.Total,
			result.Wins,
			losses,
			result.Draws,
			result.GroupMatchCount > 0,
			result.Bo3Count > 0,
		)
	}

	// GROUP BY の結果順はDB任せで不定のため、要求された recordIds の順に並べ直す。
	// 取り出した集計はmapから消し、recordIdsに重複があっても二重に返さない。
	summaries := make([]*entity.MatchSummary, 0, len(results))
	for _, recordId := range recordIds {
		summary, ok := summaryByRecordId[recordId]
		if !ok {
			continue
		}

		summaries = append(summaries, summary)
		delete(summaryByRecordId, recordId)
	}

	return summaries, nil
}

func (i *Match) FindLatest(
	ctx context.Context,
	limit int,
) ([]*entity.Match, error) {
	var results []*model.MatchJoinGame

	subQuery := i.db.Table("matches").
		Select("id").
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Limit(limit)

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
		matches.group_match_flg AS match_group_match_flg,
		matches.qualifying_round_flg AS match_qualifying_round_flg,
		matches.final_tournament_flg AS match_final_tournament_flg,
		matches.default_victory_flg AS match_default_victory_flg,
		matches.default_defeat_flg AS match_default_defeat_flg,
		matches.victory_flg AS match_victory_flg,
		matches.draw_flg AS match_draw_flg,
		matches.group_match_victory_flg AS match_group_match_victory_flg,
		matches.opponents_deck_info AS match_opponents_deck_info,
		matches.memo AS match_memo,
		matches.position AS match_position,
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
		// 論理削除済みのgamesを取得しないよう、JOIN条件でdeleted_atを除外する。
		// (Table()+Scan()のためgormのソフトデリートは自動適用されない)
		"LEFT JOIN games ON matches.id = games.match_id AND games.deleted_at IS NULL",
	).Where(
		"matches.id IN (?)", subQuery,
	).Order(
		"matches.created_at DESC, games.created_at ASC",
	).Scan(&results)

	if tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	if len(results) == 0 {
		return nil, apperror.ErrRecordNotFound
	}

	spritesByMatchId, err := findMatchPokemonSpritesByMatchIds(ctx, i.db, matchIdsOf(results))
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	tagsByMatchId, err := findTagsByMatchIds(ctx, i.db, matchIdsOf(results))
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	v := make(map[string]*entity.Match)
	var keys []string

	for _, result := range results {
		match, ok := v[result.MatchID]

		if !ok {
			var games []*entity.Game

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

			pokemonSprites := spritesByMatchId[result.MatchID]

			match := entity.NewMatch(
				result.MatchID,
				result.MatchCreatedAt,
				result.MatchRecordId,
				result.MatchDeckId,
				result.MatchDeckCodeId,
				result.MatchUserId,
				result.MatchOpponentsUserId,
				result.MatchBO3Flg,
				result.MatchGroupMatchFlg,
				result.MatchQualifyingRoundFlg,
				result.MatchFinalTournamentFlg,
				result.MatchDefaultVictoryFlg,
				result.MatchDefaultDefeatFlg,
				result.MatchVictoryFlg,
				result.MatchDrawFlg,
				result.MatchGroupMatchVictoryFlg,
				result.MatchOpponentsDeckInfo,
				result.MatchMemo,
				games,
				pokemonSprites,
			)
			match.Position = result.MatchPosition
			match.Tags = tagsByMatchId[result.MatchID]

			v[result.MatchID] = match
			keys = append(keys, result.MatchID)
		} else {
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
		entity.GroupMatchFlg,
		entity.QualifyingRoundFlg,
		entity.FinalTournamentFlg,
		entity.DefaultVictoryFlg,
		entity.DefaultDefeatFlg,
		entity.VictoryFlg,
		entity.DrawFlg,
		entity.GroupMatchVictoryFlg,
		entity.OpponentsDeckInfo,
		entity.Memo,
	)

	// 同一 record 内で末尾に追加されるよう、現在の最大 position の次を採番する
	var maxPosition sql.NullInt64
	if tx := i.db.Model(&model.Match{}).
		Where("record_id = ? AND deleted_at IS NULL", entity.RecordId).
		Select("MAX(position)").Scan(&maxPosition); tx.Error != nil {
		return tx.Error
	}
	matchModel.Position = int(maxPosition.Int64) + 1

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

	var matchPokemonSpriteModals []*model.MatchPokemonSprite
	for i, pokemonSprite := range entity.PokemonSprites {
		// position が指定されていればスロットとして使う。
		// 未指定(0)は後方互換のため配列インデックスから採番する。
		position := pokemonSprite.Position
		if position == 0 {
			position = uint(i + 1)
		}
		matchPokemonSpriteModals = append(matchPokemonSpriteModals, model.NewMatchPokemonSprite(entity.ID, position, pokemonSprite.ID))
	}

	// 対戦結果本体とタグの付与を1つのトランザクションにまとめられるよう、
	// ctx にトランザクションがあればそれを使う(usecase.Match.Create/Update)。
	return dbFromContext(ctx, i.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(matchModel).Error; err != nil {
			logError(ctx, err)
			return err
		}

		for _, gameModel := range gameModels {
			if err := tx.Save(gameModel).Error; err != nil {
				logError(ctx, err)
				return err
			}
		}

		for _, matchPokemonSpriteModal := range matchPokemonSpriteModals {
			if err := tx.Save(matchPokemonSpriteModal).Error; err != nil {
				logError(ctx, err)
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
		logError(ctx, tx.Error)
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
		entity.GroupMatchFlg,
		entity.QualifyingRoundFlg,
		entity.FinalTournamentFlg,
		entity.DefaultVictoryFlg,
		entity.DefaultDefeatFlg,
		entity.VictoryFlg,
		entity.DrawFlg,
		entity.GroupMatchVictoryFlg,
		entity.OpponentsDeckInfo,
		entity.Memo,
	)
	// position は通常の更新では変更せず、Reorder でのみ変更する
	matchModel.Position = entity.Position

	var matchPokemonSpriteModals []*model.MatchPokemonSprite
	for i, pokemonSprite := range entity.PokemonSprites {
		// position が指定されていればスロットとして使う。
		// 未指定(0)は後方互換のため配列インデックスから採番する。
		position := pokemonSprite.Position
		if position == 0 {
			position = uint(i + 1)
		}
		matchPokemonSpriteModals = append(matchPokemonSpriteModals, model.NewMatchPokemonSprite(entity.ID, position, pokemonSprite.ID))
	}

	// 対戦結果本体とタグの付与を1つのトランザクションにまとめられるよう、
	// ctx にトランザクションがあればそれを使う(usecase.Match.Create/Update)。
	return dbFromContext(ctx, i.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(matchModel).Error; err != nil {
			logError(ctx, err)
			return err
		}

		if tx := tx.Where("match_id = ?", entity.ID).Delete(&model.MatchPokemonSprite{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		for _, matchPokemonSpriteModal := range matchPokemonSpriteModals {
			if err := tx.Save(matchPokemonSpriteModal).Error; err != nil {
				logError(ctx, err)
				return err
			}
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
						logError(ctx, err)
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
						logError(ctx, err)
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
						logError(ctx, err)
						return err
					}
				} else { // 既存のGameを削除
					if tx := tx.Where("id = ?", game.ID).Delete(&model.Game{}); tx.Error != nil {
						logError(ctx, tx.Error)
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
	return dbFromContext(ctx, i.db).Transaction(func(tx *gorm.DB) error {
		if tx := tx.Where("match_id = ?", id).Delete(&model.Game{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		if tx := tx.Where("id = ?", id).Delete(&model.Match{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		return nil
	}, &sql.TxOptions{Isolation: sql.LevelDefault})
}

func (i *Match) Reorder(
	ctx context.Context,
	recordId string,
	orders []*entity.MatchOrder,
) error {
	return dbFromContext(ctx, i.db).Transaction(func(tx *gorm.DB) error {
		var count int64
		if tx := tx.Model(&model.Match{}).
			Where("record_id = ? AND deleted_at IS NULL", recordId).
			Count(&count); tx.Error != nil {
			return tx.Error
		}

		// リクエストに含まれるIDが現在のrecordの未削除match集合と過不足なく一致するか検証
		if int(count) != len(orders) {
			return apperror.ErrInvalidMatchOrder
		}

		for position, order := range orders {
			result := tx.Model(&model.Match{}).
				Where("id = ? AND record_id = ? AND deleted_at IS NULL", order.ID, recordId).
				Updates(map[string]interface{}{
					"position":             position,
					"qualifying_round_flg": order.QualifyingRoundFlg,
					"final_tournament_flg": order.FinalTournamentFlg,
				})

			if result.Error != nil {
				logError(ctx, result.Error)
				return result.Error
			}

			if result.RowsAffected == 0 {
				return apperror.ErrInvalidMatchOrder
			}
		}

		return nil
	}, &sql.TxOptions{Isolation: sql.LevelDefault})
}
