package infrastructure

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

// デッキ名エイリアス辞書の自動生成(共起マイニング)。
//
// 考え方:
// 「デッキ名」と「スプライト」を両方登録している記録が、そのまま名前→スプライトの
// 教師データになる。同じ名前で最も多く使われているスプライト構成を代表構成とみなし、
// スプライト未設定で集計から除外されている名前ぶんだけエイリアスを生成する。
// 代表構成が実データの主流構成そのものになるため、推測票が実スプライトの変種と
// 同じ指紋に合流し、指紋の分裂も同時に緩和される。
//
// 生成物は source='auto' として保存し、実行のたびに全削除→再生成する(冪等)。
// 人が登録した source='manual' のエントリは読むだけで書き換えない。

// DeckNameAliasSprite は生成された代表スプライト1体。
type DeckNameAliasSprite struct {
	PokemonSpriteId string
	Position        uint
}

// DeckNameAliasCandidate は自動生成されたエイリアス候補1件。
type DeckNameAliasCandidate struct {
	Alias        string                // 正規化済みエイリアス(alias 列にそのまま入る)
	Sprites      []DeckNameAliasSprite // 代表スプライト(position ASC)
	Support      int                   // 代表構成の支持件数
	TotalSupply  int                   // その名前の教師データ総件数
	Ratio        float64               // Support / TotalSupply(代表構成の占有率)
	Contributors int                   // 代表構成を使った実ユーザー数(重複なし)
	DemandVotes  int                   // 現在スプライト未設定で除外されている票数
}

// 落選理由(DeckNameAliasRejection.Reason)。
const (
	// DeckNameAliasRejectTooShort は正規化後の名前が MinAliasRunes 未満で誤爆を避けて除外。
	DeckNameAliasRejectTooShort = "too_short"
	// DeckNameAliasRejectManualExists は手動辞書で既に解決できるため生成不要。
	DeckNameAliasRejectManualExists = "manual_exists"
	// DeckNameAliasRejectNoSupply は教師データ(名前とスプライトが両方ある記録)が無い。
	DeckNameAliasRejectNoSupply = "no_supply"
	// DeckNameAliasRejectLowSupport は代表構成の支持件数が MinSupport 未満。
	DeckNameAliasRejectLowSupport = "low_support"
	// DeckNameAliasRejectLowRatio は代表構成の占有率が MinRatio 未満(名前が曖昧)。
	DeckNameAliasRejectLowRatio = "low_ratio"
	// DeckNameAliasRejectFewContributors は代表構成を使った実ユーザー数が MinContributors 未満。
	DeckNameAliasRejectFewContributors = "few_contributors"
)

// DeckNameAliasRejection は候補にならなかったデッキ名1件と、その理由・診断値。
// しきい値のチューニングに使う(教師データが無い名前は手動登録の候補にもなる)。
type DeckNameAliasRejection struct {
	Alias        string  // 正規化済みデッキ名
	DemandVotes  int     // この名前で現在除外されている票数(救済し損ねた票)
	Reason       string  // 落選理由(上記定数)
	Support      int     // 最頻構成の支持件数(教師データが無ければ 0)
	TotalSupply  int     // その名前の教師データ総件数(無ければ 0)
	Ratio        float64 // 最頻構成の占有率(無ければ 0)
	Contributors int     // 最頻構成を使った実ユーザー数(無ければ 0)
}

