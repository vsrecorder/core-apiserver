package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type Calendar struct {
	db *gorm.DB
}

func NewCalendar(
	db *gorm.DB,
) repository.CalendarInterface {
	return &Calendar{db}
}

// FindByUserId はカレンダーに必要なユーザの全データを取得する。
//
// 記録やデッキの件数に関係なくクエリ本数が一定になるよう、
// 「対象を1クエリでまとめて引いてからメモリ上で紐づける」方針で組む。
// (記録1件ごと・デッキ1件ごとにクエリを足すと、記録が増えたユーザほど
//
//	線形にクエリが増えて破綻するため)
func (i *Calendar) FindByUserId(
	ctx context.Context,
	userId string,
) (*entity.Calendar, error) {
	records, err := i.findRecords(ctx, userId)
	if err != nil {
		return nil, err
	}

	matchesByRecordId, err := i.findMatchesByRecordId(ctx, userId)
	if err != nil {
		return nil, err
	}

	decks, err := i.findDecks(ctx, userId)
	if err != nil {
		return nil, err
	}

	deckCodesByDeckId, err := i.findDeckCodesByDeckId(ctx, userId)
	if err != nil {
		return nil, err
	}

	calendarRecords := make([]*entity.CalendarRecord, 0, len(records))
	for _, record := range records {
		calendarRecords = append(
			calendarRecords,
			entity.NewCalendarRecord(record, matchesByRecordId[record.ID]),
		)
	}

	calendarDecks := make([]*entity.CalendarDeck, 0, len(decks))
	for _, deck := range decks {
		calendarDecks = append(
			calendarDecks,
			entity.NewCalendarDeck(deck, deckCodesByDeckId[deck.ID]),
		)
	}

	officialEvents, err := i.findOfficialEvents(ctx, records)
	if err != nil {
		return nil, err
	}

	unofficialEvents, err := i.findUnofficialEvents(ctx, records)
	if err != nil {
		return nil, err
	}

	// TonamelイベントはDBに無いため、usecase層が外部から補完する。
	return entity.NewCalendar(
		calendarRecords,
		calendarDecks,
		officialEvents,
		unofficialEvents,
		nil,
	), nil
}

func (i *Calendar) findRecords(
	ctx context.Context,
	userId string,
) ([]*entity.Record, error) {
	var recordModels []*model.Record

	if tx := i.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Order("created_at ASC").
		Find(&recordModels); tx.Error != nil {
		return nil, tx.Error
	}

	records := make([]*entity.Record, 0, len(recordModels))
	for _, m := range recordModels {
		record := entity.NewRecord(
			m.ID,
			m.CreatedAt,
			m.OfficialEventId,
			m.TonamelEventId,
			m.FriendId,
			m.UnofficialEventId,
			m.UserId,
			m.DeckId,
			m.DeckCodeId,
			m.EventDate,
			m.PrivateFlg,
			m.IgnoreStatsFlg,
			m.TCGMeisterURL,
			m.Memo,
		)
		records = append(records, record)
	}

	return records, nil
}

// findMatchesByRecordId はユーザの全対戦結果を、対局(games)とスプライトごと
// まとめて取得して record_id ごとに束ねる。
func (i *Calendar) findMatchesByRecordId(
	ctx context.Context,
	userId string,
) (map[string][]*entity.Match, error) {
	var results []*model.MatchJoinGame

	// 対戦結果と対局は1クエリ(LEFT JOIN)で取得する。
	if tx := i.db.WithContext(ctx).Table(
		"matches",
	).Select(`
		matches.id AS match_id,
		matches.created_at AS match_created_at,
		matches.record_id AS match_record_id,
		matches.deck_id AS match_deck_id,
		matches.deck_code_id AS match_deck_code_id,
		matches.user_id AS match_user_id,
		matches.opponents_user_id AS match_opponents_user_id,
		matches.bo3_flg AS match_bo3_flg,
		matches.group_match_flg AS match_group_match_flg,
		matches.group_match_victory_flg AS match_group_match_victory_flg,
		matches.qualifying_round_flg AS match_qualifying_round_flg,
		matches.final_tournament_flg AS match_final_tournament_flg,
		matches.default_victory_flg AS match_default_victory_flg,
		matches.default_defeat_flg AS match_default_defeat_flg,
		matches.victory_flg AS match_victory_flg,
		matches.opponents_deck_info AS match_opponents_deck_info,
		matches.memo AS match_memo,
		matches.position AS match_position,
		games.id AS game_id,
		games.created_at AS game_created_at,
		games.match_id AS game_match_id,
		games.user_id AS game_user_id,
		games.go_first AS game_go_first,
		games.winning_flg AS game_winning_flg,
		games.your_prize_cards AS game_your_prize_cards,
		games.opponents_prize_cards AS game_opponents_prize_cards,
		games.memo AS game_memo`,
	).Joins(`
		LEFT JOIN games
		ON matches.id = games.match_id AND games.deleted_at IS NULL`,
	).Where(
		"matches.user_id = ? AND matches.deleted_at IS NULL",
		userId,
	).Order(
		"matches.record_id ASC, matches.position ASC, games.created_at ASC",
	).Scan(&results); tx.Error != nil {
		return nil, tx.Error
	}

	spritesByMatchId, err := findMatchPokemonSpritesByMatchIds(ctx, i.db, matchIdsOf(results))
	if err != nil {
		return nil, err
	}

	matchById := make(map[string]*entity.Match)
	matchesByRecordId := make(map[string][]*entity.Match)

	for _, result := range results {
		match, ok := matchById[result.MatchID]
		if !ok {
			match = entity.NewMatch(
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
				result.MatchGroupMatchVictoryFlg,
				result.MatchOpponentsDeckInfo,
				result.MatchMemo,
				nil,
				spritesByMatchId[result.MatchID],
			)
			match.Position = result.MatchPosition

			matchById[result.MatchID] = match
			matchesByRecordId[result.MatchRecordId] = append(
				matchesByRecordId[result.MatchRecordId],
				match,
			)
		}

		// LEFT JOIN のため、対局を持たない対戦結果では game_id が空になる。
		if result.GameID == "" {
			continue
		}

		match.Games = append(match.Games, entity.NewGame(
			result.GameID,
			result.GameCreatedAt,
			result.MatchID,
			result.MatchUserId,
			result.GameGoFirst,
			result.GameWinningFlg,
			result.GameYourPrizeCards,
			result.GameOpponentsPrizeCards,
			result.GameMemo,
		))
	}

	return matchesByRecordId, nil
}

