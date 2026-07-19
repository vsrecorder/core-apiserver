package infrastructure

import (
	"context"
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

// minVariantCount はプラットフォーム公開時にデッキ変種を個別表示する最小出現数。
// これ未満の変種は匿名化・希薄化対策として「その他」に集約する（DATA_STRATEGY 第5章）。
// 暫定値であり、データ量に応じて調整する。
const minVariantCount = 5

// otherVariantLabel は minVariantCount 未満の変種をまとめる「その他」枠のラベル。
const otherVariantLabel = "その他"

type WeeklyDeckUsageStat struct {
	db *gorm.DB
}

func NewWeeklyDeckUsageStat(
	db *gorm.DB,
) repository.WeeklyDeckUsageStatInterface {
	return &WeeklyDeckUsageStat{db}
}

// weeklyMatchRow は集計対象週の1マッチ分の情報。
// デッキ名・対戦相手デッキ名（フリーテキスト）は集計に使わないため保持しない。
type weeklyMatchRow struct {
	MatchId    string
	UserId     string
	DeckId     string
	VictoryFlg bool
}

// variantGroup は正規化済みスプライト指紋ごとの集計状態。
// spritePos は表示用スプライトを position 付きで保持する。
// position ASC 順で並び、表示スロット固定(1枠目/2枠目)に使う。
type spritePos struct {
	id       string
	position uint
}

type variantGroup struct {
	key     string
	sprites []spritePos // 表示用スプライト列（重複排除のみ。並び順は position ASC）
	count   int
	wins    int
}

func (g *variantGroup) winRate() float64 {
	if g.count == 0 {
		return 0
	}
	return float64(g.wins) / float64(g.count)
}

func (i *WeeklyDeckUsageStat) FindWeeklyDeckUsageStat(
	ctx context.Context,
	fromDate time.Time,
	toDate time.Time,
) (*entity.WeeklyDeckUsageStat, error) {
	var rows []weeklyMatchRow

	// 対象週の全マッチを records と結合して取得する。
	// - 期間フィルタは既存の集計に合わせ records.event_date の半開区間 [from, to)。
	// - 論理削除は deleted_at IS NULL で除外。
	// - private_flg は現状すべて true の予約フラグのため、フィルタ条件には入れない。
	// - ignore_stats_flg が立っている記録は、個人の戦績だけでなくこの公開レポートからも除外する。
	// - デッキ名・対戦相手デッキ名（フリーテキスト）は集計に使わないため取得しない。
	query := i.db.Table("matches").
		Select(
			"matches.id AS match_id, " +
				"records.user_id AS user_id, " +
				"records.deck_id AS deck_id, " +
				"matches.victory_flg AS victory_flg",
		).
		Joins("JOIN records ON matches.record_id = records.id").
		Where("records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL")

	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("records.event_date < ?", toDate)
	}

	query = query.Order("records.event_date ASC")

	if tx := query.Scan(&rows); tx.Error != nil {
		return nil, tx.Error
	}

	if len(rows) == 0 {
		return entity.NewWeeklyDeckUsageStat(fromDate, 0, 0, []*entity.DeckUsageVariant{}), nil
	}

	// スプライトを一括取得するため、マッチIDとデッキIDを集める。
	matchIds := make([]string, 0, len(rows))
	deckIdSet := make(map[string]struct{})
	for _, r := range rows {
		matchIds = append(matchIds, r.MatchId)
		if r.DeckId != "" {
			deckIdSet[r.DeckId] = struct{}{}
		}
	}

	// 相手デッキの指紋（match_pokemon_sprites）を取得する。順番は集計に使わないため
	// position でのソートは不要だが、既存の取得パターンに合わせて指定しておく。
	spritesByMatch := make(map[string][]spritePos, len(matchIds))
	{
		var spriteModels []*model.MatchPokemonSprite
		if tx := i.db.Where("match_id IN ?", matchIds).Order("position ASC").Find(&spriteModels); tx.Error != nil {
			return nil, tx.Error
		}
		for _, s := range spriteModels {
			spritesByMatch[s.MatchId] = append(spritesByMatch[s.MatchId], spritePos{id: s.PokemonSpriteId, position: s.Position})
		}
	}

	// 自分デッキの指紋（deck_pokemon_sprites）を取得する。
	spritesByDeck := make(map[string][]spritePos, len(deckIdSet))
	if len(deckIdSet) > 0 {
		deckIds := make([]string, 0, len(deckIdSet))
		for id := range deckIdSet {
			deckIds = append(deckIds, id)
		}

		var spriteModels []*model.DeckPokemonSprite
		if tx := i.db.Where("deck_id IN ?", deckIds).Order("position ASC").Find(&spriteModels); tx.Error != nil {
			return nil, tx.Error
		}
		for _, s := range spriteModels {
			spritesByDeck[s.DeckId] = append(spritesByDeck[s.DeckId], spritePos{id: s.PokemonSpriteId, position: s.Position})
		}
	}

	groups := make(map[string]*variantGroup)
	order := make([]string, 0)
	contributors := make(map[string]struct{})
	totalVotes := 0

	// addVote は1票を該当する指紋グループへ加算する。
	// won はその指紋（デッキ）が勝ったかどうか。
	addVote := func(sprites []spritePos, won bool, userId string) {
		// 指紋キーは順序非依存(ID集合)で作る。ordered は元の position ASC 順(重複排除済み)。
		spriteIds := make([]string, len(sprites))
		for i, s := range sprites {
			spriteIds[i] = s.id
		}
		key, _ := NormalizeFingerprint(spriteIds)
		if key == "" {
			// スプライト未付与は集計不能として除外する。
			return
		}

		g, ok := groups[key]
		if !ok {
			// 表示用は position ASC 順のまま ID 重複だけ排除する(gap も保持)
			seen := make(map[string]struct{}, len(sprites))
			ordered := make([]spritePos, 0, len(sprites))
			for _, s := range sprites {
				if _, dup := seen[s.id]; dup {
					continue
				}
				seen[s.id] = struct{}{}
				ordered = append(ordered, s)
			}
			g = &variantGroup{
				key:     key,
				sprites: ordered,
			}
			groups[key] = g
			order = append(order, key)
		}

		g.count++
		if won {
			g.wins++
		}

		contributors[userId] = struct{}{}
		totalVotes++
	}

	for _, r := range rows {
		// 相手側の票: その指紋が勝った = 記録者が負けた（victory_flg=false）。
		addVote(spritesByMatch[r.MatchId], !r.VictoryFlg, r.UserId)

		// 自分側の票: マッチ単位。記録者が勝てばその指紋の勝ち。
		if r.DeckId != "" {
			addVote(spritesByDeck[r.DeckId], r.VictoryFlg, r.UserId)
		}
	}

	if totalVotes == 0 {
		return entity.NewWeeklyDeckUsageStat(fromDate, 0, len(contributors), []*entity.DeckUsageVariant{}), nil
	}

	// 使用率（count）の降順。使用率が同じ場合は勝率の降順で順位を決める。
	sort.SliceStable(order, func(a, b int) bool {
		ga, gb := groups[order[a]], groups[order[b]]
		if ga.count != gb.count {
			return ga.count > gb.count
		}
		return ga.winRate() > gb.winRate()
	})

	decks := make([]*entity.DeckUsageVariant, 0, len(order))

	// minVariantCount 未満の変種は「その他」に集約する。
	var otherCount, otherWins int

	for _, key := range order {
		g := groups[key]

		if g.count < minVariantCount {
			otherCount += g.count
			otherWins += g.wins
			continue
		}

		decks = append(decks, newVariantEntity(g, totalVotes))
	}

	if otherCount > 0 {
		usageRate := float64(otherCount) / float64(totalVotes)
		winRate := float64(otherWins) / float64(otherCount)
		decks = append(decks, entity.NewDeckUsageVariant(
			"", otherCount, usageRate, otherWins, otherCount-otherWins, winRate, []*entity.PokemonSprite{},
		))
	}

	return entity.NewWeeklyDeckUsageStat(fromDate, totalVotes, len(contributors), decks), nil
}

// newVariantEntity は集計済みの variantGroup を entity へ変換する。
func newVariantEntity(g *variantGroup, totalVotes int) *entity.DeckUsageVariant {
	usageRate := float64(g.count) / float64(totalVotes)
	losses := g.count - g.wins

	pokemonSprites := make([]*entity.PokemonSprite, 0, len(g.sprites))
	for _, s := range g.sprites {
		pokemonSprites = append(pokemonSprites, entity.NewPokemonSpriteWithPosition(s.id, s.position))
	}

	return entity.NewDeckUsageVariant(
		g.key, g.count, usageRate, g.wins, losses, g.winRate(), pokemonSprites,
	)
}
