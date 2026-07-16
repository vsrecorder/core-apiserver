// check-firebase-users は、Firebase Authentication 上のユーザーと、DB(users テーブル)の
// 有効なユーザー(deleted_at IS NULL)に差異がないかを突合して確認するための調査用ツール。
//
// 前提: users.id は Firebase の UID と同一。webapp のログイン処理(src/app/auth.ts)が
// Firebase の ID トークンを検証し、decoded.uid をそのまま core-apiserver の
// POST /api/v1beta/users に渡してユーザーを作成しているため、両者は 1 対 1 に対応する。
//
// 検出する差異は次の 2 種類:
//
//   - firebase_only: Firebase には存在するが、DB に有効なユーザーとして存在しない。
//     同じ firebase_only でも原因と取るべき対処が正反対になるため、次の A / B に分類して表示する。
//
//     A:退会済み   … DB に行はあるが deleted_at が入っている。退会処理で DB の論理削除は
//     成功したが、Firebase 側のユーザー削除に失敗して取り残されたケース。
//     本人の意思による退会なので、再登録を促すような連絡をしてはいけない。
//
//     B:登録未完了 … DB に行が無い。Firebase のユーザー作成後、DB 登録に到達せずに
//     中断したケース。本人はログインしたつもりでサービスを使えない
//     状態のため、フォローの対象になる。
//
//   - db_only: DB には有効なユーザーとして存在するが、Firebase には存在しない。
//     Firebase 側のユーザーが誤って削除された、もしくは別プロジェクトのデータを
//     参照しているケース。ログインできない状態のため影響が大きい。
//
// Firebase Authentication にはユーザーの論理削除の概念がなく、ListUsers は削除済み
// ユーザーを返さないため、Firebase 側は「存在する = 有効」として扱う。
//
// 本ツールは読み取り専用で、Firebase・DB のいずれにも一切書き込みを行わない。
//
// 認証情報:
//
//	GOOGLE_APPLICATION_CREDENTIALS にサービスアカウントJSONのパスを設定するか、
//	webapp の .env と同じ FIREBASE_PROJECT_ID / FIREBASE_CLIENT_EMAIL / FIREBASE_PRIVATE_KEY
//	を設定する(後者が設定されていればそちらを優先する)。
//	DB 接続情報は他のバッチと同様 DB_HOSTNAME 等の環境変数を使う。
//
// 使い方:
//
//	# 差異のサマリと一覧を表示する
//	go run ./cmd/check-firebase-users
//
//	# UIDだけでなくメールアドレス・作成日時も表示する
//	go run ./cmd/check-firebase-users -verbose
//
//	# 差異があった場合に終了コード 1 を返す(CI・定期実行で検知したい場合)
//	go run ./cmd/check-firebase-users -exit-code
//
// 終了コード: 0 = 正常終了(-exit-code 指定時は差異なし)、1 = エラー(または差異あり)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/joho/godotenv"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/postgres"
)

const (
	ExitCodeOK = iota
	ExitCodeNG
)

// dbUser は突合に必要な users テーブルの情報。deleted_at が入っている行(退会済み)も
// 「Firebaseには残っているが退会済み」というケースを区別して報告するために読み込む。
type dbUser struct {
	ID        string
	Name      string
	CreatedAt time.Time
	DeletedAt *time.Time
}

// firebaseUser は突合に必要な Firebase Authentication 上のユーザー情報。
type firebaseUser struct {
	UID       string
	Email     string
	CreatedAt time.Time
}

