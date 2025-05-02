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

func (r *Repository) GetNode(path string) (usecase.Node, error) {
	cleanedPath := filepath.Clean(path)
	if cleanedPath == "/" || cleanedPath == "." {
		return usecase.Node{
			ID:   1,
			Type: usecase.FileTypeDirectory,
		}, nil
	}

	// "/a/b/c" -> ["a", "b", "c"]
	components := strings.Split(strings.TrimPrefix(cleanedPath, "/"), "/")
	currentNode := usecase.Node{
		ID:   1,
		Type: usecase.FileTypeDirectory,
	}

	// UNION ALL クエリをループ外で定義
	query := `
		SELECT 'dir' AS kind, id FROM directories WHERE parent_id = ? AND name = ?
		UNION ALL
		SELECT 'room' AS kind, id FROM rooms WHERE directory_id = ? AND name = ?
		LIMIT 1
	`

	for _, name := range components {
		if name == "" {
			continue
		}
		// 現在の親がディレクトリでない場合はエラー
		if currentNode.Type != usecase.FileTypeDirectory {
			return usecase.Node{}, fmt.Errorf("path component '%s' is not a directory", cleanedPath)
		}
		var kind string
		var dbID int
		err := r.db.QueryRow(query, currentNode.ID, name, currentNode.ID, name).Scan(&kind, &dbID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return usecase.Node{}, ErrNotFound
			}
			return usecase.Node{}, fmt.Errorf("error querying node '%s' in parent %d (path: %s): %w", name, currentNode.ID, cleanedPath, err)
		}
		currentNode.ID = dbID
		switch kind {
		case "dir":
			currentNode.Type = usecase.FileTypeDirectory
		case "room":
			currentNode.Type = usecase.FileTypeFile
		default:
			return usecase.Node{}, fmt.Errorf("unexpected node kind '%s' found for component '%s' in path '%s'", kind, name, cleanedPath)
		}
	}
	return currentNode, nil
}

func (r *Repository) CreateDirectory(path string, ownerID int, parents bool) error {
	cleanedPath := filepath.Clean(path)
	if cleanedPath == "/" || cleanedPath == "." {
		return errors.New("cannot explicitly create root directory or current directory")
	}

	parentPath := filepath.Dir(cleanedPath)
	dirName := filepath.Base(cleanedPath)

	if dirName == "." || dirName == "/" {
		return errors.New("invalid directory name")
	}

	parentNode, err := r.GetNode(parentPath)
	if err != nil {
		if errors.Is(err, ErrNotFound) && parents {
			// 親ディレクトリが存在しない場合、親ディレクトリを再帰的に作成
			if err := r.CreateDirectory(parentPath, ownerID, parents); err != nil {
				return fmt.Errorf("failed to create parent directory '%s': %w", parentPath, err)
			}
			// 再度親ノードを取得
			parentNode, err = r.GetNode(parentPath)
			if err != nil {
				return fmt.Errorf("failed to get parent node for '%s': %w", parentPath, err)
			}
		} else {
			return fmt.Errorf("failed to get parent node for '%s': %w", parentPath, err)
		}
	}

	if parentNode.Type != usecase.FileTypeDirectory {
		return fmt.Errorf("parent path '%s' is not a directory", parentPath)
	}
	_, err = r.GetNode(cleanedPath)
	if err == nil {
		return ErrAlreadyExists
	}
	if !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("failed to check existence of path '%s': %w", cleanedPath, err)
	}

	now := time.Now()
	_, err = r.db.Exec("INSERT INTO directories (name, parent_id, owner_id, created_at) VALUES (?, ?, ?, ?)",
		dirName, parentNode.ID, ownerID, now)
	if err != nil {
		return fmt.Errorf("failed to insert directory '%s': %w", dirName, err)
	}

	return nil
}

