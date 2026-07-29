package usecase

import (
	"context"
	"errors"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type UserPlayerCreateParam struct {
	UserId            string
	PlayerId          string
	VerificationToken string
}

func NewUserPlayerCreateParam(
	userId string,
	playerId string,
	verificationToken string,
) *UserPlayerCreateParam {
	return &UserPlayerCreateParam{
		UserId:            userId,
		PlayerId:          playerId,
		VerificationToken: verificationToken,
	}
}

type UserPlayerInterface interface {
	FindByUserId(
		ctx context.Context,
		userId string,
	) (*entity.UserPlayer, error)

	// FindLatestPlayerRanking はプレイヤーズクラブの player_id に紐づく
	// 最新のランキング情報(チャンピオンシップポイント等)を返す。
	// ランキング履歴が存在しない場合は apperror.ErrRecordNotFound。
	FindLatestPlayerRanking(
		ctx context.Context,
		playerId string,
	) (*entity.PlayerRanking, error)

	// IssueChallengeAvatar は所有権確認で「これに変更してください」と提示する
	// アバターを1件払い出す。現在のアバターと同じものを提示しても確認にならないため、
	// currentAvatarImage とは異なる画像を返す。
	IssueChallengeAvatar(
		ctx context.Context,
		currentAvatarImage string,
	) (*entity.PokemonAvatar, error)

	Create(
		ctx context.Context,
		param *UserPlayerCreateParam,
	) (*entity.UserPlayer, error)
}

type UserPlayer struct {
	repository              repository.UserPlayerInterface
	avatarRepository        repository.PokemonAvatarInterface
	playerRankingRepository repository.PlayerRankingInterface
	transactionManager      repository.TransactionManager
}

func NewUserPlayer(
	repository repository.UserPlayerInterface,
	avatarRepository repository.PokemonAvatarInterface,
	playerRankingRepository repository.PlayerRankingInterface,
	transactionManager repository.TransactionManager,
) UserPlayerInterface {
	return &UserPlayer{repository, avatarRepository, playerRankingRepository, transactionManager}
}

func (u *UserPlayer) FindByUserId(
	ctx context.Context,
	userId string,
) (*entity.UserPlayer, error) {
	userPlayer, err := u.repository.FindByUserId(ctx, userId)

	if err != nil {
		return nil, err
	}

	return userPlayer, nil
}

func (u *UserPlayer) FindLatestPlayerRanking(
	ctx context.Context,
	playerId string,
) (*entity.PlayerRanking, error) {
	return u.playerRankingRepository.FindLatestByPlayerId(ctx, playerId)
}

func (u *UserPlayer) IssueChallengeAvatar(
	ctx context.Context,
	currentAvatarImage string,
) (*entity.PokemonAvatar, error) {
	return u.avatarRepository.FindRandomExcludingImageURL(ctx, currentAvatarImage)
}

// Create は player_id と user_id の紐付けを保存する。
//
// プレイヤーズクラブでの実在確認と、アバター変更による所有権確認は webapp(BFF)が
// 済ませており、ここではその結果である検証済みトークンの署名を確かめる。
// 紐付けの一意性(1ユーザー1件・1player_id1件)と変更ロックはこのAPIサーバの責務。
func (u *UserPlayer) Create(
	ctx context.Context,
	param *UserPlayerCreateParam,
) (*entity.UserPlayer, error) {
	claims, err := parseUserPlayerVerification(param.VerificationToken)
	if err != nil {
		return nil, err
	}

	// 検証済みトークンは、確認を行ったのと同じユーザー・同じ player_id に対してのみ有効
	if claims.UID != param.UserId || claims.PlayerId != param.PlayerId {
		return nil, apperror.ErrInvalidVerification
	}

	existing, err := u.repository.FindByUserId(ctx, param.UserId)
	if err != nil && !errors.Is(err, apperror.ErrRecordNotFound) {
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

	// player_id が既に別ユーザーの有効な紐付けに使われていないか確認
	inUse, err := u.repository.ExistsActiveByPlayerId(ctx, param.PlayerId)
	if err != nil {
		return nil, err
	}
	if inUse {
		return nil, apperror.ErrAlreadyExists
	}

	id, err := generateId()
	if err != nil {
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
				return err
			}
		}

		return u.repository.Save(ctx, userPlayer)
	})
	if err != nil {
		return nil, err
	}

	return userPlayer, nil
}
