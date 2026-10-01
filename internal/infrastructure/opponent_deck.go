package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

const (
	// maxOpponentDecksPerUser は一覧で返す組み合わせの上限。1 人の対戦の数で頭打ちになるが、
	// 応答が際限なく大きくならないよう歯止めをかける(実際の利用者は多くても数百種類)。
	maxOpponentDecksPerUser = 1000

	// opponentDeckReplaceChunkSize は置き換えで 1 文に載せる対戦の数。
	// 対戦の多い利用者でも、IN 句やスプライトの INSERT のプレースホルダが
	// PostgreSQL の上限(65535)に届かないよう分けて書く。
	opponentDeckReplaceChunkSize = 1000

	// maxOpponentDeckMatches は組み合わせ 1 つについて返す対戦の上限。どの記録で使った表記かを
	// 思い出すためのもので、新しいものから見えれば足りる。
	maxOpponentDeckMatches = 100
)

type OpponentDeck struct {
	db *gorm.DB
}

func NewOpponentDeck(
	db *gorm.DB,
) repository.OpponentDeckInterface {
	return &OpponentDeck{db}
}

type opponentDeckRow struct {
	OpponentsDeckInfo string
	Sprite1           string
	Sprite2           string
	Count             int
	LastEventDate     string
}

// opponentDeckMatches は、ユーザー自身の(論理削除されていない記録の)対戦結果に、
// 1体目・2体目のスプライトを横に並べたもの。一覧と置き換えで同じ束ね方をするために共通にしている。
func opponentDeckMatches(db *gorm.DB, userId string) *gorm.DB {
	return db.Table("matches").
		Joins("JOIN records ON records.id = matches.record_id AND records.deleted_at IS NULL").
		Joins("LEFT JOIN match_pokemon_sprites s1 ON s1.match_id = matches.id AND s1.position = 1").
		Joins("LEFT JOIN match_pokemon_sprites s2 ON s2.match_id = matches.id AND s2.position = 2").
		Where("matches.deleted_at IS NULL AND matches.user_id = ?", userId)
}

// whereOpponentDeckSpec は opponentDeckMatches を、相手デッキが spec と同じ組み合わせ
// (表記と 1体目・2体目のスプライトがすべて一致)の対戦に絞る。対戦の一覧と置き換えで同じ判定を使う。
func whereOpponentDeckSpec(db *gorm.DB, spec *entity.OpponentDeckSpec) *gorm.DB {
	return db.
		Where("matches.opponents_deck_info = ?", spec.OpponentsDeckInfo).
		Where("COALESCE(s1.pokemon_sprite_id, '') = ? AND COALESCE(s2.pokemon_sprite_id, '') = ?",
			spec.SpriteIdAt(1), spec.SpriteIdAt(2))
}

/*
 * FindByUserId は相手デッキの「表記 × 1体目 × 2体目」ごとに、ユーザー自身の対戦を数える。
 *
 * 入力候補(OpponentDeckCandidate)と同じ束ね方だが、こちらは一括編集の対象を選ばせるための
 * 一覧なので、期間で絞らず、集計対象外(ignore_stats_flg)の記録や不戦勝・不戦敗の対戦も含める。
 * 表記ゆれを直したいのは過去の対戦も同じで、ここに出ない対戦は直せなくなるため。
 * 表記もスプライトも無い対戦だけは除く(直す対象として選ぶ意味が無い)。
 *
 * 開催日は TO_CHAR で文字列にして返す。DATE 型を time.Time で受けると UTC 0 時になり、
 * 呼び出し側で JST に直したときに日付がずれる余地を残すため。
 */
func (i *OpponentDeck) FindByUserId(
	ctx context.Context,
	userId string,
) ([]*entity.OpponentDeck, error) {
	var rows []opponentDeckRow

	query := opponentDeckMatches(i.db, userId).
		Select("matches.opponents_deck_info AS opponents_deck_info, " +
			"COALESCE(s1.pokemon_sprite_id, '') AS sprite1, " +
			"COALESCE(s2.pokemon_sprite_id, '') AS sprite2, " +
			"COUNT(*) AS count, " +
			"TO_CHAR(MAX(records.event_date), 'YYYY-MM-DD') AS last_event_date").
		Where("matches.opponents_deck_info <> '' OR s1.pokemon_sprite_id IS NOT NULL OR s2.pokemon_sprite_id IS NOT NULL").
		Group("matches.opponents_deck_info, sprite1, sprite2").
		Order("count DESC, last_event_date DESC, opponents_deck_info ASC").
		Limit(maxOpponentDecksPerUser)

	if tx := query.Scan(&rows); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	ret := make([]*entity.OpponentDeck, 0, len(rows))
	for _, row := range rows {
		sprites := make([]*entity.PokemonSprite, 0, 2)
		if row.Sprite1 != "" {
			sprites = append(sprites, entity.NewPokemonSpriteWithPosition(row.Sprite1, 1))
		}
		if row.Sprite2 != "" {
			sprites = append(sprites, entity.NewPokemonSpriteWithPosition(row.Sprite2, 2))
		}

		ret = append(ret, entity.NewOpponentDeck(row.OpponentsDeckInfo, sprites, row.Count, row.LastEventDate))
	}

	return ret, nil
}

