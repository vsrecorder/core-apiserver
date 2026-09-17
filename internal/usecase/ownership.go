package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

/*
 * 書き込み時に参照先(デッキ・デッキコード・記録)が本人のものかを確かめる。
 *
 * リクエストの deck_id / deck_code_id / record_id は認可ミドルウェアを通らない
 * (ミドルウェアが見るのはパスの :id だけ)ため、ここで検証しないと他人のデッキに
 * デッキコードを足す・他人の記録に対戦結果を混ぜる、といった書き込みができてしまう。
 * 公開デッキ一覧の「最新デッキコード」を差し替えられる、他人の記録の集計に自分の
 * 対戦が混ざる、参照が残ることで相手がデッキを削除できなくなる、といった実害がある。
 *
 * 他人のものは「存在しない」(apperror.ErrRecordNotFound)として扱い、IDの存在を
 * 教えない(DeckCodePost.Publish と同じ方針)。未指定(空文字)は参照なしなので通す。
 */

func verifyDeckOwnership(
	ctx context.Context,
	deckRepository repository.DeckInterface,
	uid string,
	deckId string,
) error {
	if deckId == "" {
		return nil
	}

	deck, err := deckRepository.FindById(ctx, deckId)
	if err != nil {
		return err
	}
	if deck.UserId != uid {
		return apperror.ErrRecordNotFound
	}

	return nil
}

func verifyDeckCodeOwnership(
	ctx context.Context,
	deckCodeRepository repository.DeckCodeInterface,
	uid string,
	deckCodeId string,
) error {
	if deckCodeId == "" {
		return nil
	}

	deckCode, err := deckCodeRepository.FindById(ctx, deckCodeId)
	if err != nil {
		return err
	}
	if deckCode.UserId != uid {
		return apperror.ErrRecordNotFound
	}

	return nil
}

func verifyRecordOwnership(
	ctx context.Context,
	recordRepository repository.RecordInterface,
	uid string,
	recordId string,
) error {
	if recordId == "" {
		return nil
	}

	record, err := recordRepository.FindById(ctx, recordId)
	if err != nil {
		return err
	}
	if record.UserId != uid {
		return apperror.ErrRecordNotFound
	}

	return nil
}
