package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ponyo877/chatsh/server/domain"
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

func (r *Repository) GetNodeByPath(path domain.Path) (domain.Node, error) {
	query := `
		SELECT id, 1 AS type, created_at FROM directories WHERE parent_id = $1 AND name = $2
		UNION ALL
		SELECT id, 2 AS type, created_at FROM rooms WHERE directory_id = $1 AND name = $2
	`
	var nodeType domain.NodeType
	var nodeID, nextNodeID int
	var createdAt time.Time
	for _, name := range path {
		if err := r.db.QueryRow(query, nodeID, name).Scan(&nodeType, &nextNodeID, &createdAt); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.Node{}, ErrNotFound
			}
			return domain.Node{}, fmt.Errorf("error querying node: %w", err)
		}
		nodeID = nextNodeID
	}
	return domain.Node{
		ID:        nodeID,
		Name:      path.NodeName(),
		Type:      nodeType,
		CreatedAt: createdAt,
	}, nil
}

// showAll, longFormat はusecaseで判断
func (r *Repository) ListNodes(parentDirID int) ([]domain.Node, error) {
	query := `
		SELECT id, name, 1 AS type, created_at FROM directories WHERE parent_id = $1
		UNION ALL
		SELECT id, name, 2 AS type, created_at FROM rooms WHERE directory_id = $1
	`
	rows, err := r.db.Query(query, parentDirID)
	if err != nil {
		return nil, fmt.Errorf("failed to query subdirectories for %d: %w", parentDirID, err)
	}
	defer rows.Close()

	var results []domain.Node
	for rows.Next() {
		var id int
		var name string
		var nodeType domain.NodeType
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &nodeType, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan subdirectory info: %w", err)
		}
		results = append(results, domain.Node{
			ID:        id,
			Name:      name,
			Type:      domain.NodeTypeDirectory,
			CreatedAt: createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over subdirectories for %d: %w", parentDirID, err)
	}
	return results, nil
}

func (r *Repository) CreateDirectory(parentDirID int, name string, ownerID int) error {
	query := "INSERT INTO directories (name, parent_id, owner_id, created_at) VALUES (?, ?, ?, ?)"
	if _, err := r.db.Exec(query, name, parentDirID, ownerID, time.Now()); err != nil {
		return fmt.Errorf("failed to insert directory '%s': %w", name, err)
	}
	return nil
}

func (r *Repository) DeleteDirectory(dirID int) error {
	var count int
	query := `
		SELECT 
  			(SELECT COUNT(*) FROM directories WHERE parent_id = $1) + 
  			(SELECT COUNT(*) FROM rooms WHERE directory_id = $1);
	`
	if err := r.db.QueryRow(query, dirID).Scan(&count); err != nil {
		return fmt.Errorf("failed to check subdirectories for %s: %w", dirID, err)
	}
	if count > 0 {
		return errors.New("directory not empty")
	}
	if _, err := r.db.Exec("DELETE FROM directories WHERE id = ?", dirID); err != nil {
		return fmt.Errorf("failed to delete directory %s: %w", dirID, err)
	}
	return nil
}

func (r *Repository) UpdateDirectory(srcDirID, dstDirID int, name string) error {
	query := "UPDATE directories SET parent_id = ?, name = ? WHERE id = ?"
	if _, err := r.db.Exec(query, dstDirID, name, srcDirID); err != nil {
		return fmt.Errorf("failed to update directory path: %w", err)
	}
	return nil
}

func (r *Repository) CreateRoom(parentDirID int, name string) error {
	query := "INSERT INTO rooms (name, directory_id, created_at) VALUES (?, ?, ?)"
	if _, err := r.db.Exec(query, name, parentDirID, time.Now()); err != nil {
		return fmt.Errorf("failed to insert room '%s': %w", name, err)
	}
	return nil
}

func (r *Repository) DeleteRoom(roomID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	_, err = tx.Exec("DELETE FROM messages WHERE room_id = ?", roomID)
	if err != nil {
		return fmt.Errorf("failed to delete messages for room %d: %w", roomID, err)
	}
	_, err = tx.Exec("DELETE FROM rooms WHERE id = ?", roomID)
	if err != nil {
		return fmt.Errorf("failed to delete room %d: %w", roomID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (r *Repository) UpdateRoom(srcRoomID, dstDirID int, name string) error {
	query := "UPDATE rooms SET directory_id = ?, name = ? WHERE id = ?"
	if _, err := r.db.Exec(query, dstDirID, name, srcRoomID); err != nil {
		return fmt.Errorf("failed to update room path for %d: %w", srcRoomID, err)
	}
	return nil
}

func (r *Repository) ListMessages(roomID, limit, offset int) ([]domain.Message, error) {
	query := "SELECT id, user_id, content, created_at FROM messages WHERE room_id = ? ORDER BY created_at LIMIT ? OFFSET ?"
	rows, err := r.db.Query(query, roomID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages for room %d: %w", roomID, err)
	}
	defer rows.Close()

	var id, userID int
	var content string
	var createdAt time.Time
	messages := []domain.Message{}
	for rows.Next() {
		if err := rows.Scan(&id, &userID, &content, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan message content: %w", err)
		}
		messages = append(messages, domain.NewMessage(id, roomID, userID, content, createdAt))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over messages for room %d: %w", roomID, err)
	}

	return messages, nil
}

func (r *Repository) ListMessagesByQuery(roomID int, pattern string) ([]domain.Message, error) {
	query := "SELECT * FROM messages WHERE room_id = ? AND content REGEXP ? ORDER BY created_at"
	rows, err := r.db.Query(query, roomID, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search in room %d for query '%s': %w", roomID, pattern, err)
	}
	defer rows.Close()

	var messages []domain.Message
	var id, userID int
	var content string
	var createdAt time.Time
	for rows.Next() {
		if err := rows.Scan(&content, &id, &userID, &content, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan message content: %w", err)
		}
		messages = append(messages, domain.NewMessage(id, roomID, userID, content, createdAt))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over search results for room %d: %w", roomID, err)
	}
	return messages, nil
}