// DeckNameAliasGeneratorConfig は候補生成のしきい値と集計期間。
type DeckNameAliasGeneratorConfig struct {
	// SupplyFrom/SupplyTo は教師データ(名前とスプライトが両方ある記録)の集計期間 [from, to)。
	// 代表構成を安定させるため需要側より長く取る。
	SupplyFrom time.Time
	SupplyTo   time.Time

	// DemandFrom/DemandTo は救済対象(スプライト未設定の票)の集計期間 [from, to)。
	// 「いま効く」エイリアスを優先するため直近に絞る。
	DemandFrom time.Time
	DemandTo   time.Time

	// MinSupport は代表構成の支持件数の下限。偶然の共起を落とす。
	MinSupport int

	// MinRatio は代表構成の占有率の下限。同名で構成が割れる名前を保留する。
	MinRatio float64

	// MinContributors は代表構成を使った実ユーザー数の下限。
	// 1人の命名癖が公開統計を左右しないようにする(minVariantCount と同じ匿名化の思想)。
	MinContributors int

	// MinAliasRunes は生成するエイリアスの最小文字数。
	// 部分一致の誤爆を避けるため、手動運用の下限(minAliasRunes)より保守的にする。
	MinAliasRunes int
}

// DefaultDeckNameAliasGeneratorConfig は既定のしきい値を返す。
// 集計期間は呼び出し側(バッチ)が実行日から決める。
func DefaultDeckNameAliasGeneratorConfig() DeckNameAliasGeneratorConfig {
	return DeckNameAliasGeneratorConfig{
		MinSupport:      10,
		MinRatio:        0.6,
		MinContributors: 3,
		MinAliasRunes:   4,
	}
}

// deckNameSupplyRow は教師データ1件(1マッチ)。Layout は "1:0006,2:0018" 形式。
type deckNameSupplyRow struct {
	Name   string
	UserId string
	Layout string
}

// deckNameDemandRow はスプライト未設定の票の名前ごとの件数。
type deckNameDemandRow struct {
	Name  string
	Count int
}

// deckNameVariantStat は「正規化名 × 指紋」ごとの教師データ集計。
type deckNameVariantStat struct {
	fingerprint  string
	count        int
	users        map[string]struct{}
	layoutCounts map[string]int // 同じ指紋でも position の割り当ては揺れるため最頻を採る
}

// deckNameSupplyStat は正規化名ごとの教師データ集計。
type deckNameSupplyStat struct {
	total    int
	variants map[string]*deckNameVariantStat
}

