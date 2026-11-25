package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

/*
 * 書き込み時に参照先(デッキ・デッキコード・記録・自由形式イベント)が本人のものかを確かめる。
 *
 * リクエストの deck_id / deck_code_id / record_id / unofficial_event_id は認可ミドルウェアを通らない
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

// verifyUnofficialEventOwnership は記録に紐づける自由形式イベントが本人のものかを確かめる。
//
// 自由形式イベントは記録と1対1で作られ、記録を消すと一緒に消える(infrastructure の Record.Delete)。
// 他人のイベントを自分の記録に紐づけられると、その記録を消すだけで相手のイベントが消え、
// 相手の記録からイベント名と日付が引けなくなる。イベントのIDは公開記録の応答に含まれる。
func verifyUnofficialEventOwnership(
	ctx context.Context,
	unofficialEventRepository repository.UnofficialEventInterface,
	uid string,
	unofficialEventId string,
) error {
	if unofficialEventId == "" {
		return nil
	}

	unofficialEvent, err := unofficialEventRepository.FindById(ctx, unofficialEventId)
	if err != nil {
		return err
	}
	if unofficialEvent.UserId != uid {
		return apperror.ErrRecordNotFound
	}

	return nil
}
