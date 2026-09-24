package infrastructure

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

// minVariantCount はプラットフォーム公開時にデッキ変種を個別表示する最小出現数。
// これ未満の変種は匿名化・希薄化対策として「その他」に集約する（DATA_STRATEGY 第5章）。
// 暫定値であり、データ量に応じて調整する。
const minVariantCount = 3

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
// opponents_deck_info（対戦相手デッキ名・フリーテキスト）は指紋計算には使わず、
// スプライト未設定のマッチをデッキ名から推測するフォールバック（deck_name.go）にのみ使う。
type weeklyMatchRow struct {
	MatchId           string
	UserId            string
	DeckId            string
	VictoryFlg        bool
	DrawFlg           bool
	OpponentsDeckInfo string
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
	draws   int
	// members は、この行に束ねられた「束ねる前の変種」ごとの集計。
	// 1体目でまとめる集計（DeckUsageGroupingFirstSprite）でのみ埋まり、
	// 組み合わせ（1体目+2体目）単位の内訳を保持する。memberOrder は出現順。
	// 組み合わせ一致の集計では行そのものが組み合わせ単位なので nil のまま。
	members     map[string]*variantGroup
	memberOrder []string
}

// weeklyVote は集計対象の1票。組み合わせ(順序を無視した指紋)ごとに代表の並びを決めてから
// 数えるため、1周目では指紋と勝敗だけを控える。表示するスプライトの並びは、2周目で
// spriteOrderTally が選んだ代表のものへ揃える。
type weeklyVote struct {
	key  string
	won  bool
	draw bool
}

// spriteOrderTally は同じ組み合わせの中で、スプライトの並び(どちらを1体目に置いたか・
// どの枠に入れたか)ごとの票数を数える。「1体目がリザードン・2体目がピジョット」と
// 「1体目がピジョット・2体目がリザードン」は同じ組み合わせだが、集計では
// 多く使われている方の並びに揃える必要があるため、その多数派をここで決める。
type spriteOrderTally struct {
	counts  map[string]int
	sprites map[string][]spritePos
	best    string
}

func newSpriteOrderTally() *spriteOrderTally {
	return &spriteOrderTally{
		counts:  make(map[string]int),
		sprites: make(map[string][]spritePos),
	}
}

// add は1票ぶんの並びを数え、代表(best)を更新する。
func (t *spriteOrderTally) add(sprites []spritePos) {
	key := spriteOrderKey(sprites)
	if _, ok := t.sprites[key]; !ok {
		t.sprites[key] = sprites
	}
	t.counts[key]++

	// 票数が多い並びを代表にする。同数のときは並びのキーの辞書順で決める。
	// どちらを選んでも根拠が無いため、票が届いた順(＝記録された順)に左右されない
	// 決め方にしておく。同じ週を集計し直しても同じ結果になる。
	if t.best == "" || t.counts[key] > t.counts[t.best] ||
		(t.counts[key] == t.counts[t.best] && key < t.best) {
		t.best = key
	}
}

// canonical はこの組み合わせの代表の並びを返す(position ASC・重複排除済み)。
func (t *spriteOrderTally) canonical() []spritePos {
	return t.sprites[t.best]
}

// spriteOrderKey は並びまで区別するキー。ID集合で作る指紋(NormalizeFingerprint)とは違い、
// どちらが1体目か・どの枠に入っているかで別のキーになる。
func spriteOrderKey(sprites []spritePos) string {
	parts := make([]string, len(sprites))
	for i, s := range sprites {
		parts[i] = s.id + "#" + strconv.FormatUint(uint64(s.position), 10)
	}
	return strings.Join(parts, ",")
}

func (g *variantGroup) winRate() float64 {
	// 引き分けは勝率の分母から除外する(勝ち/(勝ち+負け))。
	decided := g.count - g.draws
	if decided == 0 {
		return 0
	}
	return float64(g.wins) / float64(decided)
}

