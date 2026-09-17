package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type OpponentDeckCandidate struct {
	db *gorm.DB
}

func NewOpponentDeckCandidate(
	db *gorm.DB,
) repository.OpponentDeckCandidateInterface {
	return &OpponentDeckCandidate{db}
}

type opponentDeckCandidateRow struct {
	OpponentsDeckInfo string
	Sprite1           string
	Sprite2           string
	Count             int
	LastUsedAt        time.Time
}

/*
 * FindOpponentDeckCandidates は「相手デッキの表記 × 1体目 × 2体目」ごとに対戦結果を数える。
 *
 * webapp が候補を組み立てるときと同じ単位で束ねる。候補の重複排除まで含めてここで行うため、
 * Limit は「候補の数」になる(生の対戦結果を何件見るか、ではない)。対戦結果を件数で切ってから
 * 呼び出し側で重複排除すると、組み合わせが偏っているユーザーほど候補が少なくなってしまう。
 *
 * 数えないもの:
 *   - 不戦勝・不戦敗(相手デッキが無い)、表記が空
 *   - 論理削除された記録・対戦結果
 *   - 集計対象外(ignore_stats_flg)の記録の対戦結果。「この記録で集計を歪めたくない」という
 *     利用者の意思表示なので、戦績・デッキ使用率と同じく除く
 *
 * 記録の private_flg は見ない(集計値だけを返し、誰の対戦かは返さないため)。
 * 同数のときは最近使われたものを先にし、それも同じなら表記順で安定させる。
 */
func (i *OpponentDeckCandidate) FindOpponentDeckCandidates(
	ctx context.Context,
	filter *repository.OpponentDeckCandidateFilter,
) ([]*entity.OpponentDeckCandidate, error) {
	var rows []opponentDeckCandidateRow

	query := i.db.Table("matches").
		Select("matches.opponents_deck_info AS opponents_deck_info, "+
			"COALESCE(s1.pokemon_sprite_id, '') AS sprite1, "+
			"COALESCE(s2.pokemon_sprite_id, '') AS sprite2, "+
			"COUNT(*) AS count, "+
			"MAX(matches.created_at) AS last_used_at").
		Joins("JOIN records ON records.id = matches.record_id AND records.deleted_at IS NULL AND records.ignore_stats_flg = false").
		Joins("LEFT JOIN match_pokemon_sprites s1 ON s1.match_id = matches.id AND s1.position = 1").
		Joins("LEFT JOIN match_pokemon_sprites s2 ON s2.match_id = matches.id AND s2.position = 2").
		Where("matches.deleted_at IS NULL AND matches.default_victory_flg = false AND matches.default_defeat_flg = false AND matches.opponents_deck_info <> ''").
		Where("matches.created_at >= ?", filter.Since)

	// 自身の履歴からの候補。matches.user_id は記録の所有者と同じ値が入る
	// (usecase.Match が認証ユーザーから設定する)。
	if filter.UserId != "" {
		query = query.Where("matches.user_id = ?", filter.UserId)
	}

	query = query.
		Group("matches.opponents_deck_info, sprite1, sprite2").
		Order("count DESC, last_used_at DESC, opponents_deck_info ASC").
		Limit(filter.Limit)

	if tx := query.Scan(&rows); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	ret := make([]*entity.OpponentDeckCandidate, 0, len(rows))
	for _, row := range rows {
		sprites := make([]*entity.PokemonSprite, 0, 2)
		if row.Sprite1 != "" {
			sprites = append(sprites, entity.NewPokemonSpriteWithPosition(row.Sprite1, 1))
		}
		if row.Sprite2 != "" {
			sprites = append(sprites, entity.NewPokemonSpriteWithPosition(row.Sprite2, 2))
		}

		ret = append(ret, entity.NewOpponentDeckCandidate(row.OpponentsDeckInfo, sprites, row.Count))
	}

	return ret, nil
}