// GenerateDeckNameAliasCandidates は共起マイニングでエイリアス候補を生成する。
//
// 需要名と完全一致する教師データが無くても、名前の包含関係で教師データへ繋ぐ:
//   - 需要名を含む供給キーのプールで評価する(略称「オロチン」→「オロチンサナ」等の実登録を継ぐ)
//   - プールが空なら、需要名に含まれる最長の供給キー(核)をエイリアスにする
//     (「マリィノオーロンゲシクボ」→「オーロンゲ」)。複数の需要名が同じ核に合流したら票を合算する
//
// 候補にならなかった需要側の名前は、理由つきで rejected に返す(しきい値調整・手動登録の材料)。
// DB への書き込みは行わない(書き込みは ReplaceAutoDeckNameAliases)。
// candidates・rejected はいずれも救済見込み票の多い順(同数は名前昇順)で決定的に並ぶ。
func GenerateDeckNameAliasCandidates(
	ctx context.Context,
	db *gorm.DB,
	cfg DeckNameAliasGeneratorConfig,
) ([]*DeckNameAliasCandidate, []*DeckNameAliasRejection, error) {
	supply, err := aggregateDeckNameSupply(ctx, db, cfg.SupplyFrom, cfg.SupplyTo)
	if err != nil {
		return nil, nil, err
	}

	demand, err := aggregateDeckNameDemand(ctx, db, cfg.DemandFrom, cfg.DemandTo)
	if err != nil {
		return nil, nil, err
	}

	// 人が登録したエントリで既に解決できる名前は生成しない(手動の意図を尊重する)。
	// 一方、正式名(pokemon_sprites.name)でしか解決できない名前は対象に含める。
	// 正式名の解決は1体止まりで実構成と指紋が分裂しがちなため、代表構成で上書きする。
	manualAliases, err := loadDeckNameAliasMap(ctx, db, model.DeckNameAliasSourceManual)
	if err != nil {
		return nil, nil, err
	}
	manualMatcher := buildDeckNameMatcher(manualAliases)

	// 需要の大きい名前から順に評価する(同数は名前昇順で決定的にする)。
	names := make([]string, 0, len(demand))
	for name := range demand {
		names = append(names, name)
	}
	sort.Slice(names, func(a, b int) bool {
		if demand[names[a]] != demand[names[b]] {
			return demand[names[a]] > demand[names[b]]
		}
		return names[a] < names[b]
	})

	candidates := make([]*DeckNameAliasCandidate, 0, len(names))
	rejected := make([]*DeckNameAliasRejection, 0)

	reject := func(name, reason string, best *deckNameVariantStat, total int) {
		r := &DeckNameAliasRejection{
			Alias:       name,
			DemandVotes: demand[name],
			Reason:      reason,
			TotalSupply: total,
		}
		if best != nil {
			r.Support = best.count
			r.Contributors = len(best.users)
			if total > 0 {
				r.Ratio = float64(best.count) / float64(total)
			}
		}
		rejected = append(rejected, r)
	}

	// 需要名に含まれる最長の供給キー(核)を引けるよう、文字数降順(同長は昇順)に並べておく。
	supplyKeys := make([]string, 0, len(supply))
	for k := range supply {
		supplyKeys = append(supplyKeys, k)
	}
	sort.Slice(supplyKeys, func(a, b int) bool {
		la, lb := len([]rune(supplyKeys[a])), len([]rune(supplyKeys[b]))
		if la != lb {
			return la > lb
		}
		return supplyKeys[a] < supplyKeys[b]
	})

	candidateByAlias := make(map[string]*DeckNameAliasCandidate)

	for _, name := range names {
		if len([]rune(name)) < cfg.MinAliasRunes {
			reject(name, DeckNameAliasRejectTooShort, nil, 0)
			continue
		}
		if manualMatcher.guess(name) != nil {
			reject(name, DeckNameAliasRejectManualExists, nil, 0)
			continue
		}

		// エイリアスと教師データのプールを決める。
		// 1. まず name 自身をエイリアスとみなす。エイリアスは突合時に「alias を含む名前」へ
		//    ヒットするため、教師データも name を含む供給キーすべてを束ねて評価する
		//    (完全一致はこの特殊形。略称「オロチン」は「オロチンサナ」等の教師データを継ぐ)。
		// 2. プールが空なら、name に含まれる最長の供給キー(核)へフォールバックする。
		//    「マリィノオーロンゲシクボ」→「オーロンゲ」のように、修飾つきの名前を
		//    教師データのある基本形へ縮めてエイリアス化する。
		alias := name
		pool := pooledSupplyFor(supply, alias)
		if pool.total == 0 {
			core := longestSupplyCore(supplyKeys, name, cfg.MinAliasRunes)
			if core == "" {
				reject(name, DeckNameAliasRejectNoSupply, nil, 0)
				continue
			}
			// 核の手動辞書チェックは不要: guess(name) が nil なら name に含まれる手動
			// エイリアスは存在せず、核 ⊂ name なので核に含まれるものも存在しない。
			alias = core
			pool = pooledSupplyFor(supply, alias)
		}

		// 別の需要名が同じエイリアスに合流したら、救済見込み票だけ合算する。
		if c, ok := candidateByAlias[alias]; ok {
			c.DemandVotes += demand[name]
			continue
		}

		best := bestDeckNameVariant(pool)
		if best == nil {
			reject(name, DeckNameAliasRejectNoSupply, nil, pool.total)
			continue
		}

		// 支持→占有率→人数の順に見て、最初に満たさなかったものを理由にする。
		ratio := float64(best.count) / float64(pool.total)
		switch {
		case best.count < cfg.MinSupport:
			reject(name, DeckNameAliasRejectLowSupport, best, pool.total)
			continue
		case ratio < cfg.MinRatio:
			reject(name, DeckNameAliasRejectLowRatio, best, pool.total)
			continue
		case len(best.users) < cfg.MinContributors:
			reject(name, DeckNameAliasRejectFewContributors, best, pool.total)
			continue
		}

		sprites := parseDeckNameLayout(mostCommonLayout(best.layoutCounts))
		if len(sprites) == 0 {
			// 教師データ集計時に壊れた layout は除いているため通常ここには来ない。
			reject(name, DeckNameAliasRejectNoSupply, best, pool.total)
			continue
		}

		c := &DeckNameAliasCandidate{
			Alias:        alias,
			Sprites:      sprites,
			Support:      best.count,
			TotalSupply:  pool.total,
			Ratio:        ratio,
			Contributors: len(best.users),
			DemandVotes:  demand[name],
		}
		candidateByAlias[alias] = c
		candidates = append(candidates, c)
	}

	// 合流で票が積み上がった候補があるため、最後に票の多い順(同数は名前昇順)へ並べ直す。
	sort.Slice(candidates, func(a, b int) bool {
		if candidates[a].DemandVotes != candidates[b].DemandVotes {
			return candidates[a].DemandVotes > candidates[b].DemandVotes
		}
		return candidates[a].Alias < candidates[b].Alias
	})

	return candidates, rejected, nil
}

