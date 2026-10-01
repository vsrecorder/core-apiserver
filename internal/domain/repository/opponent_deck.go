package repository

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

type OpponentDeckInterface interface {
	// FindByUserId はユーザー自身の対戦結果に付けた相手デッキの組み合わせ(表記 × スプライト)を、
	// 対戦の多い順に返す。表記もスプライトも無い対戦(不戦勝・不戦敗など)は含めない。
	//
	// 一括編集の対象を選ばせるためのもので、集計ではない。集計対象外(ignore_stats_flg)の記録の
	// 対戦も含める(その記録の表記も直せるように)。
	FindByUserId(
		ctx context.Context,
		userId string,
	) ([]*entity.OpponentDeck, error)

	// Replace はユーザー自身の対戦結果のうち、相手デッキが from と同じ組み合わせ(表記と
	// 1体目・2体目のスプライトがすべて一致)のものを、to の表記とスプライトに置き換える。
	// 置き換えた対戦の数を返す。存在しないスプライトIDは apperror.ErrInvalidReference。
	Replace(
		ctx context.Context,
		userId string,
		from *entity.OpponentDeckSpec,
		to *entity.OpponentDeckSpec,
	) (int, error)
}
