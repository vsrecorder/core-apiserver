package entity

import (
	"regexp"
	"strings"
	"time"
)

// 流入元(utm_campaign)の投稿タイプ。X 投稿の分類に対応する(utm-attribution-plan.md §3.2)。
// 分類を増やすほど1タイプあたりの n が減り、判断に使えるようになる時期が遠のくため、
// 足すときは「その軸で発信配分を変える気があるか」を確認してから足す(同 §5.3)。
const (
	AcquisitionCampaignPainpoint = "painpoint" // 課題提起・共感型(水曜の痛み枠)
	AcquisitionCampaignHowtoCta  = "howto_cta" // ハウツー+CTA型(主力・金土日に投下)
	AcquisitionCampaignFeature   = "feature"   // 機能紹介型(火曜)
	AcquisitionCampaignStats     = "stats"     // 戦績シェア型
	AcquisitionCampaignMeta      = "meta"      // 環境・メタ分析型(月曜)
	AcquisitionCampaignTalk      = "talk"      // 雑談・問いかけ型(アンケート等)
	AcquisitionCampaignProfile   = "profile"   // プロフィールの固定リンク

	// 以下はユーザー自身の共有から来る流入(P5_ACQUISITION_PLAN.md)。
	// 運営者の発信(utm_source=x)とは source が分かれるため、同じ campaign 空間に同居できる。
	AcquisitionCampaignRecap  = "recap"  // 週次レポート等のシェアカード
	AcquisitionCampaignRecord = "record" // 戦績シェア
	AcquisitionCampaignKizuna = "kizuna" // キズナ

	// AcquisitionCampaignOther は allowlist に無い campaign の丸め先。
	// 弾いて NULL にすると「UTM無しの直接流入」と区別が付かなくなるため、
	// 「タグは付いていたが未知の分類だった」ことが残るよう専用の値へ寄せる。
	AcquisitionCampaignOther = "(other)"
)

var acquisitionCampaigns = map[string]struct{}{
	AcquisitionCampaignPainpoint: {},
	AcquisitionCampaignHowtoCta:  {},
	AcquisitionCampaignFeature:   {},
	AcquisitionCampaignStats:     {},
	AcquisitionCampaignMeta:      {},
	AcquisitionCampaignTalk:      {},
	AcquisitionCampaignProfile:   {},
	AcquisitionCampaignRecap:     {},
	AcquisitionCampaignRecord:    {},
	AcquisitionCampaignKizuna:    {},
}

// X 投稿の運用は `<型>_<曜>_MMDD_<ネタ>` という値を utm_campaign に入れている
// (例 `pain_wed_0812_janken` / `howto_fri_0828_prep`。x-post スキル §0)。
// この対応表は先頭トークン(型)を上の正規の投稿タイプへ寄せるためのもの。
//
// **既に X へ投稿したリンクは書き換えられない**(投稿は数ヶ月にわたってクリックされ続ける)。
// 運用側の規則を変えるのではなく、サーバがこの形式を解釈する。そうしないと過去投稿からの
// 流入がすべて (other) に落ち、投稿タイプ別の比較そのものが成立しない。
var acquisitionCampaignByPrefix = map[string]string{
	"pain":    AcquisitionCampaignPainpoint,
	"howto":   AcquisitionCampaignHowtoCta,
	"feature": AcquisitionCampaignFeature,
	"share":   AcquisitionCampaignStats,
	"env":     AcquisitionCampaignMeta,
	"talk":    AcquisitionCampaignTalk,
}

const (
	// 列長。schema.sql の user_acquisitions に合わせる。
	acquisitionSourceMaxLength   = 32
	acquisitionMediumMaxLength   = 32
	acquisitionCampaignMaxLength = 64
	acquisitionContentMaxLength  = 64
	acquisitionReferrerMaxLength = 255
	acquisitionPathMaxLength     = 255
)

// utm_* として採用する文字の集合。X 投稿に貼るリンクに使う値はすべてこの範囲に収まる。
var acquisitionValuePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

