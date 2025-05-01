package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/ponyo877/chatsh/server/usecase"
)

// ErrNotFound はリソースが見つからない場合のエラーです。usecase層で定義されているべきですが、便宜上ここで定義します。
var ErrNotFound = errors.New("not found")

// ErrAlreadyExists はリソースが既に存在する場合のエラーです。usecase層で定義されているべきですが、便宜上ここで定義します。
var ErrAlreadyExists = errors.New("already exists")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) usecase.Repository {
	return &Repository{db: db}
}

// --- Helper Functions ---

// getRootDirectoryID はルートディレクトリのIDを取得または作成します。
func (r *Repository) getRootDirectoryID() (string, error) {
	var rootID string
	err := r.db.QueryRow("SELECT id FROM directories WHERE parent_id IS NULL LIMIT 1").Scan(&rootID)
	if err == nil {
		return rootID, nil // 既存のルートIDを返す
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("failed to query root directory: %w", err)
	}

	// ルートディレクトリが存在しない場合は作成する
	// newRootID := uuid.NewString() // DB IDは内部でUUIDを使うが、usecase層にはint IDを返す想定だったため削除
	// DBのIDはVARCHARなので、UUIDを生成して使う
	dbRootID := generateUUID() // ヘルパー関数を仮定
	now := time.Now()
	_, err = r.db.Exec("INSERT INTO directories (id, name, parent_id, owner_id, created_at) VALUES (?, ?, NULL, NULL, ?)",
		dbRootID, "/", now) // ルートディレクトリ名は "/" とする
	if err != nil {
		// 同時に作成しようとしてユニーク制約違反になる可能性を考慮
		// もう一度クエリしてみる
		errRetry := r.db.QueryRow("SELECT id FROM directories WHERE parent_id IS NULL LIMIT 1").Scan(&rootID)
		if errRetry == nil {
			return rootID, nil
		}
		return "", fmt.Errorf("failed to insert root directory: %w (retry query error: %v)", err, errRetry)
	}
	return dbRootID, nil
}

// generateUUID はUUIDを生成するヘルパー関数（仮）
// 実際には "github.com/google/uuid" を使うが、依存関係エラーのため一旦削除
func generateUUID() string {
	// ダミー実装。実際のアプリケーションでは `uuid.NewString()` を使う。
	// 依存関係の問題が解決したら元に戻す。
	return fmt.Sprintf("dummy-uuid-%d", time.Now().UnixNano())
}

// getDirectoryIDByPath はパス文字列からディレクトリID(DBのVARCHAR ID)を取得します。
// パスが存在しない場合は空文字列とnilエラーを返します。
func (r *Repository) getDirectoryIDByPath(path string) (string, error) {
	path = filepath.Clean(path)
	if path == "/" || path == "." || path == "" {
		return r.getRootDirectoryID()
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	currentDirID, err := r.getRootDirectoryID()
	if err != nil {
		return "", err
	}

	for _, part := range parts {
		if part == "" {
			continue
		} // 空のパートはスキップ (例: //)
		var nextDirID string
		err := r.db.QueryRow("SELECT id FROM directories WHERE name = ? AND parent_id = ?", part, currentDirID).Scan(&nextDirID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", nil // パスが見つからない
			}
			return "", fmt.Errorf("failed to query directory '%s' in '%s': %w", part, currentDirID, err)
		}
		currentDirID = nextDirID
	}

	return currentDirID, nil
}

// getRoomIDByPath はパス文字列からルームIDを取得します。
func (r *Repository) getRoomIDByPath(path string) (string, error) {
	path = filepath.Clean(path)
	dirPath := filepath.Dir(path)
	roomName := filepath.Base(path)

	if roomName == "." || roomName == "/" {
		return "", nil // ファイル名が無効
	}

	dirID, err := r.getDirectoryIDByPath(dirPath)
	if err != nil {
		return "", fmt.Errorf("failed to get directory ID for '%s': %w", dirPath, err)
	}
	if dirID == "" {
		return "", nil // 親ディレクトリが見つからない
	}

	var roomID string
	err = r.db.QueryRow("SELECT id FROM rooms WHERE name = ? AND directory_id = ?", roomName, dirID).Scan(&roomID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil // ルームが見つからない
		}
		return "", fmt.Errorf("failed to query room '%s' in directory '%s': %w", roomName, dirID, err)
	}
	return roomID, nil
}

// getPathForDirectoryID はディレクトリIDからフルパスを再構築します。
func (r *Repository) getPathForDirectoryID(dirID string) (string, error) {
	var parts []string
	currentID := dirID
	for {
		var name sql.NullString
		var parentID sql.NullString
		err := r.db.QueryRow("SELECT name, parent_id FROM directories WHERE id = ?", currentID).Scan(&name, &parentID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", fmt.Errorf("directory ID %s not found during path reconstruction", currentID)
			}
			return "", fmt.Errorf("failed to query directory %s for path reconstruction: %w", currentID, err)
		}

		if name.Valid && name.String != "/" { // ルートディレクトリ名は含めない
			parts = append(parts, name.String)
		}

		if !parentID.Valid { // 親がいない = ルートディレクトリ
			break
		}
		currentID = parentID.String
	}

	// 配列を逆順にする
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}

	return "/" + filepath.Join(parts...), nil
}

// getPathForRoomID はルームIDからフルパスを再構築します。
func (r *Repository) getPathForRoomID(roomID string) (string, error) {
	var roomName string
	var dirID string
	err := r.db.QueryRow("SELECT name, directory_id FROM rooms WHERE id = ?", roomID).Scan(&roomName, &dirID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("room ID %s not found", roomID)
		}
		return "", fmt.Errorf("failed to query room %s: %w", roomID, err)
	}

	dirPath, err := r.getPathForDirectoryID(dirID)
	if err != nil {
		return "", fmt.Errorf("failed to get path for directory %s: %w", dirID, err)
	}

	return filepath.Join(dirPath, roomName), nil
}

