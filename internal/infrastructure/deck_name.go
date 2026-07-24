package infrastructure

import (
	"context"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"golang.org/x/text/width"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

// minAliasRunes は辞書エントリとして採用する正規化後エイリアスの最小文字数。
// 1文字のエイリアス(「x」等)はほぼすべてのデッキ名に部分一致してしまうため除外する。
const minAliasRunes = 2

// NormalizeDeckName はデッキ名(自由入力)を辞書突合用に正規化する。
//
// 正規化ルール:
//  1. 全角英数→半角、半角カナ→全角カナに統一する(width.Fold)。
//     半角濁点・半濁点は結合文字(U+3099/U+309A)になるため NFC で「ﾊﾞ」→「バ」に合成する
//  2. 英字は小文字に統一する
//  3. ひらがなはカタカナに統一する
//  4. 文字(かな・漢字・英字)と数字以外(空白・記号)は除去する。長音「ー」は文字扱いで残る
func NormalizeDeckName(s string) string {
	folded, _, err := transform.String(transform.Chain(width.Fold, norm.NFC), s)
	if err != nil {
		// 変換に失敗するのは不正なバイト列のみ。原文のまま後段の処理を続ける。
		folded = s
	}
	folded = strings.ToLower(folded)

	var b strings.Builder
	b.Grow(len(folded))
	for _, r := range folded {
		if r >= 'ぁ' && r <= 'ゖ' {
			r += 'ァ' - 'ぁ'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}

	return b.String()
}

// deckNameAliasEntry は突合用に正規化済みのエイリアス1件と、その代表スプライト列。
type deckNameAliasEntry struct {
	alias   string      // 正規化済みエイリアス
	sprites []spritePos // 代表スプライト(position ASC、最大2体)
}

// deckNameMatcher はデッキ名から代表スプライトを推測するインメモリ辞書。
// deck_name_aliases(略称・通称)と pokemon_sprites(正式名)を全ロードして組み立てる。
type deckNameMatcher struct {
	entries []deckNameAliasEntry   // エイリアスの文字数降順(同長はエイリアス昇順)で決定的に並ぶ
	cache   map[string][]spritePos // 生のデッキ名→推測結果。週内は同名が多く重複計算を省く
}

// loadDeckNameMatcher は辞書とスプライトマスタを全ロードしてマッチャを組み立てる。
// エイリアス数百件＋マスタ数千件程度を想定しており、リクエスト時の全ロードで足りる。
// 辞書の更新(INSERT/UPDATE)は次リクエストから反映される。
func loadDeckNameMatcher(ctx context.Context, db *gorm.DB) (*deckNameMatcher, error) {
	byAlias, err := loadDeckNameAliasMap(ctx, db, "")
	if err != nil {
		return nil, err
	}

	// 正式名はエイリアス辞書に無いキーだけ取り込む(辞書側の代表2体定義を優先する)。
	var spriteModels []*model.PokemonSprite
	if tx := db.WithContext(ctx).Order("id ASC").Find(&spriteModels); tx.Error != nil {
		return nil, tx.Error
	}
	for _, s := range spriteModels {
		key := NormalizeDeckName(s.Name)
		if len([]rune(key)) < minAliasRunes {
			continue
		}
		if _, ok := byAlias[key]; ok {
			continue
		}
		byAlias[key] = []spritePos{{id: s.ID, position: 1}}
	}

	return buildDeckNameMatcher(byAlias), nil
}

// loadDeckNameAliasMap は deck_name_aliases を読み、正規化キー→スプライト列の対応を返す。
// source が空文字なら全件、指定があればその source のエントリだけを対象にする。
func loadDeckNameAliasMap(ctx context.Context, db *gorm.DB, source string) (map[string][]spritePos, error) {
	query := db.WithContext(ctx).Order("alias ASC, position ASC")
	if source != "" {
		query = query.Where("source = ?", source)
	}

	var aliasModels []*model.DeckNameAlias
	if tx := query.Find(&aliasModels); tx.Error != nil {
		return nil, tx.Error
	}

	// まず生エイリアス単位で position ASC のスプライト列に束ねる。
	spritesByRawAlias := make(map[string][]spritePos)
	rawOrder := make([]string, 0)
	for _, a := range aliasModels {
		if _, ok := spritesByRawAlias[a.Alias]; !ok {
			rawOrder = append(rawOrder, a.Alias)
		}
		spritesByRawAlias[a.Alias] = append(spritesByRawAlias[a.Alias], spritePos{id: a.PokemonSpriteId, position: a.Position})
	}

	// 正規化キーへ変換する。別々の生エイリアスが同一キーに潰れた場合は
	// alias ASC 順の先勝ちで決定的にする。
	byAlias := make(map[string][]spritePos, len(rawOrder))
	for _, raw := range rawOrder {
		key := NormalizeDeckName(raw)
		if len([]rune(key)) < minAliasRunes {
			continue
		}
		if _, ok := byAlias[key]; ok {
			continue
		}
		byAlias[key] = spritesByRawAlias[raw]
	}

	return byAlias, nil
}

// buildDeckNameMatcher は正規化キー→スプライト列の対応からマッチャを組み立てる。
func buildDeckNameMatcher(byAlias map[string][]spritePos) *deckNameMatcher {
	entries := make([]deckNameAliasEntry, 0, len(byAlias))
	for alias, sprites := range byAlias {
		entries = append(entries, deckNameAliasEntry{alias: alias, sprites: sprites})
	}

	// 最長一致を実現するため文字数降順に並べる。同長はエイリアス昇順で決定的にする。
	sort.Slice(entries, func(a, b int) bool {
		la, lb := len([]rune(entries[a].alias)), len([]rune(entries[b].alias))
		if la != lb {
			return la > lb
		}
		return entries[a].alias < entries[b].alias
	})

	return &deckNameMatcher{
		entries: entries,
		cache:   make(map[string][]spritePos),
	}
}

// findDeckNamesByDeckIds は decks.name を一括取得する(スプライト未設定デッキの名前推測用)。
// deck_pokemon_sprites の取得が論理削除を見ないのと平仄を合わせ、論理削除済みデッキも
// 対象に含める(Table 直指定のため gorm の soft delete フィルタは掛からない)。
func findDeckNamesByDeckIds(
	ctx context.Context,
	db *gorm.DB,
	deckIds []string,
) (map[string]string, error) {
	if len(deckIds) == 0 {
		return map[string]string{}, nil
	}

	// バインド引数の並びを決定的にする(呼び出し側の収集順に依存させない)。
	sortedIds := make([]string, len(deckIds))
	copy(sortedIds, deckIds)
	sort.Strings(sortedIds)

	var rows []struct {
		ID   string
		Name string
	}
	if tx := db.WithContext(ctx).
		Table("decks").
		Select("id, name").
		Where("id IN ?", sortedIds).
		Scan(&rows); tx.Error != nil {
		return nil, tx.Error
	}

	names := make(map[string]string, len(rows))
	for _, r := range rows {
		names[r.ID] = r.Name
	}

	return names, nil
}

// guess はデッキ名を正規化し、部分一致する最長のエイリアス1件の代表スプライトを返す。
// ヒットしない場合は nil を返す(呼び出し側で従来どおり集計除外される)。
func (m *deckNameMatcher) guess(rawName string) []spritePos {
	if rawName == "" {
		return nil
	}
	if sprites, ok := m.cache[rawName]; ok {
		return sprites
	}

	normalized := NormalizeDeckName(rawName)

	var result []spritePos
	if normalized != "" {
		for _, e := range m.entries {
			if strings.Contains(normalized, e.alias) {
				result = e.sprites
				break
			}
		}
	}

	m.cache[rawName] = result
	return result
}