// pooledSupplyFor は alias を部分文字列として含むすべての供給キーの教師データを
// 1つに束ねて返す。エイリアスは突合時に「alias を含む名前」へヒットするため、
// プールの範囲を突合の範囲と一致させ、その名前群の実登録を代表構成の根拠にする。
func pooledSupplyFor(supply map[string]*deckNameSupplyStat, alias string) *deckNameSupplyStat {
	pool := &deckNameSupplyStat{variants: make(map[string]*deckNameVariantStat)}

	for key, stat := range supply {
		if !strings.Contains(key, alias) {
			continue
		}

		pool.total += stat.total
		for fp, v := range stat.variants {
			pv, ok := pool.variants[fp]
			if !ok {
				pv = &deckNameVariantStat{
					fingerprint:  fp,
					users:        make(map[string]struct{}),
					layoutCounts: make(map[string]int),
				}
				pool.variants[fp] = pv
			}

			pv.count += v.count
			for u := range v.users {
				pv.users[u] = struct{}{}
			}
			for layout, count := range v.layoutCounts {
				pv.layoutCounts[layout] += count
			}
		}
	}

	return pool
}

// longestSupplyCore は name に部分文字列として含まれる最長の供給キー(核)を返す。
// 無ければ空文字。sortedKeys は文字数降順(同長は昇順)に並んでいること。
func longestSupplyCore(sortedKeys []string, name string, minRunes int) string {
	for _, key := range sortedKeys {
		if len([]rune(key)) < minRunes {
			// 文字数降順なので、これ以降はすべて短すぎる。
			break
		}
		if strings.Contains(name, key) {
			return key
		}
	}

	return ""
}

// ReplaceAutoDeckNameAliases は source='auto' のエントリを全削除して候補で作り直す。
// 冪等で、手動エントリ(source='manual')には触れない。
func ReplaceAutoDeckNameAliases(
	ctx context.Context,
	db *gorm.DB,
	candidates []*DeckNameAliasCandidate,
) (int, error) {
	saved := 0

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if result := tx.Where("source = ?", model.DeckNameAliasSourceAuto).
			Delete(&model.DeckNameAlias{}); result.Error != nil {
			return result.Error
		}

		// 手動エントリと alias が完全一致する候補は主キーが衝突するため取り込まない。
		// (手動側で既に解決できる名前は候補生成の時点で除いているが、念のため防ぐ)
		var existing []*model.DeckNameAlias
		if result := tx.Find(&existing); result.Error != nil {
			return result.Error
		}
		existingAliases := make(map[string]struct{}, len(existing))
		for _, e := range existing {
			existingAliases[e.Alias] = struct{}{}
		}

		rows := make([]*model.DeckNameAlias, 0, len(candidates))
		for _, c := range candidates {
			if _, ok := existingAliases[c.Alias]; ok {
				continue
			}
			for _, s := range c.Sprites {
				rows = append(rows, &model.DeckNameAlias{
					Alias:           c.Alias,
					Position:        s.Position,
					PokemonSpriteId: s.PokemonSpriteId,
					Source:          model.DeckNameAliasSourceAuto,
				})
			}
		}

		if len(rows) == 0 {
			return nil
		}

		if result := tx.Create(&rows); result.Error != nil {
			return result.Error
		}

		saved = len(rows)
		return nil
	})
	if err != nil {
		return 0, err
	}

	return saved, nil
}