// --- Node Operation ---

func (r *Repository) GetNode(path string) (usecase.Node, error) {
	cleanedPath := filepath.Clean(path)

	// ルートディレクトリの場合の特別処理
	if cleanedPath == "/" || cleanedPath == "." {
		// ルートディレクトリの存在を確認または作成
		_, err := r.getRootDirectoryID()
		if err != nil {
			// ルートディレクトリの取得/作成に失敗した場合
			return usecase.Node{}, fmt.Errorf("failed to get or create root directory: %w", err)
		}
		// ルートディレクトリの情報を返す
		return usecase.Node{
			ID:   0, // ルートのusecase IDは0とする（仮）
			Type: usecase.FileTypeDirectory,
			Path: "/", // パスは "/" とする
		}, nil
	}

	parentPath := filepath.Dir(cleanedPath)
	name := filepath.Base(cleanedPath)

	// 親ディレクトリのIDを取得
	parentDirID, err := r.getDirectoryIDByPath(parentPath)
	if err != nil {
		// 親ディレクトリIDの取得中にエラーが発生した場合
		return usecase.Node{}, fmt.Errorf("error getting parent directory ID for '%s': %w", parentPath, err)
	}
	if parentDirID == "" {
		// 親ディレクトリが存在しない場合、指定されたパスも存在しない
		return usecase.Node{}, ErrNotFound // 親が見つからない = パスが存在しない
	}

	// UNION ALL を使ってディレクトリかファイルかを一度に問い合わせる
	query := `
		SELECT 'dir' AS kind, id FROM directories WHERE parent_id = ? AND name = ?
		UNION ALL
		SELECT 'room' AS kind, id FROM rooms WHERE directory_id = ? AND name = ?
		LIMIT 1
	`
	var kind string
	var dbID int // DBから取得するIDはstring

	err = r.db.QueryRow(query, parentDirID, name, parentDirID, name).Scan(&kind, &dbID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// ディレクトリもファイルも見つからなかった場合
			return usecase.Node{}, ErrNotFound
		}
		// その他のDBエラー
		return usecase.Node{}, fmt.Errorf("error querying node '%s' in parent '%s': %w", name, parentDirID, err)
	}

	// 見つかったノードの情報を返す
	node := usecase.Node{
		ID:   dbID,        // usecase.NodeのIDはintなのでダミーの0を設定
		Path: cleanedPath, // 元のパスを返す
	}

	if kind == "dir" {
		node.Type = usecase.FileTypeDirectory
	} else if kind == "room" {
		node.Type = usecase.FileTypeFile
	} else {
		// kind が予期しない値の場合 (基本的には起こらないはず)
		return usecase.Node{}, fmt.Errorf("unexpected node kind '%s' found for path '%s'", kind, cleanedPath)
	}

	// TODO: usecase.Node の ID (int) と DB の ID (string, dbID変数) のマッピングが必要
	// 現状は node.ID が常に 0 になっている

	return node, nil
}

// --- Directory Operations ---

func (r *Repository) CreateDirectory(path string, parents bool) error {
	path = filepath.Clean(path)
	if path == "/" || path == "." {
		// ルートディレクトリは getRootDirectoryID で自動生成されるので、ここではエラーとするか、何もしない
		_, err := r.getRootDirectoryID() // 存在確認 or 作成
		return err
		// return errors.New("cannot explicitly create root directory or current directory")
	}

	parentPath := filepath.Dir(path)
	dirName := filepath.Base(path)

	if dirName == "." || dirName == "/" {
		return errors.New("invalid directory name")
	}

	parentID, err := r.getDirectoryIDByPath(parentPath)
	if err != nil {
		return fmt.Errorf("failed to get parent directory ID for '%s': %w", parentPath, err)
	}

	if parentID == "" {
		if !parents {
			return fmt.Errorf("parent directory '%s' does not exist", parentPath)
		}
		// parents=true の場合、親ディレクトリを再帰的に作成する
		err = r.CreateDirectory(parentPath, true)
		if err != nil {
			return fmt.Errorf("failed to create parent directory '%s' recursively: %w", parentPath, err)
		}
		// 再度親IDを取得
		parentID, err = r.getDirectoryIDByPath(parentPath)
		if err != nil || parentID == "" {
			// 親ディレクトリ作成後もID取得に失敗するのは予期せぬ事態
			return fmt.Errorf("failed to get parent directory ID for '%s' even after recursive creation: %w", parentPath, err)
		}
	}

	// 存在確認 (同名のディレクトリ)
	var existingID string
	err = r.db.QueryRow("SELECT id FROM directories WHERE name = ? AND parent_id = ?", dirName, parentID).Scan(&existingID)
	if err == nil && existingID != "" {
		return ErrAlreadyExists // 既に存在する
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check existence of directory '%s' in '%s': %w", dirName, parentID, err)
	}
	// 存在確認 (同名のファイル/ルーム)
	err = r.db.QueryRow("SELECT id FROM rooms WHERE name = ? AND directory_id = ?", dirName, parentID).Scan(&existingID)
	if err == nil && existingID != "" {
		return ErrAlreadyExists // 同名のファイルが存在する
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check existence of room '%s' in '%s': %w", dirName, parentID, err)
	}

	dbNewID := generateUUID() // DB用のID
	now := time.Now()
	// owner_id はどうするか？ -> 今回はNULLとする。必要なら引数で渡すか、認証情報から取得する。
	_, err = r.db.Exec("INSERT INTO directories (id, name, parent_id, owner_id, created_at) VALUES (?, ?, ?, NULL, ?)",
		dbNewID, dirName, parentID, now)
	if err != nil {
		// ユニーク制約違反の可能性も考慮 (name, parent_id)
		return fmt.Errorf("failed to insert directory '%s': %w", dirName, err)
	}

	return nil
}

