package repository

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type UserAcquisitionInterface interface {
	// Create は流入元を1件保存する。既に行がある場合は何もしない(初回タッチ優先)。
	// 登録直後の1回しか呼ばれない想定だが、webapp のリトライや同時ログインで
	// 二重に届いても後勝ちで上書きしないよう、衝突は無視する。
	Create(
		ctx context.Context,
		entity *entity.UserAcquisition,
	) error

	// SaveSurveyAnswer は登録時アンケートの回答を保存する。行が無ければ回答だけの行を作り、
	// 既に回答が入っている場合は上書きしない(初回の回答を優先)。
	//
	// 流入元(Create)は登録の瞬間・アンケートはその後の着地画面から届くため、通常は
	// 既存行への追記になる。逆順(回答が先)になるのは Create が失敗した場合だけで、
	// そのときの流入元はもともと失われている(Create は登録直後の1回しか呼ばれない)。
	SaveSurveyAnswer(
		ctx context.Context,
		uid string,
		answer string,
		now time.Time,
	) error

	// DeleteByUserId は退会時に、そのユーザの流入元を削除する。
	// このテーブルは論理削除を持たないため行ごと物理削除する。
	DeleteByUserId(
		ctx context.Context,
		uid string,
	) error
}
