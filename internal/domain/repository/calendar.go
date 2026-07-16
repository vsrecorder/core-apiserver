package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type CalendarInterface interface {
	// FindByUserId はカレンダーの組み立てに必要なユーザの全データを取得する。
	//
	// Tonamelイベントは外部サイトから取得するものでDBに存在しないため、ここでは返さない
	// (usecase層が補完する)。返り値の Calendar.TonamelEvents は常に空になる。
	FindByUserId(
		ctx context.Context,
		userId string,
	) (*entity.Calendar, error)
}
