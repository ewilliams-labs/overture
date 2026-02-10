package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
	_ "github.com/mattn/go-sqlite3" // Import the driver anonymously
)

// Adapter implements the repository port for SQLite
type Adapter struct {
	db *sql.DB
}

// NewAdapter creates a connection and runs the schema migration
func NewAdapter(storagePath string) (*Adapter, error) {
	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite db: %w", err)
	}

	adapter := &Adapter{db: db}

	// "Principal" Move: Auto-migrate on startup for local dev
	if err := adapter.migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return adapter, nil
}

// Close ensures the DB connection is closed gracefully
func (a *Adapter) Close() error {
	return a.db.Close()
}

func (a *Adapter) GetByID(ictx context.Context, id string) (domain.Playlist, error) {
	return domain.Playlist{}, nil
}

func (a *Adapter) Save(ctx context.Context, p domain.Playlist) error {
	// 1. Start Transaction
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Safety net: auto-rollback if we error/panic before commit

	// 2. Upsert Playlist (Create if new, Update name if exists)
	queryPlaylist := `
		INSERT INTO playlists (id, name) VALUES (?, ?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name;
	`
	if _, err := tx.ExecContext(ctx, queryPlaylist, p.ID, p.Name); err != nil {
		return fmt.Errorf("failed to save playlist metadata: %w", err)
	}

	// 3. Reset Links: Remove old track associations for this playlist
	// (We don't delete the tracks themselves, just the connection to this playlist)
	if _, err := tx.ExecContext(ctx, "DELETE FROM playlist_tracks WHERE playlist_id = ?", p.ID); err != nil {
		return fmt.Errorf("failed to clear old tracks: %w", err)
	}

	// 4. Upsert Tracks & Re-link
	// Prepare statements once for performance
	stmtTrack, err := tx.PrepareContext(ctx, `
		INSERT INTO tracks (id, title, artist, album, duration_ms, isrc) 
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING;
	`)
	if err != nil {
		return err
	}
	defer stmtTrack.Close()

	stmtLink, err := tx.PrepareContext(ctx, `INSERT INTO playlist_tracks (playlist_id, track_id) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer stmtLink.Close()

	for _, t := range p.Tracks {
		// Ensure track exists in the global 'tracks' table
		if _, err := stmtTrack.ExecContext(ctx, t.ID, t.Title, t.Artist, t.Album, t.DurationMs, t.ISRC, t.CoverURL); err != nil {
			return fmt.Errorf("failed to save track %s: %w", t.ID, err)
		}
		// Create the link in 'playlist_tracks'
		if _, err := stmtLink.ExecContext(ctx, p.ID, t.ID); err != nil {
			return fmt.Errorf("failed to link track %s: %w", t.ID, err)
		}
	}

	// 5. Commit Transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("transaction commit failed: %w", err)
	}

	return nil
}

func (a *Adapter) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS tracks (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		artist TEXT NOT NULL,
		album TEXT,
		duration_ms INTEGER,
		isrc TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS playlists (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS playlist_tracks (
		playlist_id TEXT,
		track_id TEXT,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (playlist_id, track_id),
		FOREIGN KEY(playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
		FOREIGN KEY(track_id) REFERENCES tracks(id) ON DELETE CASCADE
	);
	`
	_, err := a.db.Exec(query)
	return err
}
