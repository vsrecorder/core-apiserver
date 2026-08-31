package usecase

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

const (
	// landingAtMaxFuture は着地時刻として受け付ける未来方向のずれ。
	// 値は端末のローカル時計で作られるため、多少ずれていても捨てずに残す。
	landingAtMaxFuture = 24 * time.Hour
	// landingAtMaxAge は着地時刻として受け付ける過去方向の上限。
	// Cookie の寿命(90日)より長く取り、それを大きく超える値は壊れているとみなす。
	landingAtMaxAge = 400 * 24 * time.Hour
)

// UserAcquisitionRecordParam は webapp から届く生の流入元。
// すべてクライアント由来のため、正規化を通してから保存する。
type UserAcquisitionRecordParam struct {
	Source      string
	Medium      string
	Campaign    string
	Content     string
	Referrer    string
	LandingPath string
	// LandingAt は RFC3339 の文字列。読めない値は「着地時刻が不明」として捨てる。
	LandingAt string
}

type UserAcquisitionInterface interface {
	// Record は登録直後に流入元を1件保存する。
	// 既に行がある場合は上書きしない(初回タッチ優先)。
	Record(
		ctx context.Context,
		userId string,
		param *UserAcquisitionRecordParam,
	) error
}

type UserAcquisition struct {
	repository repository.UserAcquisitionInterface
}

func NewUserAcquisition(
	repository repository.UserAcquisitionInterface,
) UserAcquisitionInterface {
	return &UserAcquisition{repository}
}

func (u *UserAcquisition) Record(
	ctx context.Context,
	userId string,
	param *UserAcquisitionRecordParam,
) error {
	acquisition := entity.NewUserAcquisition(userId, timeNow())

	acquisition.Source = entity.NormalizeAcquisitionSource(param.Source)
	acquisition.Medium = entity.NormalizeAcquisitionMedium(param.Medium)
	acquisition.Campaign = entity.NormalizeAcquisitionCampaign(param.Campaign)
	acquisition.Content = entity.NormalizeAcquisitionContent(param.Content)
	acquisition.Referrer = entity.NormalizeAcquisitionReferrer(param.Referrer)
	acquisition.LandingPath = entity.NormalizeAcquisitionLandingPath(param.LandingPath)
	acquisition.LandingAt = parseLandingAt(param.LandingAt, timeNow())

	// UTM が無くてもリファラからチャネルを推定する(判明率の底上げ・§3.6)。
	// utm_source が付いている場合は確定値を優先し、推定で上書きしない。
	if acquisition.Source == "" {
		if source, medium := entity.InferAcquisitionSource(acquisition.Referrer); source != "" {
			acquisition.Source = source
			acquisition.Medium = medium
			acquisition.SourceInferred = true
		}
	}

	// 何も判明しなかった行は書かない。users との LEFT JOIN で
	// 「行が無い = 直接流入/不明」として同じように数えられる。
	if acquisition.IsEmpty() {
		return nil
	}

	if err := u.repository.Create(ctx, acquisition); err != nil {
		logError(ctx, err)
		return err
	}

	return nil
}

// parseLandingAt は着地時刻の文字列を時刻に変換する。
// 読めない値・現実的でない値はゼロ値(=保存しない)にする。値は端末のローカル時計で
// 作られるうえ Cookie を書き換えれば任意の値を送れるため、鵜呑みにしない。
//
// 保存先は timestamp without time zone(JST の壁時計)なので、Local へ寄せてから返す。
func parseLandingAt(value string, now time.Time) time.Time {
	if value == "" {
		return time.Time{}
	}

	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}

	if t.After(now.Add(landingAtMaxFuture)) || t.Before(now.Add(-landingAtMaxAge)) {
		return time.Time{}
	}

	return t.Local()
}