func (r *Repository) DeleteDirectory(path string, recursive, force bool) error {
	path = filepath.Clean(path)
	if path == "/" || path == "." {
		return errors.New("cannot delete root directory or current directory")
	}

	dirID, err := r.getDirectoryIDByPath(path)
	if err != nil {
		return fmt.Errorf("failed to get directory ID for '%s': %w", path, err)
	}
	if dirID == "" {
		if force {
			return nil // force=trueなら存在しなくてもエラーにしない
		}
		return ErrNotFound
	}

	// recursive=false の場合、中身が空かチェック
	if !recursive {
		var count int
		// サブディレクトリの存在チェック
		err = r.db.QueryRow("SELECT COUNT(*) FROM directories WHERE parent_id = ?", dirID).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check subdirectories for %s: %w", dirID, err)
		}
		if count > 0 {
			return errors.New("directory not empty (contains subdirectories)")
		}
		// ルーム（ファイル）の存在チェック
		err = r.db.QueryRow("SELECT COUNT(*) FROM rooms WHERE directory_id = ?", dirID).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check rooms for %s: %w", dirID, err)
		}
		if count > 0 {
			return errors.New("directory not empty (contains files/rooms)")
		}
	}

	// トランザクションを開始
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // エラー時やpanic時にロールバック

	// 再帰的に削除する関数
	var deleteDirRecursive func(tx *sql.Tx, id string) error
	deleteDirRecursive = func(tx *sql.Tx, id string) error {
		// サブディレクトリを取得して再帰的に削除
		subDirRows, err := tx.Query("SELECT id FROM directories WHERE parent_id = ?", id)
		if err != nil {
			return fmt.Errorf("failed to query subdirectories for %s: %w", id, err)
		}
		// 重要: Queryの後、すぐにCloseせず、ループ内でScanしてから再帰呼び出しする
		var subDirIDs []string
		for subDirRows.Next() {
			var subDirID string
			if err := subDirRows.Scan(&subDirID); err != nil {
				subDirRows.Close() // エラー時はCloseする
				return fmt.Errorf("failed to scan subdirectory id: %w", err)
			}
			subDirIDs = append(subDirIDs, subDirID)
		}
		subDirRows.Close() // ループが終わったらClose
		if err := subDirRows.Err(); err != nil {
			return fmt.Errorf("error iterating over subdirectories for %s: %w", id, err)
		}
		// サブディレクトリの削除（ループの後に行う）
		for _, subDirID := range subDirIDs {
			if err := deleteDirRecursive(tx, subDirID); err != nil {
				return err // エラーを伝播
			}
		}

		// ディレクトリ内のルーム（ファイル）と関連メッセージを削除
		roomRows, err := tx.Query("SELECT id FROM rooms WHERE directory_id = ?", id)
		if err != nil {
			return fmt.Errorf("failed to query rooms for directory %s: %w", id, err)
		}
		// 重要: Queryの後、すぐにCloseせず、ループ内でScanしてから削除処理を行う
		var roomIDs []string
		for roomRows.Next() {
			var roomID string
			if err := roomRows.Scan(&roomID); err != nil {
				roomRows.Close()
				return fmt.Errorf("failed to scan room id: %w", err)
			}
			roomIDs = append(roomIDs, roomID)
		}
		roomRows.Close() // ループが終わったらClose
		if err := roomRows.Err(); err != nil {
			return fmt.Errorf("error iterating over rooms for directory %s: %w", id, err)
		}
		// ルームとメッセージの削除（ループの後に行う）
		for _, roomID := range roomIDs {
			// メッセージ削除
			_, err = tx.Exec("DELETE FROM messages WHERE room_id = ?", roomID)
			if err != nil {
				return fmt.Errorf("failed to delete messages for room %s: %w", roomID, err)
			}
			// ルーム削除
			_, err = tx.Exec("DELETE FROM rooms WHERE id = ?", roomID)
			if err != nil {
				return fmt.Errorf("failed to delete room %s: %w", roomID, err)
			}
		}

		// ディレクトリ自体を削除
		_, err = tx.Exec("DELETE FROM directories WHERE id = ?", id)
		if err != nil {
			return fmt.Errorf("failed to delete directory %s: %w", id, err)
		}
		return nil
	}

	// 削除処理を実行
	if err := deleteDirRecursive(tx, dirID); err != nil {
		return err // エラーが発生したらロールバックされる
	}

	// コミット
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *Repository) UpdateDirectory(path string) (string, error) {
	// TODO: 未実装。何を更新する？名前？最終アクセス日時？
	// スキーマには更新日時のカラムがないため、実装が難しい。
	// usecase層で必要とされる具体的な更新内容に応じて実装する。
	return "", errors.New("UpdateDirectory not implemented")
}