// NormalizeAcquisitionValue は utm_* の値を小文字化・長さ切り詰めし、
// 想定外の文字を含むものは空文字(=記録しない)にする。
//
// utm_* はクエリ文字列であり誰でも任意の値を付けられる。無検証で保存すると
// 表記ゆれやゴミ値で Grafana の GROUP BY が割れ、UTM 運用そのものが破綻する。
// webapp の proxy でも同じ正規化を通しているが、Cookie はクライアントから
// 差し替えられるため、サーバ側でも必ずここを通す。
func NormalizeAcquisitionValue(value string, maxLength int) string {
	v := strings.ToLower(strings.TrimSpace(value))
	if len(v) > maxLength {
		v = v[:maxLength]
	}

	if !acquisitionValuePattern.MatchString(v) {
		return ""
	}

	return v
}

// NormalizeAcquisitionSource は流入チャネル('x' 等)を正規化する。
func NormalizeAcquisitionSource(source string) string {
	return NormalizeAcquisitionValue(source, acquisitionSourceMaxLength)
}

// NormalizeAcquisitionMedium は媒体種別('social' 等)を正規化する。
func NormalizeAcquisitionMedium(medium string) string {
	return NormalizeAcquisitionValue(medium, acquisitionMediumMaxLength)
}

// NormalizeAcquisitionCampaign は投稿タイプを allowlist に丸める。
// 空文字(タグ無し)はそのまま空文字を返し、未知の値だけを (other) にする。
//
// 判定は「完全一致 → 先頭トークンの対応表」の順。運用が使う
// `<型>_<曜>_MMDD_<ネタ>` 形式は後者で拾う(acquisitionCampaignByPrefix のコメント参照)。
func NormalizeAcquisitionCampaign(campaign string) string {
	v := NormalizeAcquisitionValue(campaign, acquisitionCampaignMaxLength)
	if v == "" {
		return ""
	}

	if _, ok := acquisitionCampaigns[v]; ok {
		return v
	}

	prefix, _, found := strings.Cut(v, "_")
	if found {
		if canonical, ok := acquisitionCampaignByPrefix[prefix]; ok {
			return canonical
		}
	}

	return AcquisitionCampaignOther
}

// NormalizeAcquisitionContent は投稿の識別子(YYYYMMDD+連番)を正規化する。
func NormalizeAcquisitionContent(content string) string {
	return NormalizeAcquisitionValue(content, acquisitionContentMaxLength)
}

// NormalizeAcquisitionReferrer はリファラを保存できる形に整える。
// 保存するのはホスト名のみで、パス以降は落とす(検索語などが含まれうるため)。
// 受け取る時点で proxy がホスト名に切り詰めているが、ここでも同じ形に揃える。
func NormalizeAcquisitionReferrer(referrer string) string {
	v := strings.ToLower(strings.TrimSpace(referrer))
	if v == "" {
		return ""
	}

	// スキーム・パス・ポートが混ざっていてもホスト名だけを取り出す
	if i := strings.Index(v, "://"); i >= 0 {
		v = v[i+3:]
	}
	if i := strings.IndexAny(v, "/?#"); i >= 0 {
		v = v[:i]
	}
	if i := strings.LastIndex(v, "@"); i >= 0 {
		v = v[i+1:]
	}
	if i := strings.Index(v, ":"); i >= 0 {
		v = v[:i]
	}

	if !isAcquisitionHost(v) {
		return ""
	}

	if len(v) > acquisitionReferrerMaxLength {
		return ""
	}

	return v
}

// ホスト名として妥当な文字だけで構成され、ドットを含むか(FQDN らしいか)を見る。
var acquisitionHostPattern = regexp.MustCompile(`^[a-z0-9.-]+$`)

func isAcquisitionHost(host string) bool {
	return host != "" && strings.Contains(host, ".") && acquisitionHostPattern.MatchString(host)
}

// NormalizeAcquisitionLandingPath は着地ページのパスを正規化する。
// 絶対パスでないもの(クライアントが URL 全体を入れてきた場合など)は捨てる。
func NormalizeAcquisitionLandingPath(path string) string {
	v := strings.TrimSpace(path)
	if !strings.HasPrefix(v, "/") || strings.HasPrefix(v, "//") {
		return ""
	}

	// クエリ・フラグメントは着地ページの識別には要らない(utm_* がそのまま入るため落とす)
	if i := strings.IndexAny(v, "?#"); i >= 0 {
		v = v[:i]
	}

	if len(v) > acquisitionPathMaxLength {
		return ""
	}

	return v
}

