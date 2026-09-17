package usecase

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

/*
 * 参照先の所有者検証(ownership.go)で使う、デッキ・デッキコードのリポジトリの手書きスタブ。
 *
 * 記録・対戦結果・デッキコードの usecase はデッキとデッキコードの所有者を確かめるだけで、
 * それ以外のメソッドは呼ばない。gomock だと呼び出し側のテストごとに EXPECT を書く必要があり、
 * 既存の大量のテストが所有者検証の追加だけで壊れるため、「どのIDでも owner が持ち主」と
 * 答えるスタブで済ませる。所有者検証そのものは ownership_test.go が gomock で検証する。
 *
 * インタフェースを埋め込んでいるので、想定外のメソッドが呼ばれると nil パニックで気づける。
 */

type stubDeckRepository struct {
	repository.DeckInterface
	// owner は FindById が返すデッキの持ち主。空なら誰のものでもない(所有者検証に落ちる)。
	owner string
	// notFound を立てると FindById が ErrRecordNotFound を返す(存在しないデッキの再現)。
	notFound bool
}

func (s stubDeckRepository) FindById(ctx context.Context, id string) (*entity.Deck, error) {
	if s.notFound {
		return nil, apperror.ErrRecordNotFound
	}

	return &entity.Deck{ID: id, UserId: s.owner}, nil
}

type stubDeckCodeRepository struct {
	repository.DeckCodeInterface
	owner    string
	notFound bool
}

func (s stubDeckCodeRepository) FindById(ctx context.Context, id string) (*entity.DeckCode, error) {
	if s.notFound {
		return nil, apperror.ErrRecordNotFound
	}

	return &entity.DeckCode{ID: id, UserId: s.owner}, nil
}