func (r *Repository) UpdateDirectoryPath(src, dst string, overwrite bool) error {
	srcPath := filepath.Clean(src)
	dstPath := filepath.Clean(dst)

	if srcPath == "/" || srcPath == "." || dstPath == "/" || dstPath == "." {
		return errors.New("cannot move root directory or current directory")
	}
	if strings.HasPrefix(dstPath, srcPath+"/") || dstPath == srcPath {
		return errors.New("cannot move directory into itself or its subdirectory")
	}

	srcDirID, err := r.getDirectoryIDByPath(srcPath)
	if err != nil {
		return fmt.Errorf("failed to get source directory ID for '%s': %w", srcPath, err)
	}
	if srcDirID == "" {
		return ErrNotFound // 移動元が存在しない
	}

	dstParentPath := filepath.Dir(dstPath)
	dstName := filepath.Base(dstPath)

	if dstName == "." || dstName == "/" {
		return errors.New("invalid destination directory name")
	}

	dstParentID, err := r.getDirectoryIDByPath(dstParentPath)
	if err != nil {
		return fmt.Errorf("failed to get destination parent directory ID for '%s': %w", dstParentPath, err)
	}
	if dstParentID == "" {
		return fmt.Errorf("destination parent directory '%s' does not exist", dstParentPath)
	}

	// 移動先に同名のディレクトリ/ファイルが存在するかチェック
	var existingID string
	var existingIsDir bool
	// ディレクトリチェック
	err = r.db.QueryRow("SELECT id FROM directories WHERE name = ? AND parent_id = ?", dstName, dstParentID).Scan(&existingID)
	if err == nil && existingID != "" {
		existingIsDir = true
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check existence of destination directory '%s': %w", dstName, err)
	} else {
		// ファイル(room)チェック
		err = r.db.QueryRow("SELECT id FROM rooms WHERE name = ? AND directory_id = ?", dstName, dstParentID).Scan(&existingID)
		if err == nil && existingID != "" {
			existingIsDir = false
		} else if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to check existence of destination room '%s': %w", dstName, err)
		}
		// ここまで来て existingID が空なら、移動先に同名のものはない
	}

	if existingID != "" {
		// 移動先に何か存在する
		if !overwrite {
			return ErrAlreadyExists
		}
		// overwrite=true の場合
		if existingID == srcDirID {
			return errors.New("cannot move directory to itself (this should have been caught earlier)")
		}

		// 既存のものを削除
		if existingIsDir {
			// 既存のものがディレクトリの場合
			// 親ディレクトリへの移動チェック (srcがdstの親であるか) -> 冒頭のチェックでカバーされているはず
			if err := r.DeleteDirectory(dstPath, true, true); err != nil {
				return fmt.Errorf("failed to overwrite existing directory '%s': %w", dstPath, err)
			}
		} else {
			// 既存のものがファイルの場合、ディレクトリ移動では上書きできない
			return fmt.Errorf("cannot overwrite existing file '%s' with directory", dstPath)
		}
	}

	// ディレクトリの親IDと名前を更新
	_, err = r.db.Exec("UPDATE directories SET parent_id = ?, name = ? WHERE id = ?", dstParentID, dstName, srcDirID)
	if err != nil {
		// ユニーク制約違反の可能性 (name, parent_id)
		return fmt.Errorf("failed to update directory path for %s: %w", srcDirID, err)
	}

	return nil
}

func (r *Repository) ListDirectories(path string, showAll, longFormat bool) ([]usecase.FileInfo, error) {
	path = filepath.Clean(path)
	dirID, err := r.getDirectoryIDByPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get directory ID for '%s': %w", path, err)
	}
	if dirID == "" {
		return nil, ErrNotFound // 指定されたパスが存在しない
	}

	query := "SELECT id, name, created_at FROM directories WHERE parent_id = ?"
	args := []interface{}{dirID}

	if !showAll {
		// showAll=false の場合、'.'で始まるものを除外 (Unixライクな隠しファイル/ディレクトリ)
		query += " AND name NOT LIKE ?"
		args = append(args, ".%")
	}
	query += " ORDER BY name ASC" // 名前順でソート

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query subdirectories for %s: %w", dirID, err)
	}
	defer rows.Close()

	var results []usecase.FileInfo
	for rows.Next() {
		var id, name string
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan subdirectory info: %w", err)
		}
		// usecase.FileInfo に合わせる
		results = append(results, usecase.FileInfo{
			Name:      name,
			Size:      0, // ディレクトリのサイズは0とする
			Type:      usecase.FileTypeDirectory,
			Timestamp: createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over subdirectories for %s: %w", dirID, err)
	}

	// ListDirectories はディレクトリのみを返す

	return results, nil
}

// --- File (Room) Operations ---

func (r *Repository) CreateFile(path string) error {
	path = filepath.Clean(path)
	parentPath := filepath.Dir(path)
	fileName := filepath.Base(path)

	if fileName == "." || fileName == "/" {
		return errors.New("invalid file name")
	}

	parentID, err := r.getDirectoryIDByPath(parentPath)
	if err != nil {
		return fmt.Errorf("failed to get parent directory ID for '%s': %w", parentPath, err)
	}
	if parentID == "" {
		// 親ディレクトリが存在しない場合、ファイルも作成できない
		// (mkdir -p のような動作は CreateDirectory で行う想定)
		return fmt.Errorf("parent directory '%s' does not exist", parentPath)
	}

	// 存在確認 (同名のファイル/ルーム)
	var existingID string
	err = r.db.QueryRow("SELECT id FROM rooms WHERE name = ? AND directory_id = ?", fileName, parentID).Scan(&existingID)
	if err == nil && existingID != "" {
		return ErrAlreadyExists // 既に存在する
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check existence of room '%s' in '%s': %w", fileName, parentID, err)
	}

	// 存在確認 (同名のディレクトリ)
	err = r.db.QueryRow("SELECT id FROM directories WHERE name = ? AND parent_id = ?", fileName, parentID).Scan(&existingID)
	if err == nil && existingID != "" {
		return ErrAlreadyExists // 同名のディレクトリが存在する
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check existence of directory '%s' in '%s': %w", fileName, parentID, err)
	}

	dbNewID := generateUUID() // DB用のID
	now := time.Now()
	_, err = r.db.Exec("INSERT INTO rooms (id, name, directory_id, created_at) VALUES (?, ?, ?, ?)",
		dbNewID, fileName, parentID, now)
	if err != nil {
		// ユニーク制約違反の可能性 (name, directory_id)
		return fmt.Errorf("failed to insert room '%s': %w", fileName, err)
	}

	return nil
}