func (r *Repository) DeleteDirectory(path string, recursive, force bool) error {
	cleanedPath := filepath.Clean(path)
	if cleanedPath == "/" || cleanedPath == "." {
		return errors.New("cannot delete root directory or current directory")
	}

	// GetNode を使用してディレクトリの情報を取得
	node, err := r.GetNode(cleanedPath)
	if err != nil {
		if errors.Is(err, ErrNotFound) && force {
			return nil // force=trueなら存在しなくてもエラーにしない
		}
		return fmt.Errorf("failed to get node for path '%s': %w", cleanedPath, err)
	}
	if node.Type != usecase.FileTypeDirectory {
		return fmt.Errorf("path '%s' is not a directory", cleanedPath)
	}
	dirID := node.ID // Use the int ID

	// recursive=false の場合、中身が空かチェック
	if !recursive {
		var count int
		// サブディレクトリの存在チェック
		err = r.db.QueryRow("SELECT COUNT(*) FROM directories WHERE parent_id = ?", dirID).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check subdirectories for %d: %w", dirID, err)
		}
		if count > 0 {
			return errors.New("directory not empty (contains subdirectories)")
		}
		// ルーム（ファイル）の存在チェック
		err = r.db.QueryRow("SELECT COUNT(*) FROM rooms WHERE directory_id = ?", dirID).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check rooms for %d: %w", dirID, err)
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

	// 再帰的に削除する関数 (id を int に変更)
	var deleteDirRecursive func(tx *sql.Tx, id int) error
	deleteDirRecursive = func(tx *sql.Tx, id int) error {
		// サブディレクトリを取得して再帰的に削除
		subDirRows, err := tx.Query("SELECT id FROM directories WHERE parent_id = ?", id) // Use int id
		if err != nil {
			// Query errors are possible, return them
			return fmt.Errorf("failed to query subdirectories for %d: %w", id, err)
		}
		defer subDirRows.Close() // Ensure rows are closed even if errors occur during iteration

		var subDirIDs []int // Store int IDs
		for subDirRows.Next() {
			var subDirID int // Scan int ID
			if err := subDirRows.Scan(&subDirID); err != nil {
				// Error during scan
				return fmt.Errorf("failed to scan subdirectory id: %w", err)
			}
			subDirIDs = append(subDirIDs, subDirID)
		}
		// Check for errors after the loop
		if err = subDirRows.Err(); err != nil {
			return fmt.Errorf("error iterating over subdirectories for %d: %w", id, err)
		}

		// サブディレクトリの削除（ループの後に行う）
		for _, subDirID := range subDirIDs {
			if err := deleteDirRecursive(tx, subDirID); err != nil {
				return err // エラーを伝播
			}
		}

		// ディレクトリ内のルーム（ファイル）と関連メッセージを削除
		roomRows, err := tx.Query("SELECT id FROM rooms WHERE directory_id = ?", id) // Use int id
		if err != nil {
			return fmt.Errorf("failed to query rooms for directory %d: %w", id, err)
		}
		defer roomRows.Close() // Ensure rows are closed

		var roomIDs []int // Store int IDs
		for roomRows.Next() {
			var roomID int // Scan int ID
			if err := roomRows.Scan(&roomID); err != nil {
				return fmt.Errorf("failed to scan room id: %w", err)
			}
			roomIDs = append(roomIDs, roomID)
		}
		// Check for errors after the loop
		if err = roomRows.Err(); err != nil {
			return fmt.Errorf("error iterating over rooms for directory %d: %w", id, err)
		}

		// ルームとメッセージの削除（ループの後に行う）
		for _, roomID := range roomIDs { // roomID is int
			// メッセージ削除
			_, err = tx.Exec("DELETE FROM messages WHERE room_id = ?", roomID) // Use int roomID
			if err != nil {
				return fmt.Errorf("failed to delete messages for room %d: %w", roomID, err)
			}
			// ルーム削除
			_, err = tx.Exec("DELETE FROM rooms WHERE id = ?", roomID) // Use int roomID
			if err != nil {
				return fmt.Errorf("failed to delete room %d: %w", roomID, err)
			}
		}

		// ディレクトリ自体を削除
		_, err = tx.Exec("DELETE FROM directories WHERE id = ?", id) // Use int id
		if err != nil {
			return fmt.Errorf("failed to delete directory %d: %w", id, err)
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

	// 移動元のノードを取得
	srcNode, err := r.GetNode(srcPath)
	if err != nil {
		return fmt.Errorf("failed to get source node for '%s': %w", srcPath, err)
	}
	if srcNode.Type != usecase.FileTypeDirectory {
		return fmt.Errorf("source path '%s' is not a directory", srcPath)
	}
	srcDirID := srcNode.ID // Use the int ID

	dstParentPath := filepath.Dir(dstPath)
	dstName := filepath.Base(dstPath)

	if dstName == "." || dstName == "/" {
		return errors.New("invalid destination directory name")
	}

	// 移動先の親ノードを取得
	dstParentNode, err := r.GetNode(dstParentPath)
	if err != nil {
		return fmt.Errorf("failed to get destination parent node for '%s': %w", dstParentPath, err)
	}
	if dstParentNode.Type != usecase.FileTypeDirectory {
		return fmt.Errorf("destination parent path '%s' is not a directory", dstParentPath)
	}
	dstParentID := dstParentNode.ID // Use the int ID

	// 移動先に同名のノードが存在するかチェック
	existingNode, err := r.GetNode(dstPath)
	if err == nil {
		// 移動先に何か存在する
		if !overwrite {
			return ErrAlreadyExists
		}
		// overwrite=true の場合
		if existingNode.ID == srcDirID {
			return errors.New("cannot move directory to itself")
		}

		// 既存のものを削除
		if existingNode.Type == usecase.FileTypeDirectory {
			// 既存のものがディレクトリの場合
			if err := r.DeleteDirectory(dstPath, true, true); err != nil { // recursive=true, force=true
				return fmt.Errorf("failed to overwrite existing directory '%s': %w", dstPath, err)
			}
		} else {
			// 既存のものがファイルの場合、ディレクトリ移動では上書きできない
			return fmt.Errorf("cannot overwrite existing file '%s' with directory", dstPath)
		}
	} else if !errors.Is(err, ErrNotFound) {
		// GetNodeで予期せぬエラー
		return fmt.Errorf("failed to check existence of destination path '%s': %w", dstPath, err)
	}
	// ErrNotFoundなら存在しないのでOK

	// ディレクトリの親IDと名前を更新 (IDはint型を使用)
	_, err = r.db.Exec("UPDATE directories SET parent_id = ?, name = ? WHERE id = ?", dstParentID, dstName, srcDirID)
	if err != nil {
		// ユニーク制約違反の可能性 (name, parent_id)
		return fmt.Errorf("failed to update directory path for %d: %w", srcDirID, err)
	}

	return nil
}

func (r *Repository) ListDirectories(path string, showAll, longFormat bool) ([]usecase.FileInfo, error) {
	cleanedPath := filepath.Clean(path)
	// GetNode を使用してディレクトリの情報を取得
	node, err := r.GetNode(cleanedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get node for path '%s': %w", cleanedPath, err)
	}
	if node.Type != usecase.FileTypeDirectory {
		return nil, fmt.Errorf("path '%s' is not a directory", cleanedPath)
	}
	dirID := node.ID // Use the int ID

	query := "SELECT id, name, created_at FROM directories WHERE parent_id = ? ORDER BY name"
	args := []interface{}{dirID}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query subdirectories for %d: %w", dirID, err)
	}
	defer rows.Close()

	var results []usecase.FileInfo
	for rows.Next() {
		var id int // Assuming DB ID is INT
		var name string
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
		return nil, fmt.Errorf("error iterating over subdirectories for %d: %w", dirID, err)
	}
	if !showAll {
		var newResults []usecase.FileInfo
		for _, result := range results {
			if strings.HasPrefix(result.Name, ".") {
				continue
			}
			newResults = append(newResults, result)
		}
		return newResults, nil
	}
	return results, nil
}

// --- File (Room) Operations ---

func (r *Repository) CreateFile(path string) error {
	cleanedPath := filepath.Clean(path)
	parentPath := filepath.Dir(cleanedPath)
	fileName := filepath.Base(cleanedPath)

	if fileName == "." || fileName == "/" || fileName == "" {
		return errors.New("invalid file name")
	}

	// 親ディレクトリのノードを取得
	parentNode, err := r.GetNode(parentPath)
	if err != nil {
		return fmt.Errorf("failed to get parent node for '%s': %w", parentPath, err)
	}
	if parentNode.Type != usecase.FileTypeDirectory {
		return fmt.Errorf("parent path '%s' is not a directory", parentPath)
	}
	parentID := parentNode.ID // 親ディレクトリのID (int)

	// 作成しようとしているパスに既に何か存在しないか確認
	_, err = r.GetNode(cleanedPath)
	if err == nil {
		return ErrAlreadyExists // 既に存在する
	}
	if !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("failed to check existence of path '%s': %w", cleanedPath, err)
	}
	// ErrNotFound の場合は存在しないのでOK

	now := time.Now()
	// DBスキーマのIDがINTであることを前提とする
	// TODO: スキーマのID型を明確にする。ここではINTと仮定。
	_, err = r.db.Exec("INSERT INTO rooms (name, directory_id, created_at) VALUES (?, ?, ?)",
		fileName, parentID, now) // Assuming ID is auto-increment
	if err != nil {
		// ユニーク制約違反の可能性 (name, directory_id)
		return fmt.Errorf("failed to insert room '%s': %w", fileName, err)
	}

	return nil
}

