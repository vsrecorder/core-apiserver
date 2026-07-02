package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserStatRecentInterface interface {
	// FindRecentMatches は指定ユーザーの直近 count 試合を、対戦日の古い順で返す。
	FindRecentMatches(
		ctx context.Context,
		userId string,
		count int,
		deckId string,
	) ([]*entity.RecentMatch, error)
}