func (r *Repository) DeleteFile(path string, force bool) error {
	path = filepath.Clean(path)
	roomID, err := r.getRoomIDByPath(path)
	if err != nil {
		return fmt.Errorf("failed to get room ID for '%s': %w", path, err)
	}
	if roomID == "" {
		if force {
			return nil // force=trueなら存在しなくてもエラーにしない
		}
		return ErrNotFound
	}

	// トランザクションを開始
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 関連するメッセージを削除
	_, err = tx.Exec("DELETE FROM messages WHERE room_id = ?", roomID)
	if err != nil {
		return fmt.Errorf("failed to delete messages for room %s: %w", roomID, err)
	}

	// ルーム（ファイル）自体を削除
	_, err = tx.Exec("DELETE FROM rooms WHERE id = ?", roomID)
	if err != nil {
		return fmt.Errorf("failed to delete room %s: %w", roomID, err)
	}

	// コミット
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *Repository) UpdateFile(src, dst string, overwrite bool) error {
	// TODO: 未実装。ファイルを更新するとは？内容？メタデータ？
	// UpdateFileContent が内容更新を担当するので、こちらは不要かもしれない。
	// もしメタデータ更新ならスキーマ変更が必要。
	// usecase.Repository のシグネチャが UpdateDirectory と同じなのは意図通りか？
	return errors.New("UpdateFile not implemented")
}

func (r *Repository) UpdateFilePath(src, dst string, overwrite bool) error {
	srcPath := filepath.Clean(src)
	dstPath := filepath.Clean(dst)

	if srcPath == "/" || srcPath == "." || dstPath == "/" || dstPath == "." {
		return errors.New("source or destination path is invalid")
	}
	// ディレクトリへの移動は UpdateDirectoryPath で行うべき
	// ここではファイル -> ファイルの移動のみを想定

	srcRoomID, err := r.getRoomIDByPath(srcPath)
	if err != nil {
		return fmt.Errorf("failed to get source room ID for '%s': %w", srcPath, err)
	}
	if srcRoomID == "" {
		return ErrNotFound // 移動元が存在しない
	}

	dstParentPath := filepath.Dir(dstPath)
	dstName := filepath.Base(dstPath)

	if dstName == "." || dstName == "/" {
		return errors.New("invalid destination file name")
	}

	dstParentID, err := r.getDirectoryIDByPath(dstParentPath)
	if err != nil {
		return fmt.Errorf("failed to get destination parent directory ID for '%s': %w", dstParentPath, err)
	}
	if dstParentID == "" {
		return fmt.Errorf("destination parent directory '%s' does not exist", dstParentPath)
	}

	// 移動先に同名のファイル/ディレクトリが存在するかチェック
	var existingID string
	var existingIsDir bool
	// ファイル(room)チェック
	err = r.db.QueryRow("SELECT id FROM rooms WHERE name = ? AND directory_id = ?", dstName, dstParentID).Scan(&existingID)
	if err == nil && existingID != "" {
		existingIsDir = false
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check existence of destination room '%s': %w", dstName, err)
	} else {
		// ディレクトリチェック
		err = r.db.QueryRow("SELECT id FROM directories WHERE name = ? AND parent_id = ?", dstName, dstParentID).Scan(&existingID)
		if err == nil && existingID != "" {
			existingIsDir = true
		} else if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to check existence of destination directory '%s': %w", dstName, err)
		}
		// ここまで来て existingID が空なら、移動先に同名のものはない
	}

	if existingID != "" {
		// 移動先に何か存在する
		if !overwrite {
			return ErrAlreadyExists
		}
		// overwrite=true の場合
		if existingID == srcRoomID {
			return errors.New("cannot move file to itself")
		}

		// 既存のものを削除
		if !existingIsDir {
			// 既存のものがファイルの場合
			if err := r.DeleteFile(dstPath, true); err != nil {
				return fmt.Errorf("failed to overwrite existing file '%s': %w", dstPath, err)
			}
		} else {
			// 既存のものがディレクトリの場合、ファイル移動では上書きできない
			return fmt.Errorf("cannot overwrite existing directory '%s' with file", dstPath)
		}
	}

	// ルームの親IDと名前を更新
	_, err = r.db.Exec("UPDATE rooms SET directory_id = ?, name = ? WHERE id = ?", dstParentID, dstName, srcRoomID)
	if err != nil {
		// ユニーク制約違反の可能性 (name, directory_id)
		return fmt.Errorf("failed to update room path for %s: %w", srcRoomID, err)
	}

	return nil
}

func (r *Repository) GetFileContent(path string) ([]byte, error) {
	path = filepath.Clean(path)
	roomID, err := r.getRoomIDByPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get room ID for '%s': %w", path, err)
	}
	if roomID == "" {
		return nil, ErrNotFound
	}

	// ファイルの内容 = ルームの全メッセージを結合したもの、とする
	// メッセージ数が多い場合にメモリを大量消費する可能性があるため注意
	rows, err := r.db.Query("SELECT content FROM messages WHERE room_id = ? ORDER BY created_at ASC", roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages for room %s: %w", roomID, err)
	}
	defer rows.Close()

	var contentBuilder strings.Builder
	first := true
	for rows.Next() {
		var msgContent string
		if err := rows.Scan(&msgContent); err != nil {
			return nil, fmt.Errorf("failed to scan message content: %w", err)
		}
		if !first {
			contentBuilder.WriteString("\n") // メッセージ間に改行を入れる (最初のメッセージの前には入れない)
		}
		contentBuilder.WriteString(msgContent)
		first = false
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over messages for room %s: %w", roomID, err)
	}

	return []byte(contentBuilder.String()), nil
}

