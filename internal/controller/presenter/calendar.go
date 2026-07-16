package presenter

import (
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func newCalendarPokemonSpriteResponses(
	pokemonSprites []*entity.PokemonSprite,
) []*dto.PokemonSpriteResponse {
	ret := []*dto.PokemonSpriteResponse{}

	for _, pokemonSprite := range pokemonSprites {
		ret = append(ret, &dto.PokemonSpriteResponse{
			ID: pokemonSprite.ID,
		})
	}

	return ret
}

func newCalendarMatchResponses(
	matches []*entity.Match,
) []*dto.CalendarMatchResponse {
	ret := []*dto.CalendarMatchResponse{}

	for _, match := range matches {
		games := []*dto.CalendarGameResponse{}
		for _, game := range match.Games {
			games = append(games, &dto.CalendarGameResponse{
				GoFirst:             game.GoFirst,
				YourPrizeCards:      game.YourPrizeCards,
				OpponentsPrizeCards: game.OpponentsPrizeCards,
			})
		}

		ret = append(ret, &dto.CalendarMatchResponse{
			ID:                match.ID,
			CreatedAt:         match.CreatedAt,
			OpponentsDeckInfo: match.OpponentsDeckInfo,
			DefaultVictoryFlg: match.DefaultVictoryFlg,
			DefaultDefeatFlg:  match.DefaultDefeatFlg,
			VictoryFlg:        match.VictoryFlg,
			Memo:              match.Memo,
			Games:             games,
			PokemonSprites:    newCalendarPokemonSpriteResponses(match.PokemonSprites),
		})
	}

	return ret
}

func newCalendarRecordResponses(
	records []*entity.CalendarRecord,
) []*dto.CalendarRecordResponse {
	ret := []*dto.CalendarRecordResponse{}

	for _, record := range records {
		ret = append(ret, &dto.CalendarRecordResponse{
			ID:                record.Record.ID,
			CreatedAt:         record.Record.CreatedAt,
			OfficialEventId:   record.Record.OfficialEventId,
			TonamelEventId:    record.Record.TonamelEventId,
			UnofficialEventId: record.Record.UnofficialEventId,
			DeckId:            record.Record.DeckId,
			DeckCodeId:        record.Record.DeckCodeId,
			Matches:           newCalendarMatchResponses(record.Matches),
		})
	}

	return ret
}

func newCalendarDeckResponses(
	decks []*entity.CalendarDeck,
) []*dto.CalendarDeckResponse {
	ret := []*dto.CalendarDeckResponse{}

	for _, deck := range decks {
		deckCodes := []*dto.CalendarDeckCodeResponse{}
		for _, deckCode := range deck.DeckCodes {
			deckCodes = append(deckCodes, &dto.CalendarDeckCodeResponse{
				ID:        deckCode.ID,
				CreatedAt: deckCode.CreatedAt,
				Code:      deckCode.Code,
			})
		}

		// アーカイブされていないデッキの ArchivedAt はゼロ値になる。
		// 「西暦1年かどうか」で判定させるのは分かりにくいため、null で返す。
		var archivedAt *time.Time
		if !deck.Deck.ArchivedAt.IsZero() {
			v := deck.Deck.ArchivedAt
			archivedAt = &v
		}

		ret = append(ret, &dto.CalendarDeckResponse{
			ID:             deck.Deck.ID,
			CreatedAt:      deck.Deck.CreatedAt,
			ArchivedAt:     archivedAt,
			Name:           deck.Deck.Name,
			PokemonSprites: newCalendarPokemonSpriteResponses(deck.Deck.PokemonSprites),
			DeckCodes:      deckCodes,
		})
	}

	return ret
}

func newCalendarOfficialEventResponses(
	officialEvents []*entity.OfficialEvent,
) []*dto.OfficialEventResponse {
	ret := []*dto.OfficialEventResponse{}

	for _, officialEvent := range officialEvents {
		// 他の公式イベント応答と同様、日付は時刻を落として返す。
		date := time.Date(
			officialEvent.Date.Year(),
			officialEvent.Date.Month(),
			officialEvent.Date.Day(),
			0, 0, 0, 0, time.Local,
		)

		ret = append(ret, &dto.OfficialEventResponse{
			ID:                      officialEvent.ID,
			Title:                   officialEvent.Title,
			Address:                 officialEvent.Address,
			Venue:                   officialEvent.Venue,
			Date:                    date,
			StartedAt:               officialEvent.StartedAt,
			EndedAt:                 officialEvent.EndedAt,
			TypeId:                  officialEvent.TypeId,
			TypeName:                officialEvent.TypeName,
			LeagueTitle:             officialEvent.LeagueTitle,
			RegulationTitle:         officialEvent.RegulationTitle,
			CSPFlg:                  officialEvent.CSPFlg,
			Capacity:                officialEvent.Capacity,
			ShopId:                  officialEvent.ShopId,
			ShopName:                officialEvent.ShopName,
			PrefectureId:            officialEvent.PrefectureId,
			PrefectureName:          officialEvent.PrefectureName,
			EnvironmentId:           officialEvent.EnvironmentId,
			EnvironmentTitle:        officialEvent.EnvironmentTitle,
			StandardRegulationId:    officialEvent.StandardRegulationId,
			StandardRegulationMarks: officialEvent.StandardRegulationMarks,
		})
	}

	return ret
}

func NewCalendarGetByUserIdResponse(
	calendar *entity.Calendar,
) *dto.CalendarGetByUserIdResponse {
	tonamelEvents := []*dto.CalendarTonamelEventResponse{}
	for _, tonamelEvent := range calendar.TonamelEvents {
		tonamelEvents = append(tonamelEvents, &dto.CalendarTonamelEventResponse{
			ID:    tonamelEvent.ID,
			Title: tonamelEvent.Title,
		})
	}

	unofficialEvents := []*dto.CalendarUnofficialEventResponse{}
	for _, unofficialEvent := range calendar.UnofficialEvents {
		unofficialEvents = append(unofficialEvents, &dto.CalendarUnofficialEventResponse{
			ID:    unofficialEvent.ID,
			Title: unofficialEvent.Title,
		})
	}

	return &dto.CalendarGetByUserIdResponse{
		Records:          newCalendarRecordResponses(calendar.Records),
		Decks:            newCalendarDeckResponses(calendar.Decks),
		OfficialEvents:   newCalendarOfficialEventResponses(calendar.OfficialEvents),
		TonamelEvents:    tonamelEvents,
		UnofficialEvents: unofficialEvents,
	}
}
