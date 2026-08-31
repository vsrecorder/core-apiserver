package entity

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNormalizeAcquisitionValue(t *testing.T) {
	t.Run("正常系_小文字化して返す", func(t *testing.T) {
		require.Equal(t, "howto_cta", NormalizeAcquisitionValue("Howto_CTA", 64))
	})

	t.Run("正常系_前後の空白を落とす", func(t *testing.T) {
		require.Equal(t, "x", NormalizeAcquisitionValue("  x  ", 32))
	})

	t.Run("正常系_列長を超えたら切り詰める", func(t *testing.T) {
		require.Equal(t, strings.Repeat("a", 32), NormalizeAcquisitionValue(strings.Repeat("a", 100), 32))
	})

	t.Run("異常系_想定外の文字を含む値は捨てる", func(t *testing.T) {
		// クエリ文字列は誰でも任意の値を付けられるため、
		// 集計軸を汚す値は保存せず「判明しなかった」ものとして扱う
		for _, value := range []string{
			"",
			"x y",
			"x'; DROP TABLE users;--",
			"<script>",
			"日本語",
			"x.com",
		} {
			require.Equal(t, "", NormalizeAcquisitionValue(value, 32), value)
		}
	})
}

func TestNormalizeAcquisitionCampaign(t *testing.T) {
	t.Run("正常系_allowlistの投稿タイプはそのまま採用する", func(t *testing.T) {
		for _, campaign := range []string{
			AcquisitionCampaignPainpoint,
			AcquisitionCampaignHowtoCta,
			AcquisitionCampaignFeature,
			AcquisitionCampaignStats,
			AcquisitionCampaignMeta,
			AcquisitionCampaignProfile,
		} {
			require.Equal(t, campaign, NormalizeAcquisitionCampaign(campaign))
		}
	})

	t.Run("正常系_未知の投稿タイプはotherに丸める", func(t *testing.T) {
		// 表記ゆれで GROUP BY が割れるのを防ぐ。捨てずに残すのは
		// 「タグ無しの直接流入」と区別を付けるため
		require.Equal(t, AcquisitionCampaignOther, NormalizeAcquisitionCampaign("howto"))
		require.Equal(t, AcquisitionCampaignOther, NormalizeAcquisitionCampaign("HOWTO_CTA2"))
	})

	t.Run("正常系_タグが無ければ空文字のままにする", func(t *testing.T) {
		require.Equal(t, "", NormalizeAcquisitionCampaign(""))
		require.Equal(t, "", NormalizeAcquisitionCampaign("日本語"))
	})
}

func TestNormalizeAcquisitionReferrer(t *testing.T) {
	t.Run("正常系_URLからホスト名だけを取り出す", func(t *testing.T) {
		// パス以降には検索語などが含まれうるため保存しない
		require.Equal(t, "t.co", NormalizeAcquisitionReferrer("https://t.co/EAbruFy36h"))
		require.Equal(t, "www.google.com", NormalizeAcquisitionReferrer("https://www.google.com/search?q=%E3%83%90%E3%83%88%E3%83%AC%E3%82%B3"))
		require.Equal(t, "x.com", NormalizeAcquisitionReferrer("https://X.com:443/vsrecorder_mobi"))
	})

	t.Run("正常系_ホスト名だけを渡してもそのまま通る", func(t *testing.T) {
		require.Equal(t, "t.co", NormalizeAcquisitionReferrer("t.co"))
	})

	t.Run("異常系_ホスト名として読めない値は捨てる", func(t *testing.T) {
		for _, referrer := range []string{
			"",
			"https://localhost/",
			"not a url",
			"https://<script>/",
		} {
			require.Equal(t, "", NormalizeAcquisitionReferrer(referrer), referrer)
		}
	})
}

func TestNormalizeAcquisitionLandingPath(t *testing.T) {
	t.Run("正常系_パスを返しクエリは落とす", func(t *testing.T) {
		require.Equal(t, "/", NormalizeAcquisitionLandingPath("/"))
		require.Equal(t, "/records/quick", NormalizeAcquisitionLandingPath("/records/quick?utm_source=x"))
	})

	t.Run("異常系_絶対パスでない値は捨てる", func(t *testing.T) {
		for _, path := range []string{
			"",
			"records/quick",
			"https://vsrecorder.mobi/records/quick",
			"//evil.example.com",
		} {
			require.Equal(t, "", NormalizeAcquisitionLandingPath(path), path)
		}
	})
}

func TestInferAcquisitionSource(t *testing.T) {
	t.Run("正常系_Xのリファラからチャネルを推定する", func(t *testing.T) {
		for _, host := range []string{"t.co", "x.com", "www.x.com", "twitter.com", "mobile.twitter.com"} {
			source, medium := InferAcquisitionSource(host)
			require.Equal(t, "x", source, host)
			require.Equal(t, "referral", medium, host)
		}
	})

	t.Run("正常系_検索エンジンのリファラからチャネルを推定する", func(t *testing.T) {
		source, medium := InferAcquisitionSource("www.google.com")
		require.Equal(t, "search", source)
		require.Equal(t, "organic", medium)
	})

	t.Run("異常系_似ているだけのホストは推定しない", func(t *testing.T) {
		// "notx.com" を "x.com" の一致として拾わないこと
		for _, host := range []string{"", "notx.com", "example.com", "xx.com"} {
			source, medium := InferAcquisitionSource(host)
			require.Equal(t, "", source, host)
			require.Equal(t, "", medium, host)
		}
	})
}

func TestUserAcquisitionIsEmpty(t *testing.T) {
	t.Run("正常系_何も判明していなければ空とみなす", func(t *testing.T) {
		a := NewUserAcquisition("zor5SLfEfwfZ90yRVXzlxBEFARy2", acquisitionTestTime())
		require.True(t, a.IsEmpty())
	})

	t.Run("正常系_着地ページだけでは空とみなす", func(t *testing.T) {
		// 着地ページは全訪問者に付く値で、流入元を1つも語らない
		a := NewUserAcquisition("zor5SLfEfwfZ90yRVXzlxBEFARy2", acquisitionTestTime())
		a.LandingPath = "/"
		require.True(t, a.IsEmpty())
	})

	t.Run("正常系_リファラだけでも空とはみなさない", func(t *testing.T) {
		a := NewUserAcquisition("zor5SLfEfwfZ90yRVXzlxBEFARy2", acquisitionTestTime())
		a.Referrer = "t.co"
		require.False(t, a.IsEmpty())
	})
}

// acquisitionTestTime はJSON/DBを往復しない内部比較用の固定時刻。
func acquisitionTestTime() time.Time {
	return time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
}
