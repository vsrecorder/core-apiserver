package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
)

// fakeDeckAssetS3 は実S3の代わりに使うインメモリ実装。
type fakeDeckAssetS3 struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func newFakeDeckAssetS3() *fakeDeckAssetS3 {
	return &fakeDeckAssetS3{objects: map[string][]byte{}}
}

func (f *fakeDeckAssetS3) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.objects[*params.Key]; !ok {
		return nil, &types.NotFound{}
	}

	return &s3.HeadObjectOutput{}, nil
}

func (f *fakeDeckAssetS3) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	body, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}
	f.objects[*params.Key] = body

	return &s3.PutObjectOutput{}, nil
}

// setup4DeckAssetInfrastructure はフェイクS3とhttptestサーバを組み込んだDeckAssetを返す。
func setup4DeckAssetInfrastructure(t *testing.T, handler http.HandlerFunc) (*DeckAsset, *fakeDeckAssetS3) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	originalHTML := deckResultHTMLURLFormat
	originalImage := deckImageURLFormat
	deckResultHTMLURLFormat = server.URL + "/deck/result.html/deckID/%s"
	deckImageURLFormat = server.URL + "/deck/deckView.php/deckID/%s.png"
	t.Cleanup(func() {
		deckResultHTMLURLFormat = originalHTML
		deckImageURLFormat = originalImage
	})

	fakeS3 := newFakeDeckAssetS3()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := &DeckAsset{
		logger: logger,
		newS3Client: func(ctx context.Context) (deckAssetS3API, error) {
			return fakeS3, nil
		},
	}

	return d, fakeS3
}

// newTestPNG は1x1のPNG画像を返す。
func newTestPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	return buf.Bytes()
}

func TestDeckAssetInfrastructure(t *testing.T) {
	deckCode := "5dbFbk-uBwjqP-VVk5Vv"

	t.Run("UploadDeckResultHTML", func(t *testing.T) {
		t.Run("正常系_取得したHTMLをS3へアップロードする", func(t *testing.T) {
			d, fakeS3 := setup4DeckAssetInfrastructure(t, func(w http.ResponseWriter, req *http.Request) {
				require.Equal(t, "/deck/result.html/deckID/"+deckCode, req.URL.Path)
				fmt.Fprint(w, "<html>deck result</html>")
			})

			require.NoError(t, d.UploadDeckResultHTML(context.Background(), deckCode))

			key := "deck-result_html/" + deckCode
			require.Equal(t, []byte("<html>deck result</html>"), fakeS3.objects[key])
		})

		t.Run("正常系_アップロード済みなら取得せずスキップする", func(t *testing.T) {
			d, fakeS3 := setup4DeckAssetInfrastructure(t, func(w http.ResponseWriter, req *http.Request) {
				t.Fatal("アップロード済みの場合は外部サイトへ取得しに行ってはいけない")
			})

			key := "deck-result_html/" + deckCode
			fakeS3.objects[key] = []byte("uploaded")

			require.NoError(t, d.UploadDeckResultHTML(context.Background(), deckCode))
			require.Equal(t, []byte("uploaded"), fakeS3.objects[key])
		})

		t.Run("異常系_メンテナンス中はErrUnderMaintenanceを返しアップロードしない", func(t *testing.T) {
			d, fakeS3 := setup4DeckAssetInfrastructure(t, func(w http.ResponseWriter, req *http.Request) {
				fmt.Fprint(w, "<html>現在メンテナンスをしております</html>")
			})

			err := d.UploadDeckResultHTML(context.Background(), deckCode)

			require.ErrorIs(t, err, apperror.ErrUnderMaintenance)
			require.Empty(t, fakeS3.objects)
		})

		t.Run("異常系_不正なデッキコードはErrDeckCodeInvalidを返しアップロードしない", func(t *testing.T) {
			d, fakeS3 := setup4DeckAssetInfrastructure(t, func(w http.ResponseWriter, req *http.Request) {
				fmt.Fprint(w, "<html>デッキコードが正しくありません</html>")
			})

			err := d.UploadDeckResultHTML(context.Background(), deckCode)

			require.ErrorIs(t, err, apperror.ErrDeckCodeInvalid)
			require.Empty(t, fakeS3.objects)
		})

		t.Run("異常系_200以外のステータスはエラーを返しアップロードしない", func(t *testing.T) {
			d, fakeS3 := setup4DeckAssetInfrastructure(t, func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			})

			err := d.UploadDeckResultHTML(context.Background(), deckCode)

			require.Error(t, err)
			require.Empty(t, fakeS3.objects)
		})
	})

	t.Run("UploadDeckImage", func(t *testing.T) {
		t.Run("正常系_取得した画像をJPEGへ変換してアップロードする", func(t *testing.T) {
			d, fakeS3 := setup4DeckAssetInfrastructure(t, func(w http.ResponseWriter, req *http.Request) {
				require.Equal(t, "/deck/deckView.php/deckID/"+deckCode+".png", req.URL.Path)
				w.Write(newTestPNG(t))
			})

			require.NoError(t, d.UploadDeckImage(context.Background(), deckCode))

			key := "images/decks/" + deckCode + ".jpg"
			uploaded, ok := fakeS3.objects[key]
			require.True(t, ok)
			require.Equal(t, "image/jpeg", http.DetectContentType(uploaded))
		})

		t.Run("正常系_アップロード済みなら取得せずスキップする", func(t *testing.T) {
			d, fakeS3 := setup4DeckAssetInfrastructure(t, func(w http.ResponseWriter, req *http.Request) {
				t.Fatal("アップロード済みの場合は外部サイトへ取得しに行ってはいけない")
			})

			key := "images/decks/" + deckCode + ".jpg"
			fakeS3.objects[key] = []byte("uploaded")

			require.NoError(t, d.UploadDeckImage(context.Background(), deckCode))
		})

		t.Run("異常系_画像として解釈できないレスポンスはエラーを返す", func(t *testing.T) {
			d, fakeS3 := setup4DeckAssetInfrastructure(t, func(w http.ResponseWriter, req *http.Request) {
				fmt.Fprint(w, "not an image")
			})

			err := d.UploadDeckImage(context.Background(), deckCode)

			require.Error(t, err)
			require.Empty(t, fakeS3.objects)
		})

		t.Run("異常系_200以外のステータスはエラーを返しアップロードしない", func(t *testing.T) {
			d, fakeS3 := setup4DeckAssetInfrastructure(t, func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			})

			err := d.UploadDeckImage(context.Background(), deckCode)

			require.Error(t, err)
			require.Empty(t, fakeS3.objects)
		})
	})
}

func TestConvertPNG2JPG(t *testing.T) {
	t.Run("正常系_PNGをJPEGへ変換する", func(t *testing.T) {
		ret, err := convertPNG2JPG(newTestPNG(t))

		require.NoError(t, err)
		require.Equal(t, "image/jpeg", http.DetectContentType(ret))
	})

	t.Run("異常系_PNG以外はエラーを返す", func(t *testing.T) {
		_, err := convertPNG2JPG([]byte("not a png"))

		require.Error(t, err)
	})
}
