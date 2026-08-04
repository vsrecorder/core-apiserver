package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type UserDailyActivityInterface interface {
	Record(
		ctx context.Context,
		userId string,
		categories []string,
	) error
}

type UserDailyActivity struct {
	repository repository.UserDailyActivityInterface
}

func NewUserDailyActivity(
	repository repository.UserDailyActivityInterface,
) UserDailyActivityInterface {
	return &UserDailyActivity{repository}
}

// Record は userId の当日の活動を、指定されたカテゴリごとに記録する。
//
// 日付はサーバ側でJSTの当日に確定させる(クライアントの時計・タイムゾーンを信用しない)。
// 未知のカテゴリは黙って捨て、既知のものだけを記録する。webapp と core-apiserver は
// 別々にデプロイされるため、新しいカテゴリを載せた webapp が先に出ても
// 既存の計測が丸ごと落ちないようにする(既知が1つも無いときだけエラーを返す)。
//
// カテゴリ名で分岐しないため、カテゴリを増やしてもこの関数は変更不要。
func (u *UserDailyActivity) Record(
	ctx context.Context,
	userId string,
	categories []string,
) error {
	now := timeNow()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	seen := make(map[string]struct{}, len(categories))
	entities := make([]*entity.UserDailyActivity, 0, len(categories))

	for _, category := range categories {
		if !entity.IsKnownUserDailyActivityCategory(category) {
			continue
		}

		// 同一リクエスト内の重複は1回に丸める。同じ (user_id, date, category) が
		// 1つのINSERT文に複数回現れると ON CONFLICT が自分自身と衝突してエラーになる。
		if _, ok := seen[category]; ok {
			continue
		}
		seen[category] = struct{}{}

		entities = append(entities, entity.NewUserDailyActivity(
			userId,
			today,
			category,
			now,
		))
	}

	if len(entities) == 0 {
		return apperror.ErrNoKnownActivityCategory
	}

	return u.repository.Touch(ctx, entities)
}
