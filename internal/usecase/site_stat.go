package usecase

import (
	"context"
	"sync"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	// siteStatCacheTTL は集計結果を保持する時間。
	// 未認証で叩ける経路なので、リクエストのたびに3テーブルを数えない。
	// トップページの統計カウンタは数分遅れても困らない(webapp 側も5分キャッシュする)。
	siteStatCacheTTL = time.Minute
)

type SiteStatInterface interface {
	Find(
		ctx context.Context,
	) (*entity.SiteStat, error)
}

type SiteStat struct {
	repository repository.SiteStatInterface

	mu        sync.Mutex
	cached    *entity.SiteStat
	expiresAt time.Time
}

func NewSiteStat(
	repository repository.SiteStatInterface,
) SiteStatInterface {
	return &SiteStat{repository: repository}
}

func (u *SiteStat) Find(
	ctx context.Context,
) (*entity.SiteStat, error) {
	now := timeNow()

	if stat := u.load(now); stat != nil {
		return stat, nil
	}

	stat, err := u.repository.Find(ctx)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	u.store(stat, now)

	return stat, nil
}

func (u *SiteStat) load(now time.Time) *entity.SiteStat {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.cached == nil || !now.Before(u.expiresAt) {
		return nil
	}

	return u.cached
}

func (u *SiteStat) store(stat *entity.SiteStat, now time.Time) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.cached = stat
	u.expiresAt = now.Add(siteStatCacheTTL)
}
