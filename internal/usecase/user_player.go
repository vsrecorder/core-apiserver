package usecase

import (
	"context"
	"errors"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

type UserPlayerCreateParam struct {
	UserId         string
	PlayerId       string
	ChallengeToken string
}

func NewUserPlayerCreateParam(
	userId string,
	playerId string,
	challengeToken string,
) *UserPlayerCreateParam {
	return &UserPlayerCreateParam{
		UserId:         userId,
		PlayerId:       playerId,
		ChallengeToken: challengeToken,
	}
}

// UserPlayerVerification は player_id の実在確認結果と、所有権確認チャレンジを
// あわせて持つ。Verify のレスポンスとしてコントローラ層に渡される。
type UserPlayerVerification struct {
	Account   *PlayerAccount
	Challenge *OwnershipChallenge
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

	Create(
		ctx context.Context,
		param *UserPlayerCreateParam,
	) (*entity.UserPlayer, error)

	Verify(
		ctx context.Context,
		uid string,
		playerId string,
	) (*UserPlayerVerification, error)
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

// Verify は player_id の実在を確認し、あわせて所有権確認チャレンジ
// (現在と異なるアバターへの変更依頼)を発行する。
func (u *UserPlayer) Verify(
	ctx context.Context,
	uid string,
	playerId string,
) (*UserPlayerVerification, error) {
	account, err := fetchPlayerAccount(playerId)
	if err != nil {
		return nil, err
	}

	avatar, err := u.avatarRepository.FindRandomExcludingImageURL(ctx, account.AvatarImage)
	if err != nil {
		return nil, err
	}

	token, expiresAt, err := signUserPlayerChallenge(uid, playerId, avatar.ImageURL)
	if err != nil {
		return nil, err
	}

	return &UserPlayerVerification{
		Account: account,
		Challenge: &OwnershipChallenge{
			Token:                   token,
			ChallengeAvatarID:       avatar.ID,
			ChallengeAvatarTitle:    avatar.Title,
			ChallengeAvatarImageURL: avatar.ImageURL,
			ChallengeAvatarDetail:   avatar.Detail,
			ExpiresAt:               expiresAt,
		},
	}, nil
}

func (u *UserPlayer) Create(
	ctx context.Context,
	param *UserPlayerCreateParam,
) (*entity.UserPlayer, error) {
	claims, err := parseUserPlayerChallenge(param.ChallengeToken)
	if err != nil {
		return nil, err
	}

	// チャレンジは発行時と同じユーザー・同じ player_id に対してのみ有効
	if claims.UID != param.UserId || claims.PlayerId != param.PlayerId {
		return nil, apperror.ErrInvalidChallenge
	}

	// 現在のアバターがチャレンジで指定した画像に変更されているか再確認する
	account, err := fetchPlayerAccount(param.PlayerId)
	if err != nil {
		return nil, err
	}
	if account.AvatarImage != claims.ChallengeAvatarImageURL {
		return nil, apperror.ErrOwnershipNotVerified
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
