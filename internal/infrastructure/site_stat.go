package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// siteStatQuery は3つの件数を1往復で数える。
//
// 定義は Grafana の「規模・成長サマリー」の現在値(ユーザ数・対戦結果数・デッキコード数)と同じ。
// トップページの数字とダッシュボードの数字を食い違わせないため、条件を変えるときは両方を揃える。
const siteStatQuery = `
SELECT
	(SELECT COUNT(*) FROM users WHERE deleted_at IS NULL) AS user_count,
	(SELECT COUNT(*) FROM matches WHERE deleted_at IS NULL) AS match_count,
	(SELECT COUNT(*) FROM deck_codes WHERE deleted_at IS NULL AND code IS NOT NULL) AS deck_code_count
`

type SiteStat struct {
	db *gorm.DB
}

func NewSiteStat(
	db *gorm.DB,
) repository.SiteStatInterface {
	return &SiteStat{db}
}

type siteStatRow struct {
	UserCount     int64
	MatchCount    int64
	DeckCodeCount int64
}

func (i *SiteStat) Find(
	ctx context.Context,
) (*entity.SiteStat, error) {
	var row siteStatRow

	if tx := i.db.WithContext(ctx).Raw(siteStatQuery).Scan(&row); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	return entity.NewSiteStat(row.UserCount, row.MatchCount, row.DeckCodeCount), nil
}