func main() {
	verbose := flag.Bool("verbose", false, "UIDに加えてメールアドレスや作成日時などの詳細を表示する")
	exitCode := flag.Bool("exit-code", false, "true の場合、差異が見つかったら終了コード1で終了する")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	ctx := context.Background()

	authClient, err := newFirebaseAuthClient(ctx)
	if err != nil {
		log.Printf("failed to create firebase client: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	db, err := postgres.NewDB(
		os.Getenv("DB_HOSTNAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER_NAME"),
		os.Getenv("DB_USER_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	if err != nil {
		log.Printf("failed to connect database: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	firebaseUsers, err := listFirebaseUsers(ctx, authClient)
	if err != nil {
		log.Printf("failed to list firebase users: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	dbUsers, err := listDBUsers(db)
	if err != nil {
		log.Printf("failed to list db users: %v\n", err)
		os.Exit(ExitCodeNG)
	}

	activeCount := 0
	for _, u := range dbUsers {
		if u.DeletedAt == nil {
			activeCount++
		}
	}

	log.Printf("firebase: %d users, db: %d users (有効: %d, 退会済み: %d)\n",
		len(firebaseUsers), len(dbUsers), activeCount, len(dbUsers)-activeCount)

	firebaseOnly, dbOnly := diff(firebaseUsers, dbUsers)

	report(firebaseOnly, dbOnly, dbUsers, firebaseUsers, *verbose)

	if *exitCode && (len(firebaseOnly) > 0 || len(dbOnly) > 0) {
		os.Exit(ExitCodeNG)
	}

	os.Exit(ExitCodeOK)
}

// newFirebaseAuthClient は Firebase Admin SDK の認証クライアントを生成する。
// webapp(src/firebase/admin.ts)と同じ FIREBASE_* の環境変数が揃っていればそれを使い、
// 揃っていなければ GOOGLE_APPLICATION_CREDENTIALS などの Application Default Credentials に任せる。
func newFirebaseAuthClient(ctx context.Context) (*auth.Client, error) {
	projectId := os.Getenv("FIREBASE_PROJECT_ID")
	clientEmail := os.Getenv("FIREBASE_CLIENT_EMAIL")
	privateKey := os.Getenv("FIREBASE_PRIVATE_KEY")

	var opts []option.ClientOption
	config := &firebase.Config{}

	if projectId != "" && clientEmail != "" && privateKey != "" {
		// .env に格納する都合で改行が \n にエスケープされているため、webapp 側と同様に元に戻す
		credentials, err := buildCredentialsJSON(projectId, clientEmail, privateKey)
		if err != nil {
			return nil, err
		}

		opts = append(opts, option.WithCredentialsJSON(credentials))
		config.ProjectID = projectId
	} else if projectId != "" {
		config.ProjectID = projectId
	}

	app, err := firebase.NewApp(ctx, config, opts...)
	if err != nil {
		return nil, err
	}

	return app.Auth(ctx)
}

// buildCredentialsJSON は FIREBASE_* の環境変数から、Admin SDK が受け取れる
// サービスアカウントJSONを組み立てる。
func buildCredentialsJSON(projectId string, clientEmail string, privateKey string) ([]byte, error) {
	credentials := map[string]string{
		"type":         "service_account",
		"project_id":   projectId,
		"client_email": clientEmail,
		// .env では改行が \n にエスケープされているため、webapp(src/firebase/admin.ts)と同様に元に戻す
		"private_key": strings.ReplaceAll(privateKey, "\\n", "\n"),
		"token_uri":   "https://oauth2.googleapis.com/token",
	}

	return json.Marshal(credentials)
}

// listFirebaseUsers は Firebase Authentication 上の全ユーザーを取得する。
// Firebase Authentication に論理削除の概念はなく、削除済みユーザーは返ってこないため、
// ここで取得できたユーザーは全て有効なユーザーとして扱う。
func listFirebaseUsers(ctx context.Context, client *auth.Client) (map[string]*firebaseUser, error) {
	users := make(map[string]*firebaseUser)

	// ListUsers は内部で1000件ずつページングして全件を辿る
	iter := client.Users(ctx, "")
	for {
		user, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		users[user.UID] = &firebaseUser{
			UID:   user.UID,
			Email: user.Email,
			// UserMetadata の時刻はミリ秒のUNIX時間
			CreatedAt: time.UnixMilli(user.UserMetadata.CreationTimestamp).Local(),
		}
	}

	return users, nil
}

// listDBUsers は users テーブルの全ユーザーを取得する。gorm.DeletedAt によって
// 論理削除された行はデフォルトで除外されるため、Unscoped で退会済みも含めて取得し、
// 有効/退会済みは DeletedAt の有無で判別する。
func listDBUsers(db *gorm.DB) (map[string]*dbUser, error) {
	var rows []*model.User
	if tx := db.Unscoped().Model(&model.User{}).Order("id ASC").Find(&rows); tx.Error != nil {
		return nil, tx.Error
	}

	users := make(map[string]*dbUser, len(rows))
	for _, row := range rows {
		u := &dbUser{
			ID:        row.ID,
			Name:      row.Name,
			CreatedAt: row.CreatedAt,
		}
		if row.DeletedAt.Valid {
			deletedAt := row.DeletedAt.Time
			u.DeletedAt = &deletedAt
		}

		users[row.ID] = u
	}

	return users, nil
}

// diff は Firebase と DB の突合を行い、差異のある UID を返す。
//
//   - firebaseOnly: Firebase に存在するが、DB に有効なユーザーとして存在しない UID
//     (DBに行が無い場合と、行はあるが退会済みの場合の両方を含む)
//   - dbOnly:       DB に有効なユーザーとして存在するが、Firebase に存在しない UID
func diff(firebaseUsers map[string]*firebaseUser, dbUsers map[string]*dbUser) (firebaseOnly []string, dbOnly []string) {
	for uid := range firebaseUsers {
		u, ok := dbUsers[uid]
		if !ok || u.DeletedAt != nil {
			firebaseOnly = append(firebaseOnly, uid)
		}
	}

	for id, u := range dbUsers {
		if u.DeletedAt != nil {
			continue
		}
		if _, ok := firebaseUsers[id]; !ok {
			dbOnly = append(dbOnly, id)
		}
	}

	sort.Strings(firebaseOnly)
	sort.Strings(dbOnly)

	return firebaseOnly, dbOnly
}

// classifyFirebaseOnly は firebase_only のユーザーを A / B のどちらかに分類し、
// ラベルと、判断の根拠になった状態を返す。
//
//   - A:退会済み   … DBに行はあるが deleted_at が入っている。本人の意思で退会しているため、
//     再登録を促すような連絡をしてはいけない。残った Firebase ユーザーの削除が対処になる。
//   - B:登録未完了 … DBに行が無い。本人はログインしたつもりでサービスを使えない状態のため、
//     フォローの対象になる。
func classifyFirebaseOnly(uid string, dbUsers map[string]*dbUser) (label string, state string) {
	if u, ok := dbUsers[uid]; ok && u.DeletedAt != nil {
		return "A:退会済み", "DB上は退会済み(deleted_at=" + u.DeletedAt.Format(time.RFC3339) + ")"
	}

	return "B:登録未完了", "DBに行なし"
}

// report は突合結果を標準出力へ出力する。
func report(
	firebaseOnly []string,
	dbOnly []string,
	dbUsers map[string]*dbUser,
	firebaseUsers map[string]*firebaseUser,
	verbose bool,
) {
	if len(firebaseOnly) == 0 && len(dbOnly) == 0 {
		log.Printf("OK: Firebaseのユーザーと、DBの有効なユーザー(deleted_at IS NULL)に差異はありません\n")
		return
	}

	if len(firebaseOnly) > 0 {
		log.Printf("NG: Firebaseにのみ存在するユーザーが %d 件あります(DBに未登録、またはDB上は退会済み)\n", len(firebaseOnly))
		log.Printf("    A:退会済み   = 退会時にFirebase側の削除が失敗して残ったもの。本人は退会済みのため、再登録を促す連絡をしてはいけない\n")
		log.Printf("    B:登録未完了 = 登録がDB登録前に中断したもの。本人はログインしたつもりでサービスを使えていない\n")

		for _, uid := range firebaseOnly {
			label, state := classifyFirebaseOnly(uid, dbUsers)

			if verbose {
				fu := firebaseUsers[uid]
				log.Printf("  firebase_only uid=%s [%s] %s email=%s firebase_created_at=%s\n",
					uid, label, state, fu.Email, fu.CreatedAt.Format(time.RFC3339))
			} else {
				log.Printf("  firebase_only uid=%s [%s] %s\n", uid, label, state)
			}
		}
	}

	if len(dbOnly) > 0 {
		log.Printf("NG: DBにのみ有効なユーザーとして存在するUIDが %d 件あります(Firebaseに存在しないためログインできません)\n", len(dbOnly))
		for _, id := range dbOnly {
			if verbose {
				u := dbUsers[id]
				log.Printf("  db_only uid=%s name=%s db_created_at=%s\n",
					id, u.Name, u.CreatedAt.Format(time.RFC3339))
			} else {
				log.Printf("  db_only uid=%s\n", id)
			}
		}
	}
}
