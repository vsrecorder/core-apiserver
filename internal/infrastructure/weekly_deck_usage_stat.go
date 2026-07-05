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
type weeklyMatchRow struct {
	MatchId    string
	UserId     string
	DeckId     string
	DeckName   string
	DeckInfo   string
	VictoryFlg bool
}

// variantGroup は正規化済みスプライト指紋ごとの集計状態。
type variantGroup struct {
	key     string
	primary string
	sprites []string // 正規化済みの表示用スプライト列
	count   int
	wins    int
	labels  map[string]int // 補助ラベル候補（デッキ名テキスト → 出現数）
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
	// - デッキ名は自分側の票の補助ラベルに使うため LEFT JOIN で取得する。
	query := i.db.Table("matches").
		Select(
			"matches.id AS match_id, " +
				"records.user_id AS user_id, " +
				"records.deck_id AS deck_id, " +
				"COALESCE(decks.name, '') AS deck_name, " +
				"matches.opponents_deck_info AS deck_info, " +
				"matches.victory_flg AS victory_flg",
		).
		Joins("JOIN records ON matches.record_id = records.id").
		Joins("LEFT JOIN decks ON records.deck_id = decks.id").
		Where("records.deleted_at IS NULL AND matches.deleted_at IS NULL")

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

	// 相手デッキの指紋（match_pokemon_sprites）を position 順に取得する。
	spritesByMatch := make(map[string][]string, len(matchIds))
	{
		var spriteModels []*model.MatchPokemonSprite
		if tx := i.db.Where("match_id IN ?", matchIds).Order("position ASC").Find(&spriteModels); tx.Error != nil {
			return nil, tx.Error
		}
		for _, s := range spriteModels {
			spritesByMatch[s.MatchId] = append(spritesByMatch[s.MatchId], s.PokemonSpriteId)
		}
	}

	// 自分デッキの指紋（deck_pokemon_sprites）を position 順に取得する。
	spritesByDeck := make(map[string][]string, len(deckIdSet))
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
			spritesByDeck[s.DeckId] = append(spritesByDeck[s.DeckId], s.PokemonSpriteId)
		}
	}

	groups := make(map[string]*variantGroup)
	order := make([]string, 0)
	contributors := make(map[string]struct{})
	totalVotes := 0

	// addVote は1票を該当する指紋グループへ加算する。
	// won はその指紋（デッキ）が勝ったかどうか。
	addVote := func(spriteIds []string, won bool, label string, userId string) {
		key, primary, normalized := NormalizeFingerprint(spriteIds)
		if key == "" {
			// スプライト未付与は集計不能として除外する。
			return
		}

		g, ok := groups[key]
		if !ok {
			g = &variantGroup{
				key:     key,
				primary: primary,
				sprites: normalized,
				labels:  make(map[string]int),
			}
			groups[key] = g
			order = append(order, key)
		}

		g.count++
		if won {
			g.wins++
		}
		if label != "" {
			g.labels[label]++
		}

		contributors[userId] = struct{}{}
		totalVotes++
	}

	for _, r := range rows {
		// 相手側の票: その指紋が勝った = 記録者が負けた（victory_flg=false）。
		addVote(spritesByMatch[r.MatchId], !r.VictoryFlg, r.DeckInfo, r.UserId)

		// 自分側の票: マッチ単位。記録者が勝てばその指紋の勝ち。
		if r.DeckId != "" {
			addVote(spritesByDeck[r.DeckId], r.VictoryFlg, r.DeckName, r.UserId)
		}
	}

	if totalVotes == 0 {
		return entity.NewWeeklyDeckUsageStat(fromDate, 0, len(contributors), []*entity.DeckUsageVariant{}), nil
	}

	// 出現件数の降順（同数は初出順を維持）にソートする。
	sort.SliceStable(order, func(a, b int) bool {
		return groups[order[a]].count > groups[order[b]].count
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
			"", otherVariantLabel, "", otherCount, usageRate, otherWins, otherCount-otherWins, winRate, []*entity.PokemonSprite{},
		))
	}

	return entity.NewWeeklyDeckUsageStat(fromDate, totalVotes, len(contributors), decks), nil
}

// newVariantEntity は集計済みの variantGroup を entity へ変換する。
func newVariantEntity(g *variantGroup, totalVotes int) *entity.DeckUsageVariant {
	usageRate := float64(g.count) / float64(totalVotes)

	losses := g.count - g.wins
	var winRate float64
	if g.count > 0 {
		winRate = float64(g.wins) / float64(g.count)
	}

	pokemonSprites := make([]*entity.PokemonSprite, 0, len(g.sprites))
	for _, spriteId := range g.sprites {
		pokemonSprites = append(pokemonSprites, entity.NewPokemonSprite(spriteId))
	}

	return entity.NewDeckUsageVariant(
		g.key, mostFrequentLabel(g.labels), g.primary, g.count, usageRate, g.wins, losses, winRate, pokemonSprites,
	)
}

// mostFrequentLabel は補助ラベル候補のうち最頻出のものを返す。同数の場合は辞書順で安定させる。
func mostFrequentLabel(labels map[string]int) string {
	best := ""
	bestCount := 0
	for label, count := range labels {
		if count > bestCount || (count == bestCount && (best == "" || label < best)) {
			best = label
			bestCount = count
		}
	}
	return best
}