// UserAcquisition は登録者の流入元。ユーザー1人につき1行で、登録の瞬間にだけ作られる。
// 初回タッチ(first-touch)を採るため、一度作られた行は上書きしない。
type UserAcquisition struct {
	UserId string
	// Source は流入チャネル。空文字なら流入元が判明しなかったことを表す。
	Source string
	Medium string
	// Campaign は投稿タイプ。allowlist 外は AcquisitionCampaignOther に丸められている。
	Campaign string
	Content  string
	// Referrer はホスト名のみ。
	Referrer    string
	LandingPath string
	// LandingAt は着地時刻。登録時刻との差が「登録はリーチの数週間後に起きる」遅延構造の実測値になる。
	LandingAt time.Time
	// SourceInferred は Source が utm_source ではなくリファラからの推定であることを表す。
	// 確定値と混ぜて数えると判明率を過大評価するため、分けて数えられるように持つ。
	SourceInferred bool
	// SurveyAnswer は登録時アンケート「どこで知ったか」の回答(S4)。
	SurveyAnswer string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewUserAcquisition(
	userId string,
	createdAt time.Time,
) *UserAcquisition {
	return &UserAcquisition{
		UserId:    userId,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

// IsEmpty は流入元として保存する価値のある値が1つも無いかを返す。
// 何も判明しなかった行を作っても「UTM無しの直接流入」と同じことしか言えず、
// users との LEFT JOIN で同じように判定できるため、書かずに済ませる。
func (a *UserAcquisition) IsEmpty() bool {
	return a.Source == "" &&
		a.Medium == "" &&
		a.Campaign == "" &&
		a.Content == "" &&
		a.Referrer == "" &&
		a.SurveyAnswer == ""
}

// リファラのホスト名から流入チャネルを推定する対応表。
//
// UTM はタグ付きリンクを踏んだ人しか捕捉できず、「リンクを踏まず後日サービス名で
// 直接来た人」「非タグリンク経由」を取りこぼす(utm-attribution-plan.md §3.6)。
// リファラは着地時点で既に手元にあるため、これを見るだけで判明率を最も安く底上げできる。
//
// 推定値は utm_source による確定値と区別できるよう SourceInferred で印を付ける。
// 混ぜて数えると判明率を過大評価するため、Grafana でも分けて出す。
var acquisitionSourceByReferrerHost = []struct {
	suffix string
	source string
	medium string
}{
	// X。t.co は X が貼り付けたリンクを包むドメインで、アプリ内ブラウザからの着地はこれになる
	{suffix: "t.co", source: "x", medium: "referral"},
	{suffix: "x.com", source: "x", medium: "referral"},
	{suffix: "twitter.com", source: "x", medium: "referral"},
	// 検索。エンジンの内訳までは追わない(知りたいのは「検索から来たか」の粒度)
	{suffix: "google.com", source: "search", medium: "organic"},
	{suffix: "google.co.jp", source: "search", medium: "organic"},
	{suffix: "yahoo.co.jp", source: "search", medium: "organic"},
	{suffix: "bing.com", source: "search", medium: "organic"},
	{suffix: "duckduckgo.com", source: "search", medium: "organic"},
}

// InferAcquisitionSource はリファラのホスト名から流入チャネルを推定する。
// 推定できなかった場合は空文字を返す。
func InferAcquisitionSource(referrerHost string) (source string, medium string) {
	host := NormalizeAcquisitionReferrer(referrerHost)
	if host == "" {
		return "", ""
	}

	for _, rule := range acquisitionSourceByReferrerHost {
		// "x.com" と "www.x.com" の両方を拾いつつ、"notx.com" は拾わない
		if host == rule.suffix || strings.HasSuffix(host, "."+rule.suffix) {
			return rule.source, rule.medium
		}
	}

	return "", ""
}