type opponentDeckMatchRow struct {
	MatchId              string
	RecordId             string
	EventDate            string
	OfficialEventId      uint
	TonamelEventId       string
	UnofficialEventId    string
	OfficialEventTitle   string
	TonamelEventTitle    string
	UnofficialEventTitle string
	DeckName             string
	Bo3Flg               bool
	GroupMatchFlg        bool
	GroupMatchVictoryFlg bool
	DefaultVictoryFlg    bool
	DefaultDefeatFlg     bool
	VictoryFlg           bool
	DrawFlg              bool
}

type opponentDeckMatchGameRow struct {
	MatchId    string
	GoFirst    bool
	WinningFlg bool
}

/*
 * FindMatchesBySpec は spec と同じ組み合わせの対戦を、記録の見出し(開催日・イベント名・使用デッキ)と
 * 対局の先攻・後攻と勝敗を付けて返す。一致の判定は Replace と同じで、ここに出た対戦が置き換わる。
 *
 * 並びは記録の開催日の新しい順(同じ日は記録を作った順の新しい順)、同じ記録の中は対戦の並び順。
 * イベント名は記録の種類ごとに別のテーブルにあるので、3 つとも外部結合して Go で選ぶ。
 * Tonamel の大会名は取得済みのもの(tonamel_events)だけを使い、外部サイトへは取りに行かない。
 * 対局は対戦の数にかかわらず 1 文で引く。
 */
func (i *OpponentDeck) FindMatchesBySpec(
	ctx context.Context,
	userId string,
	spec *entity.OpponentDeckSpec,
) ([]*entity.OpponentDeckMatch, error) {
	var rows []opponentDeckMatchRow

	query := whereOpponentDeckSpec(opponentDeckMatches(i.db, userId), spec).
		Joins("LEFT JOIN official_events ON official_events.id = records.official_event_id").
		Joins("LEFT JOIN tonamel_events ON tonamel_events.id = records.tonamel_event_id").
		Joins("LEFT JOIN unofficial_events ON unofficial_events.id = records.unofficial_event_id").
		Joins("LEFT JOIN decks ON decks.id = records.deck_id").
		Select("matches.id AS match_id, " +
			"matches.record_id AS record_id, " +
			"COALESCE(TO_CHAR(records.event_date, 'YYYY-MM-DD'), '') AS event_date, " +
			"COALESCE(records.official_event_id, 0) AS official_event_id, " +
			"COALESCE(records.tonamel_event_id, '') AS tonamel_event_id, " +
			"COALESCE(records.unofficial_event_id, '') AS unofficial_event_id, " +
			"COALESCE(official_events.title, '') AS official_event_title, " +
			"COALESCE(tonamel_events.title, '') AS tonamel_event_title, " +
			"COALESCE(unofficial_events.title, '') AS unofficial_event_title, " +
			"COALESCE(decks.name, '') AS deck_name, " +
			"matches.bo3_flg AS bo3_flg, " +
			"matches.group_match_flg AS group_match_flg, " +
			"matches.group_match_victory_flg AS group_match_victory_flg, " +
			"matches.default_victory_flg AS default_victory_flg, " +
			"matches.default_defeat_flg AS default_defeat_flg, " +
			"matches.victory_flg AS victory_flg, " +
			"matches.draw_flg AS draw_flg").
		Order("records.event_date DESC NULLS LAST, records.created_at DESC, matches.position ASC, matches.created_at ASC").
		Limit(maxOpponentDeckMatches)

	if tx := query.Scan(&rows); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	if len(rows) == 0 {
		return []*entity.OpponentDeckMatch{}, nil
	}

	matchIds := make([]string, 0, len(rows))
	for _, row := range rows {
		matchIds = append(matchIds, row.MatchId)
	}

	var gameRows []opponentDeckMatchGameRow
	if tx := i.db.Table("games").
		Select("match_id, COALESCE(go_first, false) AS go_first, COALESCE(winning_flg, false) AS winning_flg").
		Where("deleted_at IS NULL AND match_id IN ?", matchIds).
		Order("match_id ASC, created_at ASC").
		Scan(&gameRows); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	gamesByMatchId := make(map[string][]*entity.OpponentDeckMatchGame, len(rows))
	for _, game := range gameRows {
		gamesByMatchId[game.MatchId] = append(gamesByMatchId[game.MatchId], &entity.OpponentDeckMatchGame{
			GoFirst:    game.GoFirst,
			WinningFlg: game.WinningFlg,
		})
	}

	ret := make([]*entity.OpponentDeckMatch, 0, len(rows))
	for _, row := range rows {
		eventType, eventTitle := opponentDeckMatchEventOf(row)

		games := gamesByMatchId[row.MatchId]
		if games == nil {
			games = []*entity.OpponentDeckMatchGame{}
		}

		ret = append(ret, &entity.OpponentDeckMatch{
			MatchId:              row.MatchId,
			RecordId:             row.RecordId,
			EventDate:            row.EventDate,
			EventType:            eventType,
			EventTitle:           eventTitle,
			DeckName:             row.DeckName,
			BO3Flg:               row.Bo3Flg,
			GroupMatchFlg:        row.GroupMatchFlg,
			GroupMatchVictoryFlg: row.GroupMatchVictoryFlg,
			DefaultVictoryFlg:    row.DefaultVictoryFlg,
			DefaultDefeatFlg:     row.DefaultDefeatFlg,
			VictoryFlg:           row.VictoryFlg,
			DrawFlg:              row.DrawFlg,
			Games:                games,
		})
	}

	return ret, nil
}