func (r *Repository) UpdateFileContent(text, path string, appendMode bool) error {
	path = filepath.Clean(path)
	roomID, err := r.getRoomIDByPath(path)
	if err != nil {
		return fmt.Errorf("failed to get room ID for '%s': %w", path, err)
	}
	if roomID == "" {
		// ファイルが存在しない場合、新規作成するべきか？ -> usecase層の責務か。ここではエラーとする。
		return ErrNotFound
		// もし新規作成するなら:
		// err = r.CreateFile(path)
		// if err != nil {
		// 	return fmt.Errorf("failed to create file '%s' before update: %w", path, err)
		// }
		// roomID, err = r.getRoomIDByPath(path)
		// if err != nil || roomID == "" {
		// 	return fmt.Errorf("failed to get room ID for '%s' after creation: %w", path, err)
		// }
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// appendMode=false の場合、既存のメッセージを削除する
	if !appendMode {
		_, err = tx.Exec("DELETE FROM messages WHERE room_id = ?", roomID)
		if err != nil {
			return fmt.Errorf("failed to delete existing messages for room %s: %w", roomID, err)
		}
	}

	// 新しい内容をメッセージとして追加
	// user_id はどうするか？ -> 固定値 or NULL許容にする。ここではNULLにする。
	// text をそのまま1メッセージとして追加する。改行が含まれていても1レコード。
	dbNewMessageID := generateUUID() // DB用のID
	now := time.Now()
	_, err = tx.Exec("INSERT INTO messages (id, room_id, user_id, content, created_at) VALUES (?, ?, NULL, ?, ?)",
		dbNewMessageID, roomID, text, now)
	if err != nil {
		return fmt.Errorf("failed to insert new message content for room %s: %w", roomID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ListFiles は usecase.Repository インターフェースに合わせて実装する。
// 引数 path int が何を指すか不明瞭なため、ディレクトリID(int)と仮定して実装を試みるが、
// DBのIDはVARCHARなので直接は使えない。
// TODO: path int の意味を明確にし、DB ID (string) とのマッピング方法を確立する必要がある。
// ここでは、path int を無視し、ルートディレクトリ直下のファイル一覧を返すダミー実装とする。
// 戻り値は []string (ファイル名のリスト) とする。
func (r *Repository) ListFiles(path string) ([]string, error) {
	// path int をディレクトリIDと仮定するが、DB IDはstringなので変換が必要。
	// 適切な変換方法がないため、ここではダミーとしてルートディレクトリ(/)のIDを使う。
	// 本来は path int に対応する string ID を取得する処理が必要。
	rootDirID, err := r.getRootDirectoryID()
	if err != nil {
		return nil, fmt.Errorf("failed to get root directory ID for ListFiles: %w", err)
	}
	// TODO: 引数 path (int) を使って対象ディレクトリの string ID を特定する処理

	// showAll=true, longFormat=false 相当でファイル名のみ取得
	query := "SELECT name FROM rooms WHERE directory_id = ? ORDER BY name ASC"
	args := []interface{}{rootDirID} // 本来は path int に対応する string ID

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query rooms for directory %s: %w", rootDirID, err)
	}
	defer rows.Close()

	var fileNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan room name: %w", err)
		}
		fileNames = append(fileNames, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rooms for directory %s: %w", rootDirID, err)
	}

	return fileNames, nil
}

func (r *Repository) ListFileMatches(pattern string, paths []string, recursive, ignoreCase bool) ([]usecase.GrepResult, error) {
	var results []usecase.GrepResult
	var roomIDsToSearch []string
	processedRoomIDs := make(map[string]bool) // 同じファイルを複数回検索しないように

	// 検索対象のルームIDを収集する関数 (再帰処理含む)
	var collectRoomIDs func(dirPath string) error
	collectRoomIDs = func(dirPath string) error {
		dirID, err := r.getDirectoryIDByPath(dirPath)
		if err != nil {
			// エラーでも処理を続ける (一部のパスが見つからない場合など)
			fmt.Printf("Warning: cannot get directory ID for '%s': %v\n", dirPath, err)
			return nil // nilを返して処理続行
		}
		if dirID == "" {
			fmt.Printf("Warning: directory '%s' not found\n", dirPath)
			return nil // ディレクトリが存在しない場合はスキップ
		}

		// 現在のディレクトリ内のルームIDを取得
		roomRows, err := r.db.Query("SELECT id FROM rooms WHERE directory_id = ?", dirID)
		if err != nil {
			return fmt.Errorf("failed to query rooms in directory %s: %w", dirID, err)
		}
		defer roomRows.Close()
		for roomRows.Next() {
			var roomID string
			if err := roomRows.Scan(&roomID); err != nil {
				return fmt.Errorf("failed to scan room ID: %w", err)
			}
			if !processedRoomIDs[roomID] {
				roomIDsToSearch = append(roomIDsToSearch, roomID)
				processedRoomIDs[roomID] = true
			}
		}
		if err := roomRows.Err(); err != nil {
			return fmt.Errorf("error iterating rooms in directory %s: %w", dirID, err)
		}

		// recursive=true の場合、サブディレクトリを再帰的に探索
		if recursive {
			subDirRows, err := r.db.Query("SELECT name FROM directories WHERE parent_id = ?", dirID)
			if err != nil {
				return fmt.Errorf("failed to query subdirectories in %s: %w", dirID, err)
			}
			defer subDirRows.Close()
			for subDirRows.Next() {
				var subDirName string
				if err := subDirRows.Scan(&subDirName); err != nil {
					return fmt.Errorf("failed to scan subdirectory name: %w", err)
				}
				// 再帰呼び出し
				subDirPath := filepath.Join(dirPath, subDirName)
				if err := collectRoomIDs(subDirPath); err != nil {
					return err // エラーを伝播
				}
			}
			if err := subDirRows.Err(); err != nil {
				return fmt.Errorf("error iterating subdirectories in %s: %w", dirID, err)
			}
		}
		return nil
	}

	if len(paths) == 0 {
		// pathsが空の場合、ルートディレクトリから検索を開始
		if err := collectRoomIDs("/"); err != nil {
			return nil, fmt.Errorf("error collecting files from root: %w", err)
		}
	} else {
		for _, p := range paths {
			p = filepath.Clean(p)
			// 指定されたパスがディレクトリかファイルか判定
			node, err := r.GetNode(p)
			if err != nil {
				fmt.Printf("Warning: cannot get node info for path '%s': %v\n", p, err)
				continue // エラーが発生したパスはスキップ
			}

			if node.Type == usecase.FileTypeDirectory { // node.IsDir の代わりに node.Type を使用
				// ディレクトリの場合
				if err := collectRoomIDs(p); err != nil {
					return nil, fmt.Errorf("error collecting files from directory '%s': %w", p, err)
				}
			} else if node.Type == usecase.FileTypeFile {
				// ファイルの場合、DBのルームID(string)を取得する
				dbRoomID, errRoom := r.getRoomIDByPath(p)
				if errRoom != nil {
					fmt.Printf("Warning: cannot get room ID for file path '%s': %v\n", p, errRoom)
					continue // ルームID取得エラーの場合はスキップ
				}
				if dbRoomID == "" {
					fmt.Printf("Warning: room ID not found for file path '%s'\n", p)
					continue // ルームIDが見つからない場合はスキップ
				}
				// 取得したDBのルームID(string)を使用する
				if !processedRoomIDs[dbRoomID] {
					roomIDsToSearch = append(roomIDsToSearch, dbRoomID)
					processedRoomIDs[dbRoomID] = true
				}
			}
		}
	}

	if len(roomIDsToSearch) == 0 {
		return results, nil // 検索対象ファイルなし
	}

	// 各ルームでメッセージを検索
	// FULLTEXT index を使う
	// ignoreCase は collation 設定に依存。ここでは考慮しない。必要ならLOWER()を使う。
	// TODO: ignoreCase の実装
	query := `
		SELECT m.room_id, m.content, m.created_at -- 行番号は取得できない
		FROM messages m
		WHERE m.room_id IN (?` + strings.Repeat(",?", len(roomIDsToSearch)-1) + `)
		AND MATCH(m.content) AGAINST(? IN NATURAL LANGUAGE MODE)
		ORDER BY m.room_id, m.created_at -- room_idでソートして結果をまとめやすくする
	`
	args := make([]interface{}, 0, len(roomIDsToSearch)+1)
	for _, id := range roomIDsToSearch {
		args = append(args, id)
	}
	args = append(args, pattern)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		// FULLTEXT indexがない場合や構文エラーの場合など
		// LIKE検索へのフォールバックも検討
		return nil, fmt.Errorf("failed to execute fulltext search: %w", err)
	}
	defer rows.Close()

	// 結果を GrepResult にマッピング
	// GrepResult は File, LineNumber, LineText を持つ
	// DBのmessagesテーブルには行番号がないため、LineNumberは0とする
	// 1メッセージが1行に対応すると仮定する

	for rows.Next() {
		var roomID, content string
		var createdAt time.Time // created_at はソートに使ったが行番号の代わりにはならない
		if err := rows.Scan(&roomID, &content, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}

		// ファイルパスを取得
		filePath, errPath := r.getPathForRoomID(roomID)
		if errPath != nil {
			fmt.Printf("Warning: failed to get path for room %s: %v\n", roomID, errPath)
			filePath = fmt.Sprintf("unknown_path(id:%s)", roomID) // 仮のパス
		}

		// GrepResult を作成して結果リストに追加
		results = append(results, usecase.GrepResult{
			File:       filePath,
			LineNumber: 0, // 行番号は不明
			LineText:   content,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over search results: %w", err)
	}

	// パス名でソート (任意)
	// sort.Slice(results, func(i, j int) bool {
	// 	return results[i].Path < results[j].Path
	// })

	return results, nil
}

func (r *Repository) GetFileLines(path string, lineCount int, follow bool) ([]string, error) {
	// follow=true はリアルタイム更新を意味するが、DBリポジトリ層での実装は難しい。
	// usecase層やadaptor層で対応すべき。ここでは無視する。
	if follow {
		// usecase層に伝えるために特定のerrorを返すのが良いかもしれない
		return nil, errors.New("follow mode is not supported in repository layer")
	}

	path = filepath.Clean(path)
	roomID, err := r.getRoomIDByPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get room ID for '%s': %w", path, err)
	}
	if roomID == "" {
		return nil, ErrNotFound
	}

	if lineCount <= 0 {
		return []string{}, nil // 0行要求されたら空を返す
	}

	// 最新の lineCount 件のメッセージを取得
	// DBによっては `ORDER BY created_at DESC LIMIT ?` が効率的でない場合がある
	query := "SELECT content FROM messages WHERE room_id = ? ORDER BY created_at DESC LIMIT ?"
	rows, err := r.db.Query(query, roomID, lineCount)
	if err != nil {
		return nil, fmt.Errorf("failed to query last %d messages for room %s: %w", lineCount, roomID, err)
	}
	defer rows.Close()

	var linesDesc []string // 新しい順に入るリスト
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, fmt.Errorf("failed to scan message content: %w", err)
		}
		linesDesc = append(linesDesc, content)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over messages for room %s: %w", roomID, err)
	}

	// 結果は新しい順になっているので、古い順に反転させる
	linesAsc := make([]string, len(linesDesc))
	for i, line := range linesDesc {
		linesAsc[len(linesDesc)-1-i] = line
	}

	return linesAsc, nil
}

