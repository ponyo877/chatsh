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

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) usecase.Repository {
	return &Repository{db: db}
}

func (r *Repository) CheckDirectoryExists(path domain.Path) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM directories WHERE path = ?)"
	var exists bool
	err := r.db.QueryRow(query, path.String()).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error checking directory existence: %w", err)
	}
	return exists, nil
}

func (r *Repository) GetConfig(ownerToken string) (domain.Config, error) {
	query := "SELECT display_name FROM users WHERE token = ?"
	var config domain.Config
	config.OwnerToken = ownerToken
	if err := r.db.QueryRow(query, config.OwnerToken).Scan(&config.DisplayName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Config{}, usecase.ErrNotFound
		}
		return domain.Config{}, fmt.Errorf("error querying config: %w", err)
	}
	return config, nil
}

func (r *Repository) CreateConfig(config domain.Config) error {
	query := "INSERT INTO users (token, display_name, created_at) VALUES (?, ?, ?)"
	if _, err := r.db.Exec(query, config.OwnerToken, config.DisplayName, time.Now()); err != nil {
		return fmt.Errorf("error inserting config: %w", err)
	}
	return nil
}

func (r *Repository) GetNodeByPath(path domain.Path) (domain.Node, error) {
	query := `
		SELECT
			d.id,
			1 AS type,
			d.owner_token,
			u.display_name,
			d.created_at
		FROM directories d
		JOIN users u ON d.owner_token = u.token
		WHERE d.path = $1
		UNION ALL
		SELECT
			r.id,
			2 AS type,
			r.owner_token,
			u.display_name,
			r.created_at
		FROM rooms r
		JOIN users u ON r.owner_token = u.token
		WHERE r.path = $1;
	`

	var nodeType domain.NodeType
	var nodeID int
	var ownerToken, displayName string
	var createdAt time.Time
	if err := r.db.QueryRow(query, path.String()).Scan(&nodeID, &nodeType, &ownerToken, &displayName, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Node{}, usecase.ErrNotFound
		}
		return domain.Node{}, fmt.Errorf("error querying node: %w", err)
	}
	return domain.Node{
		ID:         nodeID,
		Name:       path.NodeName(),
		Type:       nodeType,
		OwnerToken: ownerToken,
		OwnerName:  displayName,
		CreatedAt:  createdAt,
	}, nil
}

