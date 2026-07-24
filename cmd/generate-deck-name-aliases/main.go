// generate-deck-name-aliases は、デッキ名エイリアス辞書(deck_name_aliases)を
// 実データの共起から自動生成するバッチ。
//
// 週次デッキ使用率の集計では、スプライト未設定の票をデッキ名から推測して救済している
// (internal/infrastructure/deck_name.go)。その辞書を人手で育てる代わりに、
// 「デッキ名とスプライトを両方登録している記録」を教師データとして代表構成を求め、
// 現在スプライト未設定で除外されている名前ぶんだけエイリアスを作る。
//
// 冪等性: source='auto' の行だけを毎回全削除→再生成する。
// 人が登録した source='manual' の行は読むだけで書き換えない。
// 手動エントリで既に解決できる名前は候補にしない(手動の意図を尊重する)。
//
// 使い方:
//
//	# 生成される候補を確認するだけ(デフォルト。DBは変更しない)
//	go run ./cmd/generate-deck-name-aliases
//
//	# 実際に deck_name_aliases へ反映する
//	go run ./cmd/generate-deck-name-aliases -dry-run=false
//
//	# しきい値と集計期間を調整する
//	go run ./cmd/generate-deck-name-aliases -min-support=20 -min-ratio=0.7 -supply-weeks=24
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

func main() {
	defaults := infrastructure.DefaultDeckNameAliasGeneratorConfig()

	dryRun := flag.Bool("dry-run", true, "true の場合、書き込みは行わず生成される候補の確認のみ行う")
	supplyWeeks := flag.Int("supply-weeks", 12, "教師データ(名前とスプライトが両方ある記録)を遡る週数")
	demandWeeks := flag.Int("demand-weeks", 4, "救済対象(スプライト未設定の票)を遡る週数")
	minSupport := flag.Int("min-support", defaults.MinSupport, "代表構成の支持件数の下限")
	minRatio := flag.Float64("min-ratio", defaults.MinRatio, "代表構成の占有率の下限(0-1)")
	minContributors := flag.Int("min-contributors", defaults.MinContributors, "代表構成を使った実ユーザー数の下限")
	minAliasRunes := flag.Int("min-alias-runes", defaults.MinAliasRunes, "生成するエイリアスの最小文字数")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	db, err := postgres.NewDB(
		os.Getenv("DB_HOSTNAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER_NAME"),
		os.Getenv("DB_USER_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	if err != nil {
		log.Printf("failed to connect database: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	ctx := context.Background()

	// 集計期間は実行日から遡って決める(終端は当日を含めるため翌日 0 時)。
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	to := today.AddDate(0, 0, 1)

	cfg := infrastructure.DeckNameAliasGeneratorConfig{
		SupplyFrom:      today.AddDate(0, 0, -7*(*supplyWeeks)),
		SupplyTo:        to,
		DemandFrom:      today.AddDate(0, 0, -7*(*demandWeeks)),
		DemandTo:        to,
		MinSupport:      *minSupport,
		MinRatio:        *minRatio,
		MinContributors: *minContributors,
		MinAliasRunes:   *minAliasRunes,
	}

	log.Printf(
		"教師データ %s〜 / 救済対象 %s〜 / しきい値: 支持%d件以上・占有率%.0f%%以上・%d人以上・%d文字以上\n",
		cfg.SupplyFrom.Format("2006-01-02"),
		cfg.DemandFrom.Format("2006-01-02"),
		cfg.MinSupport,
		cfg.MinRatio*100,
		cfg.MinContributors,
		cfg.MinAliasRunes,
	)

	candidates, err := infrastructure.GenerateDeckNameAliasCandidates(ctx, db, cfg)
	if err != nil {
		log.Printf("failed to generate deck name alias candidates: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	rescuedVotes := 0
	for _, c := range candidates {
		rescuedVotes += c.DemandVotes
		log.Printf(
			"  %-24s → %-32s 救済%4d票 (支持%d/%d件 %.0f%% %d人)\n",
			c.Alias,
			formatSprites(c.Sprites),
			c.DemandVotes,
			c.Support,
			c.TotalSupply,
			c.Ratio*100,
			c.Contributors,
		)
	}

	log.Printf("候補 %d 件 / 救済見込み %d 票\n", len(candidates), rescuedVotes)

	if *dryRun {
		log.Printf("[dry-run] 書き込みは行いません(-dry-run=false で反映)\n")
		os.Exit(ExitCodeOK)
	}

	saved, err := infrastructure.ReplaceAutoDeckNameAliases(ctx, db, candidates)
	if err != nil {
		log.Printf("failed to replace auto deck name aliases: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	log.Printf("完了: source='auto' を %d 行で再生成しました\n", saved)
	os.Exit(ExitCodeOK)
}

// formatSprites はログ用に代表スプライトを "0006(1) 0018(2)" 形式へ整形する。
func formatSprites(sprites []infrastructure.DeckNameAliasSprite) string {
	parts := make([]string, 0, len(sprites))
	for _, s := range sprites {
		parts = append(parts, fmt.Sprintf("%s(%d)", s.PokemonSpriteId, s.Position))
	}

	return strings.Join(parts, " ")
}