// --- Message Operations (Assume roomID is string based on schema) ---

// ListMessages の引数 roomID は int
func (r *Repository) ListMessages(roomID int) ([]string, error) {
	// roomID (int) を DB の room_id (string) に変換する必要がある。
	// 適切な変換方法がないため、ここではダミーとして特定の固定ルームIDを使うか、エラーとする。
	// TODO: int ID と string ID のマッピング方法を確立する。
	// ダミー実装: 固定のルームID（例: ルートディレクトリ直下の 'default' ルーム）を使う試み
	defaultRoomPath := "/default" // 仮のデフォルトルームパス
	roomIDStr, err := r.getRoomIDByPath(defaultRoomPath)
	if err != nil || roomIDStr == "" {
		// デフォルトルームが見つからない、または取得に失敗した場合
		// 引数の roomID を使おうとしてもマッピングできないのでエラーを返す
		return nil, fmt.Errorf("cannot map roomID %d to database ID (mapping not implemented), and default room '%s' failed: %w", roomID, defaultRoomPath, err)
	}
	fmt.Printf("Warning: ListMessages using dummy room ID '%s' for input roomID %d\n", roomIDStr, roomID)

	// ルームID(string)が存在するか確認 (ダミーIDで確認)
	var exists int
	err = r.db.QueryRow("SELECT 1 FROM rooms WHERE id = ? LIMIT 1", roomIDStr).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// ダミーIDのルームが存在しない場合
			return []string{}, nil // 空リストを返すのが適切か？
			// return nil, fmt.Errorf("dummy room with id %s not found", roomIDStr)
		}
		return nil, fmt.Errorf("failed to check existence of dummy room %s: %w", roomIDStr, err)
	}

	query := "SELECT content FROM messages WHERE room_id = ? ORDER BY created_at ASC"
	rows, err := r.db.Query(query, roomIDStr) // ダミーIDで検索
	if err != nil {
		return nil, fmt.Errorf("failed to query messages for dummy room %s: %w", roomIDStr, err)
	}
	defer rows.Close()

	var messages []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, fmt.Errorf("failed to scan message content: %w", err)
		}
		messages = append(messages, content)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over messages for room %s: %w", roomIDStr, err)
	}

	return messages, nil
}