// showAll, longFormat はusecaseで判断
func (r *Repository) ListNodes(parentDirID int) ([]domain.Node, error) {
	query := `
		SELECT
			d.id,
			d.name,
			1 AS type,
			d.owner_token,
			u.display_name,
			d.created_at
		FROM directories d
		JOIN users u ON d.owner_token = u.token
		WHERE parent_id = $1
		UNION ALL
		SELECT
			r.id,
			r.name,
			2 AS type,
			r.owner_token,
			u.display_name,
			r.created_at
		FROM rooms r
		JOIN users u ON r.owner_token = u.token
		WHERE directory_id = $1
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
		var ownerToken, displayName string
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &nodeType, &ownerToken, &displayName, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan subdirectory info: %w", err)
		}
		results = append(results, domain.NewNode(
			id,
			name,
			domain.NodeType(nodeType),
			ownerToken,
			displayName,
			createdAt,
		))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over subdirectories for %d: %w", parentDirID, err)
	}
	return results, nil
}

func (r *Repository) CreateDirectory(parentDirID int, parentPath string, name string, ownerToken string) error {
	newPath := filepath.Join(parentPath, name)
	query := "INSERT INTO directories (name, parent_id, owner_token, path, created_at) VALUES (?, ?, ?, ?, ?)"
	if _, err := r.db.Exec(query, name, parentDirID, ownerToken, newPath, time.Now()); err != nil {
		return fmt.Errorf("failed to insert directory '%s': %w", name, err)
	}
	return nil
}

// For Empty Directory
func (r *Repository) DeleteDirectory(dirID int) error {
	query := "DELETE FROM directories WHERE id = ?"
	if _, err := r.db.Exec(query, dirID); err != nil {
		return fmt.Errorf("failed to delete directory %d: %w", dirID, err)
	}
	return nil
}

// For Empty Directory
func (r *Repository) UpdateDirectory(srcDirID, dstDirID int, dstDirPath string, name string) error {
	query := "UPDATE directories SET parent_id = ?, name = ?, path = ? WHERE id = ?"
	newPath := filepath.Join(dstDirPath, name)
	if _, err := r.db.Exec(query, dstDirID, name, newPath, srcDirID); err != nil {
		return fmt.Errorf("failed to update directory path: %w", err)
	}
	return nil
}

func (r *Repository) CreateRoom(parentDirID int, parentDirPath, name, ownerToken string) error {
	newPath := filepath.Join(parentDirPath, name)
	query := "INSERT INTO rooms (name, directory_id, path, owner_token, created_at) VALUES (?, ?, ?, ?, ?)"
	if _, err := r.db.Exec(query, name, parentDirID, newPath, ownerToken, time.Now()); err != nil {
		return fmt.Errorf("failed to insert room '%s': %w", name, err)
	}
	return nil
}

func (r *Repository) CreateExistRoom(roomID, dstDirID int, dstDirPath, name, ownerToken string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	newPath := filepath.Join(dstDirPath, name)
	query := "INSERT INTO rooms (name, directory_id, path, owner_token, created_at) VALUES (?, ?, ?, ?, ?)"
	result, err := tx.Exec(query, name, dstDirID, newPath, ownerToken, time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert room '%s': %w", name, err)
	}
	newRoomID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	query = "INSERT INTO messages (room_id, content, display_name, created_at) SELECT ?, content, display_name, created_at FROM messages WHERE room_id = ?"
	if _, err := tx.Exec(query, newRoomID, roomID); err != nil {
		return fmt.Errorf("failed to insert messages '%s': %w", name, err)
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
	query := "DELETE FROM messages WHERE room_id = ?"
	_, err = tx.Exec(query, roomID)
	if err != nil {
		return fmt.Errorf("failed to delete messages for room %d: %w", roomID, err)
	}
	query = "DELETE FROM rooms WHERE id = ?"
	_, err = tx.Exec(query, roomID)
	if err != nil {
		return fmt.Errorf("failed to delete room %d: %w", roomID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (r *Repository) UpdateRoom(srcRoomID, dstDirID int, dstDirPath, name string) error {
	newPath := filepath.Join(dstDirPath, name)
	query := "UPDATE rooms SET directory_id = ?, name = ?, path = ? WHERE id = ?"
	if _, err := r.db.Exec(query, dstDirID, name, newPath, srcRoomID); err != nil {
		return fmt.Errorf("failed to update room path for %d: %w", srcRoomID, err)
	}
	return nil
}

func (r *Repository) CreateMessage(roomID int, displayName, message string) error {
	query := "INSERT INTO messages (room_id, display_name, content, created_at) VALUES (?, ?, ?, ?)"
	if _, err := r.db.Exec(query, roomID, displayName, message, time.Now()); err != nil {
		return fmt.Errorf("failed to insert message for room %d: %w", roomID, err)
	}
	return nil
}

func (r *Repository) ListMessages(roomID, limit, offset int) ([]domain.Message, error) {
	query := "SELECT id, display_name, content, created_at FROM messages WHERE room_id = ? ORDER BY created_at LIMIT ? OFFSET ?"
	rows, err := r.db.Query(query, roomID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages for room %d: %w", roomID, err)
	}
	defer rows.Close()

	var id int
	var content, displayName string
	var createdAt time.Time
	messages := []domain.Message{}
	for rows.Next() {
		if err := rows.Scan(&id, &displayName, &content, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan message content: %w", err)
		}
		messages = append(messages, domain.NewMessage(id, roomID, displayName, content, createdAt))
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
	var id int
	var content, displayName string
	var createdAt time.Time
	for rows.Next() {
		if err := rows.Scan(&content, &id, &displayName, &content, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan message content: %w", err)
		}
		messages = append(messages, domain.NewMessage(id, roomID, displayName, content, createdAt))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over search results for room %d: %w", roomID, err)
	}
	return messages, nil
}