// aggregateDeckNameSupply は名前とスプライトが両方ある記録を、正規化名×指紋で集計する。
// 相手デッキ(opponents_deck_info × match_pokemon_sprites)と
// 自分デッキ(decks.name × deck_pokemon_sprites)の双方を教師データにする。
func aggregateDeckNameSupply(
	ctx context.Context,
	db *gorm.DB,
	fromDate time.Time,
	toDate time.Time,
) (map[string]*deckNameSupplyStat, error) {
	stats := make(map[string]*deckNameSupplyStat)

	// スプライトは1マッチにつき複数行あるため、position 順に "位置:ID" を連結して1行に畳む。
	// 指紋は Go 側の NormalizeFingerprint で作り、集計時とまったく同じ正規化に揃える。
	opponentQuery := db.WithContext(ctx).Table("matches").
		Select(
			"matches.opponents_deck_info AS name, " +
				"records.user_id AS user_id, " +
				"string_agg(match_pokemon_sprites.position::text || ':' || match_pokemon_sprites.pokemon_sprite_id, ',' ORDER BY match_pokemon_sprites.position) AS layout",
		).
		Joins("JOIN records ON matches.record_id = records.id").
		Joins("JOIN match_pokemon_sprites ON match_pokemon_sprites.match_id = matches.id").
		Where("records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL AND matches.opponents_deck_info <> ''").
		Group("matches.id, matches.opponents_deck_info, records.user_id")

	ownQuery := db.WithContext(ctx).Table("matches").
		Select(
			"decks.name AS name, " +
				"records.user_id AS user_id, " +
				"string_agg(deck_pokemon_sprites.position::text || ':' || deck_pokemon_sprites.pokemon_sprite_id, ',' ORDER BY deck_pokemon_sprites.position) AS layout",
		).
		Joins("JOIN records ON matches.record_id = records.id").
		Joins("JOIN decks ON decks.id = records.deck_id").
		Joins("JOIN deck_pokemon_sprites ON deck_pokemon_sprites.deck_id = decks.id").
		Where("records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL AND decks.name <> ''").
		Group("matches.id, decks.name, records.user_id")

	for _, query := range []*gorm.DB{opponentQuery, ownQuery} {
		if !fromDate.IsZero() {
			query = query.Where("records.event_date >= ?", fromDate)
		}
		if !toDate.IsZero() {
			query = query.Where("records.event_date < ?", toDate)
		}

		var rows []deckNameSupplyRow
		if tx := query.Scan(&rows); tx.Error != nil {
			return nil, tx.Error
		}

		for _, r := range rows {
			name := NormalizeDeckName(r.Name)
			if name == "" {
				continue
			}

			sprites := parseDeckNameLayout(r.Layout)
			if len(sprites) == 0 {
				continue
			}

			spriteIds := make([]string, 0, len(sprites))
			for _, s := range sprites {
				spriteIds = append(spriteIds, s.PokemonSpriteId)
			}
			fingerprint, _ := NormalizeFingerprint(spriteIds)
			if fingerprint == "" {
				continue
			}

			stat, ok := stats[name]
			if !ok {
				stat = &deckNameSupplyStat{variants: make(map[string]*deckNameVariantStat)}
				stats[name] = stat
			}

			variant, ok := stat.variants[fingerprint]
			if !ok {
				variant = &deckNameVariantStat{
					fingerprint:  fingerprint,
					users:        make(map[string]struct{}),
					layoutCounts: make(map[string]int),
				}
				stat.variants[fingerprint] = variant
			}

			stat.total++
			variant.count++
			variant.users[r.UserId] = struct{}{}
			variant.layoutCounts[r.Layout]++
		}
	}

	return stats, nil
}

