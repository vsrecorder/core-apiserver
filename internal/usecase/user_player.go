package usecase

import (
	"context"
	"errors"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type UserPlayerCreateParam struct {
	UserId   string
	PlayerId string
}

func NewUserPlayerCreateParam(
	userId string,
	playerId string,
) *UserPlayerCreateParam {
	return &UserPlayerCreateParam{
		UserId:   userId,
		PlayerId: playerId,
	}
}

type UserPlayerInterface interface {
	FindByUserId(
		ctx context.Context,
		userId string,
	) (*entity.UserPlayer, error)

	Create(
		ctx context.Context,
		param *UserPlayerCreateParam,
	) (*entity.UserPlayer, error)

	// FindCityleagueResultsByUserId は userId に紐付いたプレイヤーIDの、
	// season(空文字なら現在のシーズン)における入賞を新しい順に返す。
	// 紐付けが無い場合は apperror.ErrRecordNotFound を返す。
	FindCityleagueResultsByUserId(
		ctx context.Context,
		userId string,
		season string,
	) ([]*entity.PlayerCityleagueResult, error)
}

type UserPlayer struct {
	repository                   repository.UserPlayerInterface
	cityleagueResultRepository   repository.CityleagueResultInterface
	championshipSeriesRepository repository.ChampionshipSeriesInterface
	transactionManager           repository.TransactionManager
}

func NewUserPlayer(
	repository repository.UserPlayerInterface,
	cityleagueResultRepository repository.CityleagueResultInterface,
	championshipSeriesRepository repository.ChampionshipSeriesInterface,
	transactionManager repository.TransactionManager,
) UserPlayerInterface {
	return &UserPlayer{
		repository,
		cityleagueResultRepository,
		championshipSeriesRepository,
		transactionManager,
	}
}

// FindCityleagueResultsByUserId は「本人が自己申告したプレイヤーID」の入賞を返す。
// 紐付けは所有権を検証していない(usecase.UserPlayer.Create 参照)ため、返る内容は
// 公開情報である cityleague_results の範囲に限る。ここから記録や他ユーザーの
// 情報へ広げないこと。
func (u *UserPlayer) FindCityleagueResultsByUserId(
	ctx context.Context,
	userId string,
	season string,
) ([]*entity.PlayerCityleagueResult, error) {
	userPlayer, err := u.repository.FindByUserId(ctx, userId)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	fromDate, toDate, err := seasonRange(ctx, u.championshipSeriesRepository, season, timeNow().Local())
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return u.cityleagueResultRepository.FindByPlayerId(ctx, userPlayer.PlayerId, fromDate, toDate)
}

func (u *UserPlayer) FindByUserId(
	ctx context.Context,
	userId string,
) (*entity.UserPlayer, error) {
	userPlayer, err := u.repository.FindByUserId(ctx, userId)

	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return userPlayer, nil
}

// Create は player_id と user_id の紐付けを保存する。
//
// player_id がプレイヤーズクラブに実在するか、また利用者がその持ち主かどうかは
// 確認しない(自己申告として受け入れる)。同じ player_id を複数のユーザーが登録することも
// 許容する。したがってこの紐付けは「本人であることの証明」ではなく、
// あくまで利用者が自分で入力した値として扱うこと。
//
// 一方「1ユーザーにつき有効な紐付けは1件」と「紐付けから1ヶ月は変更不可」は維持する。
func (u *UserPlayer) Create(
	ctx context.Context,
	param *UserPlayerCreateParam,
) (*entity.UserPlayer, error) {
	existing, err := u.repository.FindByUserId(ctx, param.UserId)
	if err != nil && !errors.Is(err, apperror.ErrRecordNotFound) {
		logError(ctx, err)
		return nil, err
	}

	// 現在の紐付けと同じ player_id が指定された場合は変更不要
	if existing != nil && existing.PlayerId == param.PlayerId {
		return existing, nil
	}

	now := timeNow().Local()

	// 既に有効な紐付けがあり、かつ紐付けから1ヶ月経過していない場合は変更不可
	if existing != nil && now.Before(existing.LockedUntil()) {
		return nil, apperror.ErrLocked
	}

	id, err := generateId()
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	userPlayer := entity.NewUserPlayer(
		id,
		now,
		param.UserId,
		param.PlayerId,
	)

	err = u.transactionManager.Do(ctx, func(ctx context.Context) error {
		// 既存の紐付けがある場合(=1ヶ月経過後の変更)は旧レコードをsoft deleteしてから新規作成する
		if existing != nil {
			if err := u.repository.Delete(ctx, existing.ID); err != nil {
				logError(ctx, err)
				return err
			}
		}

		return u.repository.Save(ctx, userPlayer)
	})
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	return userPlayer, nil
}