func (i *WeeklyDeckUsageStat) FindWeeklyDeckUsageStat(
	ctx context.Context,
	fromDate time.Time,
	toDate time.Time,
	grouping entity.DeckUsageGrouping,
) (*entity.WeeklyDeckUsageStat, error) {
	// 未知の値・未指定は既定(組み合わせ一致)へ寄せる。ここで落とさないのは、
	// 集計単位は表示の粒度でしかなく、不正値でレポート全体を見せない理由にはならないため。
	if !grouping.IsValid() {
		grouping = entity.DeckUsageGroupingExact
	}

	stat, err := i.aggregateWeek(ctx, fromDate, toDate, grouping)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	// 前週比較: 変種が1件でもあれば前週 [from-7d, from) を同じ規則で集計し、
	// 指紋で突き合わせて前週の順位・使用率・勝率を付与する(UI の上昇/下降表示用)。
	// 前週も同じ集計単位で集計する(単位が違うと指紋が噛み合わず、全行が NEW になる)。
	if len(stat.Decks) > 0 && !fromDate.IsZero() {
		prev, err := i.aggregateWeek(ctx, fromDate.AddDate(0, 0, -7), fromDate, grouping)
		if err != nil {
			logError(ctx, err)
			return nil, err
		}
		annotatePreviousWeek(stat, prev)
	}

	return stat, nil
}

