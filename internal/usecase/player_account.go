package usecase

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/httpclient"
)

// PlayerAccount はポケモンカードゲーム プレイヤーズクラブの外部APIから取得した
// プレイヤー情報を表す。DBには永続化しない、確認画面表示専用の値。
type PlayerAccount struct {
	PlayerId      string
	Nickname      string
	AvatarImage   string
	CurrentLeague string
	Prefecture    string
}

type playerAccountOtherResponse struct {
	Code   int `json:"code"`
	Player *struct {
		PlayerId      string `json:"player_id"`
		Nickname      string `json:"nickname"`
		AvatarImage   string `json:"avatar_image"`
		CurrentLeague string `json:"current_league"`
		Prefecture    string `json:"prefecture"`
	} `json:"player"`
}

// playerAccountAPIURL はプレイヤーズクラブの実在確認APIのURL。外部サイトへ実通信せずに
// テストできるよう、httptestサーバへ差し替え可能な変数にしている。
var playerAccountAPIURL = "https://players.pokemon-card.com/get_player_account_other"

// fetchPlayerAccount はプレイヤーズクラブの外部APIへ player_id の実在確認を行い、
// 存在すればその情報を返す。存在しない場合は apperror.ErrRecordNotFound を返す。
func fetchPlayerAccount(playerId string) (*PlayerAccount, error) {
	data := url.Values{}
	data.Add("player_id", playerId)

	resp, err := httpclient.PostForm(playerAccountAPIURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// このAPIは player_id が存在しない/マイページが非公開の場合、200以外の
	// ステータス(404など)とともに {"code":404,"message":"..."} 形式のJSONを返す。
	// そのため一律にステータスコードだけでエラー扱いにはせず、まずボディを
	// 読んでJSONの code / player フィールドで存在確認を行う。
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res playerAccountOtherResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}

	if res.Code != http.StatusOK || res.Player == nil {
		return nil, apperror.ErrRecordNotFound
	}

	return &PlayerAccount{
		PlayerId:      res.Player.PlayerId,
		Nickname:      res.Player.Nickname,
		AvatarImage:   res.Player.AvatarImage,
		CurrentLeague: res.Player.CurrentLeague,
		Prefecture:    res.Player.Prefecture,
	}, nil
}