func (r *Repository) DeleteFile(path string, force bool) error {
	cleanedPath := filepath.Clean(path)
	// GetNode を使用してファイルの情報を取得
	node, err := r.GetNode(cleanedPath)
	if err != nil {
		if errors.Is(err, ErrNotFound) && force {
			return nil // force=trueなら存在しなくてもエラーにしない
		}
		return fmt.Errorf("failed to get node for path '%s': %w", cleanedPath, err)
	}
	if node.Type != usecase.FileTypeFile {
		return fmt.Errorf("path '%s' is not a file", cleanedPath)
	}
	roomID := node.ID // Use the int ID

	// トランザクションを開始
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 関連するメッセージを削除
	_, err = tx.Exec("DELETE FROM messages WHERE room_id = ?", roomID)
	if err != nil {
		return fmt.Errorf("failed to delete messages for room %d: %w", roomID, err)
	}

	// ルーム（ファイル）自体を削除
	_, err = tx.Exec("DELETE FROM rooms WHERE id = ?", roomID)
	if err != nil {
		return fmt.Errorf("failed to delete room %d: %w", roomID, err)
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

	// 移動元のノードを取得
	srcNode, err := r.GetNode(srcPath)
	if err != nil {
		return fmt.Errorf("failed to get source node for '%s': %w", srcPath, err)
	}
	if srcNode.Type != usecase.FileTypeFile {
		return fmt.Errorf("source path '%s' is not a file", srcPath)
	}
	srcRoomID := srcNode.ID // Use the int ID

	dstParentPath := filepath.Dir(dstPath)
	dstName := filepath.Base(dstPath)

	if dstName == "." || dstName == "/" || dstName == "" {
		return errors.New("invalid destination file name")
	}

	// 移動先の親ノードを取得
	dstParentNode, err := r.GetNode(dstParentPath)
	if err != nil {
		return fmt.Errorf("failed to get destination parent node for '%s': %w", dstParentPath, err)
	}
	if dstParentNode.Type != usecase.FileTypeDirectory {
		return fmt.Errorf("destination parent path '%s' is not a directory", dstParentPath)
	}
	dstParentID := dstParentNode.ID // Use the int ID

	// 移動先に同名のノードが存在するかチェック
	existingNode, err := r.GetNode(dstPath)
	if err == nil {
		// 移動先に何か存在する
		if !overwrite {
			return ErrAlreadyExists
		}
		// overwrite=true の場合
		if existingNode.ID == srcRoomID {
			return errors.New("cannot move file to itself")
		}

		// 既存のものを削除
		if existingNode.Type == usecase.FileTypeFile {
			// 既存のものがファイルの場合
			if err := r.DeleteFile(dstPath, true); err != nil { // force=true
				return fmt.Errorf("failed to overwrite existing file '%s': %w", dstPath, err)
			}
		} else {
			// 既存のものがディレクトリの場合、ファイル移動では上書きできない
			return fmt.Errorf("cannot overwrite existing directory '%s' with file", dstPath)
		}
	} else if !errors.Is(err, ErrNotFound) {
		// GetNodeで予期せぬエラー
		return fmt.Errorf("failed to check existence of destination path '%s': %w", dstPath, err)
	}
	// ErrNotFoundなら存在しないのでOK

	// ルームの親IDと名前を更新 (IDはint型を使用)
	_, err = r.db.Exec("UPDATE rooms SET directory_id = ?, name = ? WHERE id = ?", dstParentID, dstName, srcRoomID)
	if err != nil {
		// ユニーク制約違反の可能性 (name, directory_id)
		return fmt.Errorf("failed to update room path for %d: %w", srcRoomID, err)
	}

	return nil
}

// ListFiles は usecase.Repository インターフェースに合わせて実装する。
func (r *Repository) ListFiles(path int) ([]string, error) { // path is directory ID (int)
	dirID := path

	// showAll=true, longFormat=false 相当でファイル名のみ取得
	query := "SELECT name FROM rooms WHERE directory_id = ? ORDER BY name ASC"
	args := []interface{}{dirID} // Use the provided int ID

	rows, err := r.db.Query(query, args...)
	if err != nil {
		// Query error is possible
		return nil, fmt.Errorf("failed to query rooms for directory %d: %w", dirID, err)
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
		return nil, fmt.Errorf("error iterating over rooms for directory %d: %w", dirID, err)
	}

	return fileNames, nil
}

func (r *Repository) ListFileMatches(pattern string, paths []string, recursive, ignoreCase bool) ([]usecase.GrepResult, error) {
	// TODO: 未実装。GetNodeを使ってパスからファイルIDを取得し、それを使って検索する必要がある。
	return nil, errors.New("ListFileMatches not implemented")
}

func (r *Repository) ListMessages(roomID int) ([]string, error) {
	// Use the provided roomID int directly
	query := "SELECT content FROM messages WHERE room_id = ? ORDER BY created_at ASC"
	rows, err := r.db.Query(query, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages for room %d: %w", roomID, err)
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
		// Use %d for roomID in error message
		return nil, fmt.Errorf("error iterating over messages for room %d: %w", roomID, err)
	}

	return messages, nil
}

func (r *Repository) ListMessagesByQuery(roomID int, query string) ([]string, error) {
	// Note: LIKE '%?%' is incorrect for prepared statements. Use MATCH or concatenate.
	sqlQuery := `
		SELECT content
		FROM messages
		WHERE room_id = ? AND content LIKE '%?%'
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(sqlQuery, roomID, query)
	if err != nil {
		// Consider fallback to LIKE if MATCH fails or index doesn't exist
		// sqlQueryLike := "SELECT content FROM messages WHERE room_id = ? AND content LIKE ? ORDER BY created_at ASC"
		// rows, err = r.db.Query(sqlQueryLike, roomID, "%"+query+"%")
		// if err != nil {
		return nil, fmt.Errorf("failed to execute search in room %d for query '%s': %w", roomID, query, err)
		// }
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
		// Use %d for roomID in error message
		return nil, fmt.Errorf("error iterating over search results for room %d: %w", roomID, err)
	}

	return messages, nil
}