// aggregateWeek は1週ぶんの使用率統計を集計する(前週比較の情報は付与しない)。
// grouping はどこまでを「同じデッキ」として束ねるか(指紋の作り方)を決める。
func (i *WeeklyDeckUsageStat) aggregateWeek(
	ctx context.Context,
	fromDate time.Time,
	toDate time.Time,
	grouping entity.DeckUsageGrouping,
) (*entity.WeeklyDeckUsageStat, error) {
	var rows []weeklyMatchRow

	// 対象週の全マッチを records と結合して取得する。
	// - 期間フィルタは既存の集計に合わせ records.event_date の半開区間 [from, to)。
	// - 論理削除は deleted_at IS NULL で除外。
	// - private_flg は現状すべて true の予約フラグのため、フィルタ条件には入れない。
	// - ignore_stats_flg が立っている記録は、個人の戦績だけでなくこの公開レポートからも除外する。
	// - 不戦勝/不戦敗（default_victory_flg / default_defeat_flg）は対戦そのものが行われて
	//   いないため除外する。相手側の票は相手デッキ情報もスプライトも空で指紋を作れず元から
	//   落ちていたが、自分側の票（records.deck_id の指紋）は使用数と勝敗に混入していた
	//   （不戦勝=勝ち1票、不戦敗=負け1票）。
	// - レギュレーションはスタンダードの記録だけを集計する。エクストラ・殿堂は使える
	//   カードプールがそもそも違い、同じ「対戦環境」のメタとして混ぜると分布が歪むため。
	// - 退会済みユーザー（users.deleted_at IS NOT NULL）の記録は【意図的に除外しない】。
	//   users と JOIN していないのは書き漏れではない。これは環境（メタ）の分析であって
	//   ユーザーの分析ではなく、「その週に実際に使われたデッキ」の母数は一つでも多いほど
	//   精度が上がるため。記録した人がその後退会しても、その週にそのデッキが使われた
	//   事実は変わらない（2026-08-12 決定）。
	//   ※ 運営用の Grafana ダッシュボードは逆に全パネルで退会者を除外する。あちらは
	//     「ユーザーが使っているか」を測るものなので基準が違う。両者の記録者数が
	//     一致しないのは仕様（grafana/README.md のパネル㉕の項を参照）。
	// - 対戦相手デッキ名（フリーテキスト）は指紋計算には使わないが、スプライト未設定の
	//   マッチをデッキ名から推測するフォールバック（deck_name.go）のために取得する。
	query := i.db.Table("matches").
		Select(
			"matches.id AS match_id, "+
				"records.user_id AS user_id, "+
				"records.deck_id AS deck_id, "+
				"matches.victory_flg AS victory_flg, "+
				"matches.draw_flg AS draw_flg, "+
				"matches.opponents_deck_info AS opponents_deck_info",
		).
		Joins("JOIN records ON matches.record_id = records.id").
		Where(
			"records.deleted_at IS NULL AND records.ignore_stats_flg = false"+
				" AND records.regulation_id = ? AND matches.deleted_at IS NULL"+
				" AND matches.default_victory_flg = false AND matches.default_defeat_flg = false",
			entity.RegulationIdStandard,
		)

	if !fromDate.IsZero() {
		query = query.Where("records.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("records.event_date < ?", toDate)
	}

	query = query.Order("records.event_date ASC")

	if tx := query.Scan(&rows); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	if len(rows) == 0 {
		return entity.NewWeeklyDeckUsageStat(fromDate, grouping, 0, 0, []*entity.DeckUsageVariant{}), nil
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
			logError(ctx, tx.Error)
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
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
		for _, s := range spriteModels {
			spritesByDeck[s.DeckId] = append(spritesByDeck[s.DeckId], spritePos{id: s.PokemonSpriteId, position: s.Position})
		}
	}

	// スプライトが未設定の票はデッキ名からの推測にフォールバックする(deck_name.go)。
	// 推測対象が1件も無い週では、デッキ名・辞書のクエリを一切発行しない。
	deckNames := make(map[string]string)
	var matcher *deckNameMatcher
	{
		needMatcher := false
		nameDeckIds := make([]string, 0)
		seenNameDeckIds := make(map[string]struct{})
		for _, r := range rows {
			if len(spritesByMatch[r.MatchId]) == 0 && r.OpponentsDeckInfo != "" {
				needMatcher = true
			}
			if r.DeckId != "" && len(spritesByDeck[r.DeckId]) == 0 {
				if _, ok := seenNameDeckIds[r.DeckId]; !ok {
					seenNameDeckIds[r.DeckId] = struct{}{}
					nameDeckIds = append(nameDeckIds, r.DeckId)
				}
			}
		}

		if len(nameDeckIds) > 0 {
			names, err := findDeckNamesByDeckIds(ctx, i.db, nameDeckIds)
			if err != nil {
				logError(ctx, err)
				return nil, err
			}
			deckNames = names
			for _, name := range deckNames {
				if name != "" {
					needMatcher = true
					break
				}
			}
		}

		if needMatcher {
			m, err := loadDeckNameMatcher(ctx, i.db)
			if err != nil {
				logError(ctx, err)
				return nil, err
			}
			matcher = m
		}
	}

	// 同じ組み合わせ(1体目・2体目)でも、票によってどちらを1体目に置いたかは揃っていない。
	// 並びが割れたままだと、組み合わせ一致の集計では行に出るアイコンの左右が「最初に来た票」
	// 次第で決まり、1体目でまとめる集計では同じ構築が1体目違いで2つの行に分かれてしまう。
	// そこで票を一度ためて、組み合わせごとに最も多く使われた並びを代表として選び、その並びへ
	// 揃えてから数える。少数派の並びの票は多数派の行へ吸収される。
	votes := make([]weeklyVote, 0, len(rows)*2)
	orders := make(map[string]*spriteOrderTally)
	contributors := make(map[string]struct{})

	// collectVote は1票を控え、その票のスプライトの並びを代表の候補として数える。
	// won はその指紋（デッキ）が勝ったかどうか。
	collectVote := func(sprites []spritePos, won bool, draw bool, userId string) {
		// 表示は position 1/2 の2枠に限られるため、指紋も同じ範囲で計算する。
		// 3体目以降(position>2)を含めると、画面に現れないスプライトが指紋だけを分けて
		// 「見た目が同じ行」が複数並んでしまう(表示と集計の単位を一致させる)。
		visible := make([]spritePos, 0, len(sprites))
		for _, s := range sprites {
			if s.position <= 2 {
				visible = append(visible, s)
			}
		}
		visible = dedupeSprites(visible)

		// 指紋キーは順序非依存(ID集合)で作る。並びの違いはこのキーでは潰れるため、
		// どの並びが多数派かは spriteOrderTally が別に数える。
		spriteIds := make([]string, len(visible))
		for i, s := range visible {
			spriteIds[i] = s.id
		}
		key, _ := NormalizeFingerprint(spriteIds)
		if key == "" {
			// スプライト未付与は集計不能として除外する。
			return
		}

		tally, ok := orders[key]
		if !ok {
			tally = newSpriteOrderTally()
			orders[key] = tally
		}
		tally.add(visible)

		votes = append(votes, weeklyVote{key: key, won: won, draw: draw})
		contributors[userId] = struct{}{}
	}

	for _, r := range rows {
		// 相手側の票: その指紋が勝った = 記録者が負けた（victory_flg=false かつ 引き分けでない）。
		// 引き分けはどちらの勝ちでもないため won=false・draw=true とする。
		// スプライト未設定なら対戦相手デッキ名からの推測にフォールバックする。
		opponentSprites := spritesByMatch[r.MatchId]
		if len(opponentSprites) == 0 && matcher != nil {
			opponentSprites = matcher.guess(r.OpponentsDeckInfo)
		}
		collectVote(opponentSprites, !r.VictoryFlg && !r.DrawFlg, r.DrawFlg, r.UserId)

		// 自分側の票: マッチ単位。記録者が勝てばその指紋の勝ち。
		// スプライト未設定ならデッキ名からの推測にフォールバックする。
		if r.DeckId != "" {
			ownSprites := spritesByDeck[r.DeckId]
			if len(ownSprites) == 0 && matcher != nil {
				ownSprites = matcher.guess(deckNames[r.DeckId])
			}
			collectVote(ownSprites, r.VictoryFlg, r.DrawFlg, r.UserId)
		}
	}

	groups := make(map[string]*variantGroup)
	order := make([]string, 0)
	totalVotes := len(votes)

	for _, v := range votes {
		// 票ごとの並びではなく、その組み合わせの代表の並びを使う。これで「1体目と2体目が逆」
		// の票も、多数派の並びの行として数えられる。
		sprites := orders[v.key].canonical()
		key := v.key

		// 1体目でまとめる集計では、指紋も表示も先頭のスプライト1体だけにする。
		// 先頭は代表の並びの1体目、つまり「その組み合わせで最も多く1体目に置かれた
		// スプライト」であって、票ごとの1体目ではない。代表の並びは position ASC を
		// 保つため、position==1 で抜き出さない点は従来どおり(1枠目が欠けて2枠目だけに
		// 登録されている票（旧データ）を指紋なしとして丸ごと捨ててしまわないため)。
		// 表示用の position は 1 に揃える。この集計単位では行の意味が「1体目が○○の
		// デッキ」であり、元の枠（2枠目）のまま返すと UI が2枠目に描いてしまう。
		//
		// 束ねる前の組み合わせは memberKey / memberSprites に控えて、行の内訳として
		// 別に数える（UI のアコーディオンで展開する）。1体目へ潰した時点で2体目の情報は
		// 指紋からも表示用スプライトからも消えるため、ここで取らないと後から復元できない。
		var memberKey string
		var memberSprites []spritePos
		if grouping == entity.DeckUsageGroupingFirstSprite {
			memberKey = key
			memberSprites = sprites
			sprites = []spritePos{{id: sprites[0].id, position: 1}}
			key, _ = NormalizeFingerprint([]string{sprites[0].id})
		}

		g, ok := groups[key]
		if !ok {
			g = &variantGroup{
				key:     key,
				sprites: sprites,
			}
			groups[key] = g
			order = append(order, key)
		}

		g.count++
		if v.draw {
			g.draws++
		} else if v.won {
			g.wins++
		}

		// 1体目でまとめた行では、束ねる前の組み合わせごとの数も同じ規則で数える。
		if memberKey != "" {
			m, ok := g.members[memberKey]
			if !ok {
				if g.members == nil {
					g.members = make(map[string]*variantGroup)
				}
				m = &variantGroup{
					key:     memberKey,
					sprites: memberSprites,
				}
				g.members[memberKey] = m
				g.memberOrder = append(g.memberOrder, memberKey)
			}

			m.count++
			if v.draw {
				m.draws++
			} else if v.won {
				m.wins++
			}
		}
	}

	if totalVotes == 0 {
		return entity.NewWeeklyDeckUsageStat(fromDate, grouping, 0, len(contributors), []*entity.DeckUsageVariant{}), nil
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
	// 集約した個別変種は otherMembers に保持し、UI のアコーディオンで一覧表示できるようにする。
	var otherCount, otherWins, otherDraws int
	var otherMembers []*entity.DeckUsageVariant

	for _, key := range order {
		g := groups[key]

		if g.count < minVariantCount {
			otherCount += g.count
			otherWins += g.wins
			otherDraws += g.draws
			// order は使用率降順・同数は勝率降順に整列済みなので、内訳もその順序を引き継ぐ。
			// 1体目でまとめた集計では、この内訳はさらに組み合わせ単位の内訳を持つ。
			// 「その他」に落ちた行こそ1体目しか分からないまま消えてしまうため、
			// 何と組んだデッキだったのかを追えるようにそのまま残す
			// (組み合わせ一致の集計では行そのものが組み合わせ単位なので元から nil)。
			otherMembers = append(otherMembers, newVariantEntity(g, totalVotes))
			continue
		}

		decks = append(decks, newVariantEntity(g, totalVotes))
	}

	if otherCount > 0 {
		usageRate := float64(otherCount) / float64(totalVotes)
		// 引き分けは負けに数えず、勝率の分母からも除外する。
		otherLosses := otherCount - otherWins - otherDraws
		var winRate float64
		if decided := otherWins + otherLosses; decided > 0 {
			winRate = float64(otherWins) / float64(decided)
		}
		other := entity.NewDeckUsageVariant(
			"", otherCount, usageRate, otherWins, otherLosses, winRate, []*entity.PokemonSprite{},
		)
		other.Members = otherMembers
		decks = append(decks, other)
	}

	return entity.NewWeeklyDeckUsageStat(fromDate, grouping, totalVotes, len(contributors), decks), nil
}

// annotatePreviousWeek は前週の統計を指紋(スプライトの組み合わせ)で突き合わせ、
// 現在週の各変種に前週の順位・使用率・勝率を付与する。
//
//   - 前週「その他」に集約されていた変種も指紋で比較する(前週のUIは内訳を
//     「その他」行に続く連番で表示しているため、順位もその番号を引き継ぐ)。
//     これが無いと、前週は内訳に表示されていた組み合わせが今週ランクインしたとき
//     NEW と誤表示される
//   - 前週に一度も現れなかった指紋だけが比較なし(NEW 扱い)になる
//   - 「その他」行同士は使用率・勝率のみ比較する(順位を持たない)
//   - 現在週の内訳(Members)には付与しない(一覧の行にだけ意味がある)
func annotatePreviousWeek(current, prev *entity.WeeklyDeckUsageStat) {
	type prevStat struct {
		rank      int // 0 は「順位なし」(その他)
		usageRate float64
		winRate   float64
	}

	prevByFingerprint := make(map[string]prevStat, len(prev.Decks))
	rank := 0
	for _, d := range prev.Decks {
		if d.Fingerprint == "" {
			prevByFingerprint[""] = prevStat{usageRate: d.UsageRate, winRate: d.WinRate}
			// 「その他」は常に末尾のため、ここで rank は個別表示された変種の数。
			// 内訳の連番(その他行の次から)を引き継ぐ。
			for i, m := range d.Members {
				prevByFingerprint[m.Fingerprint] = prevStat{
					rank:      rank + 1 + i,
					usageRate: m.UsageRate,
					winRate:   m.WinRate,
				}
			}
			continue
		}
		rank++
		prevByFingerprint[d.Fingerprint] = prevStat{rank: rank, usageRate: d.UsageRate, winRate: d.WinRate}
	}

	for _, d := range current.Decks {
		p, ok := prevByFingerprint[d.Fingerprint]
		if !ok {
			continue
		}

		if p.rank > 0 {
			rank := p.rank
			d.PreviousRank = &rank
		}
		usageRate, winRate := p.usageRate, p.winRate
		d.PreviousUsageRate = &usageRate
		d.PreviousWinRate = &winRate
	}
}

// newVariantEntity は集計済みの variantGroup を entity へ変換する。
func newVariantEntity(g *variantGroup, totalVotes int) *entity.DeckUsageVariant {
	usageRate := float64(g.count) / float64(totalVotes)
	// 引き分けは負けに数えない(勝率は winRate() が分母から除外する)。
	losses := g.count - g.wins - g.draws

	pokemonSprites := make([]*entity.PokemonSprite, 0, len(g.sprites))
	for _, s := range g.sprites {
		pokemonSprites = append(pokemonSprites, entity.NewPokemonSpriteWithPosition(s.id, s.position))
	}

	variant := entity.NewDeckUsageVariant(
		g.key, g.count, usageRate, g.wins, losses, g.winRate(), pokemonSprites,
	)

	// 1体目でまとめた行には、束ねる前の組み合わせを内訳として持たせる。
	// 並び順は一覧の行と同じ規則(件数の降順・同数は勝率の降順)、
	// 使用率も行と同じ全体件数を分母にする(内訳の合計が行の使用率に一致する)。
	if len(g.memberOrder) > 0 {
		memberOrder := append([]string(nil), g.memberOrder...)
		sort.SliceStable(memberOrder, func(a, b int) bool {
			ma, mb := g.members[memberOrder[a]], g.members[memberOrder[b]]
			if ma.count != mb.count {
				return ma.count > mb.count
			}
			return ma.winRate() > mb.winRate()
		})

		members := make([]*entity.DeckUsageVariant, 0, len(memberOrder))
		for _, key := range memberOrder {
			members = append(members, newVariantEntity(g.members[key], totalVotes))
		}
		variant.Members = members
	}

	return variant
}

// dedupeSprites は表示用スプライト列から ID の重複だけを取り除く。
// 並びは position ASC のまま保ち、枠の gap(1枠目が欠けた旧データ)も潰さない。
func dedupeSprites(sprites []spritePos) []spritePos {
	seen := make(map[string]struct{}, len(sprites))
	ordered := make([]spritePos, 0, len(sprites))
	for _, s := range sprites {
		if _, dup := seen[s.id]; dup {
			continue
		}
		seen[s.id] = struct{}{}
		ordered = append(ordered, s)
	}
	return ordered
}
