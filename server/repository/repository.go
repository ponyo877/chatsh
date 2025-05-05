package repository

import (
	"database/sql"
	"path/filepath"

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

func (r *Repository) _GetNodeByPath(path domain.Path) (domain.Node, error) {
	query := `
		SELECT id, 1 AS type, owner_id, created_at FROM directories WHERE parent_id = $1 AND name = $2
		UNION ALL
		SELECT id, 2 AS type, owner_id, created_at FROM rooms WHERE directory_id = $1 AND name = $2
	`
	var nodeType domain.NodeType
	var nodeID, nextNodeID, ownerID int
	var createdAt time.Time
	for _, name := range path.Components {
		if err := r.db.QueryRow(query, nodeID, name).Scan(&nextNodeID, &nodeType, &ownerID, &createdAt); err != nil {
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
		OwnerID:   ownerID,
		CreatedAt: createdAt,
	}, nil
}

func (r *Repository) GetNodeByPath(path domain.Path) (domain.Node, error) {
	query := `
		SELECT id, 1 AS type, owner_id, created_at FROM directories WHERE path = $1
		UNION ALL
		SELECT id, 2 AS type, owner_id, created_at FROM rooms WHERE path = $1
	`

	var nodeType domain.NodeType
	var nodeID, ownerID int
	var createdAt time.Time
	if err := r.db.QueryRow(query, path.String()).Scan(&nodeID, &nodeType, &ownerID, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Node{}, ErrNotFound
		}
		return domain.Node{}, fmt.Errorf("error querying node: %w", err)
	}
	return domain.Node{
		ID:        nodeID,
		Name:      path.NodeName(),
		Type:      nodeType,
		OwnerID:   ownerID,
		CreatedAt: createdAt,
	}, nil
}

// showAll, longFormat はusecaseで判断
func (r *Repository) ListNodes(parentDirID int) ([]domain.Node, error) {
	query := `
		SELECT id, name, 1 AS type, owner_id, created_at FROM directories WHERE parent_id = $1
		UNION ALL
		SELECT id, name, 2 AS type, owner_id, created_at FROM rooms WHERE directory_id = $1
	`
	rows, err := r.db.Query(query, parentDirID)
	if err != nil {
		return nil, fmt.Errorf("failed to query subdirectories for %d: %w", parentDirID, err)
	}
	defer rows.Close()

	var results []domain.Node
	for rows.Next() {
		var id, ownerID int
		var name string
		var nodeType domain.NodeType
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &nodeType, &ownerID, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan subdirectory info: %w", err)
		}
		results = append(results, domain.Node{
			ID:        id,
			Name:      name,
			Type:      domain.NodeTypeDirectory,
			OwnerID:   ownerID,
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

// For Empty Directory
func (r *Repository) DeleteDirectory(dirID int) error {
	query := "DELETE FROM directories WHERE id = ?"
	if _, err := r.db.Exec(query, dirID); err != nil {
		return fmt.Errorf("failed to delete directory %s: %w", dirID, err)
	}
	return nil
}

// For Empty Directory
func (r *Repository) UpdateDirectory(srcDirID, dstDirID int, name string) error {
	query := "SELECT path FROM directories WHERE id = ?"
	var dstDirPath string
	if err := r.db.QueryRow(query, dstDirID).Scan(&dstDirPath); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("directory %d not found: %w", srcDirID, ErrNotFound)
		}
		return fmt.Errorf("failed to query directory path: %w", err)
	}

	query = "UPDATE directories SET parent_id = ?, name = ?, path = ? WHERE id = ?"
	newPath := filepath.Join(dstDirPath, name)
	if _, err := r.db.Exec(query, dstDirID, name, newPath, srcDirID); err != nil {
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

func (r *Repository) CreateExistRoom(roomID, dstDirID int, name string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	query := "INSERT INTO rooms (name, directory_id, created_at) VALUES (?, ?, ?)"
	if _, err := tx.Exec(query, name, dstDirID, time.Now()); err != nil {
		return fmt.Errorf("failed to insert room '%s': %w", name, err)
	}
	query = "INSERT INTO messages (room_id, user_id, content, created_at) SELECT room_id, user_id, content, created_at FROM messages WHERE room_id = ?"
	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to insert room '%s': %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (r *Repository) DeleteRoom(roomID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	query := "DELETE FROM rooms WHERE id = ?"
	_, err = tx.Exec(query, roomID)
	if err != nil {
		return fmt.Errorf("failed to delete room %d: %w", roomID, err)
	}
	query = "DELETE FROM messages WHERE room_id = ?"
	_, err = tx.Exec(query, roomID)
	if err != nil {
		return fmt.Errorf("failed to delete messages for room %d: %w", roomID, err)
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

func (r *Repository) CreateMessage(roomID, userID int, message string) error {
	query := "INSERT INTO messages (room_id, user_id, content, created_at) VALUES (?, ?, ?, ?)"
	if _, err := r.db.Exec(query, roomID, userID, message, time.Now()); err != nil {
		return fmt.Errorf("failed to insert message for room %d: %w", roomID, err)
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