// aggregateDeckNameDemand はスプライト未設定で集計から除外されている票を正規化名ごとに数える。
func aggregateDeckNameDemand(
	ctx context.Context,
	db *gorm.DB,
	fromDate time.Time,
	toDate time.Time,
) (map[string]int, error) {
	demand := make(map[string]int)

	opponentQuery := db.WithContext(ctx).Table("matches").
		Select("matches.opponents_deck_info AS name, COUNT(*) AS count").
		Joins("JOIN records ON matches.record_id = records.id").
		Joins("LEFT JOIN match_pokemon_sprites ON match_pokemon_sprites.match_id = matches.id").
		Where("records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL AND matches.opponents_deck_info <> '' AND match_pokemon_sprites.match_id IS NULL").
		Group("matches.opponents_deck_info")

	ownQuery := db.WithContext(ctx).Table("matches").
		Select("decks.name AS name, COUNT(*) AS count").
		Joins("JOIN records ON matches.record_id = records.id").
		Joins("JOIN decks ON decks.id = records.deck_id").
		Joins("LEFT JOIN deck_pokemon_sprites ON deck_pokemon_sprites.deck_id = decks.id").
		Where("records.deleted_at IS NULL AND records.ignore_stats_flg = false AND matches.deleted_at IS NULL AND decks.name <> '' AND deck_pokemon_sprites.deck_id IS NULL").
		Group("decks.name")

	for _, query := range []*gorm.DB{opponentQuery, ownQuery} {
		if !fromDate.IsZero() {
			query = query.Where("records.event_date >= ?", fromDate)
		}
		if !toDate.IsZero() {
			query = query.Where("records.event_date < ?", toDate)
		}

		var rows []deckNameDemandRow
		if tx := query.Scan(&rows); tx.Error != nil {
			return nil, tx.Error
		}

		for _, r := range rows {
			name := NormalizeDeckName(r.Name)
			if name == "" {
				continue
			}
			demand[name] += r.Count
		}
	}

	return demand, nil
}

// bestDeckNameVariant は最も支持の多い指紋を返す。同数は指紋の昇順で決定的にする。
func bestDeckNameVariant(stat *deckNameSupplyStat) *deckNameVariantStat {
	var best *deckNameVariantStat
	for _, v := range stat.variants {
		if best == nil ||
			v.count > best.count ||
			(v.count == best.count && v.fingerprint < best.fingerprint) {
			best = v
		}
	}

	return best
}

// mostCommonLayout は最頻の position 割り当てを返す。同数は文字列昇順で決定的にする。
func mostCommonLayout(layoutCounts map[string]int) string {
	best, bestCount := "", 0
	for layout, count := range layoutCounts {
		if count > bestCount || (count == bestCount && layout < best) {
			best, bestCount = layout, count
		}
	}

	return best
}

// parseDeckNameLayout は "1:0006,2:0018" 形式を position ASC のスプライト列に戻す。
// 壊れた要素は読み飛ばす。
func parseDeckNameLayout(layout string) []DeckNameAliasSprite {
	if layout == "" {
		return nil
	}

	sprites := make([]DeckNameAliasSprite, 0, 2)
	for _, part := range strings.Split(layout, ",") {
		pos, id, ok := strings.Cut(part, ":")
		if !ok || id == "" {
			continue
		}

		position, err := strconv.ParseUint(pos, 10, 32)
		if err != nil || position == 0 {
			continue
		}

		sprites = append(sprites, DeckNameAliasSprite{PokemonSpriteId: id, Position: uint(position)})
	}

	sort.SliceStable(sprites, func(a, b int) bool {
		return sprites[a].Position < sprites[b].Position
	})

	return sprites
}
