package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

// deckCardAPIDefaultBaseURL は deckcard-api の既定の接続先。
// 本番の nginx は /api/v1beta/deckcards を deckcard-api へ振り分けているため、
// 専用の環境変数(DECKCARD_API_BASE_URL)が無ければ公開ドメイン経由で引く。
const deckCardAPIDefaultBaseURL = "https://vsrecorder.mobi"

// deckCardAceSpecPathFormat は ACE SPEC 判定のパス。テストで httptest サーバへ
// 差し替えるのはベースURL側(NewDeckCard の引数)で行う。
const deckCardAceSpecPathFormat = "/api/v1beta/deckcards/%s/acespec"

// deckCardAPITimeout は deckcard-api への問い合わせの上限。
//
// そのデッキコードを deckcard-api が初めて見るときは公式サイトからの取得が挟まり、
// 数秒かかることがある。ここで諦めると「ACE SPEC なし」として空のまま保存され、
// 表示だけでなく OGP 画像・ACE SPEC での絞り込みにも効き続けるため、公開操作が
// 多少長くなっても取りこぼさないほうを取る(共通の httpclient と同じ10秒)。
const deckCardAPITimeout = 10 * time.Second

// deckCardHTTPClient は deckcard-api 用の HTTP クライアント。
var deckCardHTTPClient = &http.Client{Timeout: deckCardAPITimeout}

// DeckCardAPIBaseURLFromEnv は環境変数 DECKCARD_API_BASE_URL(未設定なら既定値)を返す。
// godotenv.Load() の後に呼ぶ必要があるため、パッケージ変数ではなく関数にしている。
func DeckCardAPIBaseURLFromEnv() string {
	if v := strings.TrimRight(os.Getenv("DECKCARD_API_BASE_URL"), "/"); v != "" {
		return v
	}

	return deckCardAPIDefaultBaseURL
}

// DeckCard は deckcard-api(別サービス)への問い合わせ。
type DeckCard struct {
	baseURL string
}

func NewDeckCard(baseURL string) repository.DeckCardInterface {
	return &DeckCard{baseURL: strings.TrimRight(baseURL, "/")}
}

// deckCardAceSpecResponse は deckcard-api の acespec 応答(webapp の AcespecType と同じ形)。
// card_id は数値(例: 46197)で返るため、文字列としても数値としても受けられる json.Number にする。
type deckCardAceSpecResponse struct {
	CardId   json.Number `json:"card_id"`
	CardName string      `json:"card_name"`
	ImageURL string      `json:"image_url"`
}

// FindAceSpec はデッキコードの ACE SPEC を deckcard-api に問い合わせる。
// 204(該当なし)は nil, nil。それ以外の失敗は error を返し、呼び出し側(usecase)が
// 公開処理を止めずに「判定なし」として扱う。
func (i *DeckCard) FindAceSpec(
	ctx context.Context,
	deckCode string,
) (*entity.AceSpecCard, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, i.baseURL+fmt.Sprintf(deckCardAceSpecPathFormat, url.PathEscape(deckCode)), nil)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	res, err := deckCardHTTPClient.Do(req)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusNoContent:
		return nil, nil
	case http.StatusOK:
		var body deckCardAceSpecResponse
		decoder := json.NewDecoder(res.Body)
		decoder.UseNumber()
		if err := decoder.Decode(&body); err != nil {
			logError(ctx, err)
			return nil, err
		}
		// deckcard-api は該当なしを 204 で返すが、念のため空の応答も「なし」として扱う。
		if body.CardId.String() == "" {
			return nil, nil
		}

		return &entity.AceSpecCard{CardId: body.CardId.String(), CardName: body.CardName, ImageURL: body.ImageURL}, nil
	default:
		err := fmt.Errorf("deckcard-api responded with %d", res.StatusCode)
		logError(ctx, err)
		return nil, err
	}
}