// opponentDeckMatchEventOf は記録のイベントの種類とタイトルを決める。
// 判定の順(公式 → Tonamel → 自由形式)は webapp の記録カードの出し分けと同じ。
func opponentDeckMatchEventOf(row opponentDeckMatchRow) (string, string) {
	switch {
	case row.OfficialEventId != 0:
		return "official", row.OfficialEventTitle
	case row.TonamelEventId != "":
		return "tonamel", row.TonamelEventTitle
	case row.UnofficialEventId != "":
		return "unofficial", row.UnofficialEventTitle
	default:
		return "", ""
	}
}

/*
 * Replace は from と同じ組み合わせの対戦を探し、表記とスプライトを to に置き換える。
 *
 * 一致は表記と 1体目・2体目のスプライトがすべて同じもの(一覧と同じ束ね方)。表記だけで
 * 選ぶと、同じ表記でスプライトを付け分けている対戦まで巻き込むため。
 * 表記の更新・スプライトの削除・追加を 1 つのトランザクションで行い、途中で失敗したときに
 * 「表記だけ変わってスプライトが消えた」状態を残さない。
 */
func (i *OpponentDeck) Replace(
	ctx context.Context,
	userId string,
	from *entity.OpponentDeckSpec,
	to *entity.OpponentDeckSpec,
) (int, error) {
	updated := 0

	err := dbFromContext(ctx, i.db).Transaction(func(tx *gorm.DB) error {
		var ids []string
		if err := whereOpponentDeckSpec(opponentDeckMatches(tx, userId), from).
			Order("matches.id ASC").
			Pluck("matches.id", &ids).Error; err != nil {
			logError(ctx, err)
			return err
		}

		for start := 0; start < len(ids); start += opponentDeckReplaceChunkSize {
			end := min(start+opponentDeckReplaceChunkSize, len(ids))
			chunk := ids[start:end]

			if err := tx.Model(&model.Match{}).
				Where("id IN ?", chunk).
				Update("opponents_deck_info", to.OpponentsDeckInfo).Error; err != nil {
				logError(ctx, err)
				return err
			}

			if err := tx.Where("match_id IN ?", chunk).Delete(&model.MatchPokemonSprite{}).Error; err != nil {
				logError(ctx, err)
				return err
			}

			if len(to.PokemonSprites) == 0 {
				continue
			}

			sprites := make([]*model.MatchPokemonSprite, 0, len(chunk)*len(to.PokemonSprites))
			for _, matchId := range chunk {
				for _, sprite := range to.PokemonSprites {
					sprites = append(sprites, model.NewMatchPokemonSprite(matchId, sprite.Position, sprite.ID))
				}
			}
			if err := tx.Create(&sprites).Error; err != nil {
				logError(ctx, err)
				// 存在しないスプライトIDは外部キーで弾かれる。クライアント起因なので 400 で返せるよう変換する。
				return wrapForeignKeyViolation(err)
			}
		}

		updated = len(ids)
		return nil
	})
	if err != nil {
		return 0, err
	}

	return updated, nil
}