// findDecks はアーカイブ済みも含むユーザの全デッキを取得する。
// カレンダーでは作成・アーカイブの両方を活動ログとして扱うため、区別せず全件返す。
func (i *Calendar) findDecks(
	ctx context.Context,
	userId string,
) ([]*entity.Deck, error) {
	var deckModels []*model.Deck

	if tx := i.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Order("created_at ASC").
		Find(&deckModels); tx.Error != nil {
		return nil, tx.Error
	}

	deckIds := make([]string, 0, len(deckModels))
	for _, m := range deckModels {
		deckIds = append(deckIds, m.ID)
	}

	spritesByDeckId, err := findDeckPokemonSpritesByDeckIds(ctx, i.db, deckIds)
	if err != nil {
		return nil, err
	}

	decks := make([]*entity.Deck, 0, len(deckModels))
	for _, m := range deckModels {
		// LatestDeckCode はカレンダーでは使わない(デッキコードは別途全件返す)ため nil でよい。
		decks = append(decks, entity.NewDeck(
			m.ID,
			m.CreatedAt,
			m.ArchivedAt.Time,
			m.UserId,
			m.Name,
			m.PrivateFlg,
			nil,
			spritesByDeckId[m.ID],
		))
	}

	return decks, nil
}

// findDeckCodesByDeckId はユーザの全デッキコードを1クエリで取得して deck_id ごとに束ねる。
func (i *Calendar) findDeckCodesByDeckId(
	ctx context.Context,
	userId string,
) (map[string][]*entity.DeckCode, error) {
	var deckCodeModels []*model.DeckCode

	if tx := i.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Order("created_at ASC").
		Find(&deckCodeModels); tx.Error != nil {
		return nil, tx.Error
	}

	deckCodesByDeckId := make(map[string][]*entity.DeckCode)
	for _, m := range deckCodeModels {
		deckCodesByDeckId[m.DeckId] = append(deckCodesByDeckId[m.DeckId], entity.NewDeckCode(
			m.ID,
			m.CreatedAt,
			m.UserId,
			m.DeckId,
			m.Code,
			m.PrivateCodeFlg,
			m.Memo,
		))
	}

	return deckCodesByDeckId, nil
}

// findOfficialEvents は記録から参照されている公式イベントだけを1クエリで取得する。
func (i *Calendar) findOfficialEvents(
	ctx context.Context,
	records []*entity.Record,
) ([]*entity.OfficialEvent, error) {
	idSet := make(map[uint]struct{})
	for _, record := range records {
		if record.OfficialEventId != 0 {
			idSet[record.OfficialEventId] = struct{}{}
		}
	}

	if len(idSet) == 0 {
		return nil, nil
	}

	ids := make([]uint, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	var eventModels []*model.OfficialEvent
	if tx := i.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&eventModels); tx.Error != nil {
		return nil, tx.Error
	}

	officialEvents := make([]*entity.OfficialEvent, 0, len(eventModels))
	for _, m := range eventModels {
		officialEvents = append(officialEvents, entity.NewOfficialEvent(
			m.ID,
			m.Title,
			m.Address,
			m.Venue,
			m.Date,
			m.StartedAt,
			m.EndedAt,
			m.TypeId,
			m.TypeName,
			m.LeagueTitle,
			m.RegulationTitle,
			m.CSPFlg,
			m.Capacity,
			m.ShopId,
			m.ShopName,
			m.PrefectureId,
			m.PrefectureName,
			m.EnvironmentId,
			m.EnvironmentTitle,
			m.StandardRegulationId,
			m.StandardRegulationMarks,
		))
	}

	return officialEvents, nil
}

// findUnofficialEvents は記録から参照されている自由形式イベントだけを1クエリで取得する。
func (i *Calendar) findUnofficialEvents(
	ctx context.Context,
	records []*entity.Record,
) ([]*entity.UnofficialEvent, error) {
	idSet := make(map[string]struct{})
	for _, record := range records {
		if record.UnofficialEventId != "" {
			idSet[record.UnofficialEventId] = struct{}{}
		}
	}

	if len(idSet) == 0 {
		return nil, nil
	}

	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	var eventModels []*model.UnofficialEvent
	if tx := i.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&eventModels); tx.Error != nil {
		return nil, tx.Error
	}

	unofficialEvents := make([]*entity.UnofficialEvent, 0, len(eventModels))
	for _, m := range eventModels {
		unofficialEvents = append(unofficialEvents, entity.NewUnofficialEvent(
			m.ID,
			m.UserId,
			m.Title,
			m.Date,
		))
	}

	return unofficialEvents, nil
}