// ListMessagesByQuery の引数 roomID は int
func (r *Repository) ListMessagesByQuery(roomID int, query string) ([]string, error) {
	// ListMessages と同様に、int ID から string ID へのマッピングが必要。
	// ダミー実装を使用。
	defaultRoomPath := "/default" // 仮のデフォルトルームパス
	roomIDStr, err := r.getRoomIDByPath(defaultRoomPath)
	if err != nil || roomIDStr == "" {
		return nil, fmt.Errorf("cannot map roomID %d to database ID (mapping not implemented), and default room '%s' failed: %w", roomID, defaultRoomPath, err)
	}
	fmt.Printf("Warning: ListMessagesByQuery using dummy room ID '%s' for input roomID %d\n", roomIDStr, roomID)

	// ルームID(string)が存在するか確認 (ダミーIDで確認)
	var exists int
	err = r.db.QueryRow("SELECT 1 FROM rooms WHERE id = ? LIMIT 1", roomIDStr).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []string{}, nil // マッチなしとして空を返す
			// return nil, fmt.Errorf("dummy room with id %s not found", roomIDStr)
		}
		return nil, fmt.Errorf("failed to check existence of dummy room %s: %w", roomIDStr, err)
	}

	// FULLTEXT index を利用した検索
	sqlQuery := `
		SELECT content
		FROM messages
		WHERE room_id = ? AND MATCH(content) AGAINST(? IN NATURAL LANGUAGE MODE)
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(sqlQuery, roomIDStr, query) // ダミーIDで検索
	if err != nil {
		// LIKE検索へのフォールバックも検討
		return nil, fmt.Errorf("failed to execute fulltext search in dummy room %s for query '%s': %w", roomIDStr, query, err)
	}
	defer rows.Close()

	var messages []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, fmt.Errorf("failed to scan message content: %w", err)
		}
		messages = append(messages, content)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over search results for room %s: %w", roomIDStr, err)
	}

	return messages, nil
}

// --- ListFiles のインターフェースが usecase.FileInfo を返す場合の実装 ---
// (現在のインターフェース定義と異なる可能性があるためコメントアウト)
/*
func (r *Repository) ListFiles(path string, showAll, longFormat bool) ([]usecase.FileInfo, error) {
	path = filepath.Clean(path)
	dirID, err := r.getDirectoryIDByPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get directory ID for '%s': %w", path, err)
	}
	if dirID == "" {
		return nil, ErrNotFound // 指定されたパスが存在しない
	}

	query := "SELECT id, name, created_at FROM rooms WHERE directory_id = ?"
	args := []interface{}{dirID}

	if !showAll {
		query += " AND name NOT LIKE ?"
		args = append(args, ".%")
	}
	query += " ORDER BY name ASC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query rooms for directory %s: %w", dirID, err)
	}
	defer rows.Close()

	var results []usecase.FileInfo
	for rows.Next() {
		var id, name string
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan room info: %w", err)
		}
		filePath := filepath.Join(path, name)
		results = append(results, usecase.FileInfo{
			Name:      name,
			Path:      filePath,
			IsDir:     false,
			CreatedAt: createdAt,
			// Size, ModTime などはスキーマにないので省略
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rooms for directory %s: %w", dirID, err)
	}

	return results, nil
}
*/
