// Package sqlite provides a SQLite-backed implementation of the repository port.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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
	row := a.db.QueryRowContext(ictx, "SELECT id, name FROM playlists WHERE id = ?", id)
	var playlist domain.Playlist
	if err := row.Scan(&playlist.ID, &playlist.Name); err != nil {
		if err == sql.ErrNoRows {
			return domain.Playlist{}, domain.ErrNotFound
		}
		return domain.Playlist{}, fmt.Errorf("failed to load playlist: %w", err)
	}
	playlist.Tracks = []domain.Track{}

	trackRows, err := a.db.QueryContext(ictx, `
		SELECT t.id, t.title, t.artist, t.album, t.duration_ms, t.isrc, t.cover_url, t.preview_url,
			IFNULL(t.danceability, 0), IFNULL(t.energy, 0), IFNULL(t.valence, 0),
			IFNULL(t.tempo, 0), IFNULL(t.instrumentalness, 0), IFNULL(t.acousticness, 0)
		FROM tracks t
		JOIN playlist_tracks pt ON pt.track_id = t.id
		WHERE pt.playlist_id = ?
		ORDER BY pt.added_at ASC
	`, playlist.ID)
	if err != nil {
		return domain.Playlist{}, fmt.Errorf("failed to load playlist tracks: %w", err)
	}
	defer trackRows.Close()

	for trackRows.Next() {
		var track domain.Track
		var album sql.NullString
		var isrc sql.NullString
		var coverURL sql.NullString
		var previewURL sql.NullString
		var duration sql.NullInt64
		if err := trackRows.Scan(
			&track.ID,
			&track.Title,
			&track.Artist,
			&album,
			&duration,
			&isrc,
			&coverURL,
			&previewURL,
			&track.Features.Danceability,
			&track.Features.Energy,
			&track.Features.Valence,
			&track.Features.Tempo,
			&track.Features.Instrumentalness,
			&track.Features.Acousticness,
		); err != nil {
			return domain.Playlist{}, fmt.Errorf("failed to scan playlist track: %w", err)
		}
		if album.Valid {
			track.Album = album.String
		}
		if duration.Valid {
			track.DurationMs = int(duration.Int64)
		}
		if isrc.Valid {
			track.ISRC = isrc.String
		}
		if coverURL.Valid {
			track.CoverURL = coverURL.String
		}
		if previewURL.Valid {
			track.PreviewURL = previewURL.String
		}
		playlist.Tracks = append(playlist.Tracks, track)
	}
	if err := trackRows.Err(); err != nil {
		return domain.Playlist{}, fmt.Errorf("failed to iterate playlist tracks: %w", err)
	}

	return playlist, nil
}

func (a *Adapter) GetPlaylistAudioFeatures(ctx context.Context, playlistID string) (domain.AudioFeatures, error) {
	row := a.db.QueryRowContext(ctx, "SELECT id FROM playlists WHERE id = ?", playlistID)
	var id string
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return domain.AudioFeatures{}, domain.ErrNotFound
		}
		return domain.AudioFeatures{}, fmt.Errorf("failed to load playlist: %w", err)
	}

	query := `
		SELECT
			COALESCE(AVG(t.danceability), 0),
			COALESCE(AVG(t.energy), 0),
			COALESCE(AVG(t.valence), 0),
			COALESCE(AVG(t.tempo), 0),
			COALESCE(AVG(t.instrumentalness), 0),
			COALESCE(AVG(t.acousticness), 0)
		FROM tracks t
		JOIN playlist_tracks pt ON pt.track_id = t.id
		WHERE pt.playlist_id = ?
	`

	var features domain.AudioFeatures
	if err := a.db.QueryRowContext(ctx, query, playlistID).Scan(
		&features.Danceability,
		&features.Energy,
		&features.Valence,
		&features.Tempo,
		&features.Instrumentalness,
		&features.Acousticness,
	); err != nil {
		return domain.AudioFeatures{}, fmt.Errorf("failed to load playlist audio features: %w", err)
	}

	return features, nil
}

func (a *Adapter) UpdateTrackFeatures(ctx context.Context, trackID string, features domain.AudioFeatures) error {
	query := `
		UPDATE tracks
		SET
			danceability = ?,
			energy = ?,
			valence = ?,
			tempo = ?,
			instrumentalness = ?,
			acousticness = ?
		WHERE id = ?
	`
	if _, err := a.db.ExecContext(
		ctx,
		query,
		features.Danceability,
		features.Energy,
		features.Valence,
		features.Tempo,
		features.Instrumentalness,
		features.Acousticness,
		trackID,
	); err != nil {
		return fmt.Errorf("failed to update track features: %w", err)
	}

	return nil
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
		INSERT INTO tracks (
			id, title, artist, album, duration_ms, isrc, cover_url, preview_url,
			danceability, energy, valence, tempo, instrumentalness, acousticness
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title=excluded.title,
			artist=excluded.artist,
			album=excluded.album,
			duration_ms=excluded.duration_ms,
			isrc=excluded.isrc,
			cover_url=excluded.cover_url,
			preview_url=excluded.preview_url,
			danceability=excluded.danceability,
			energy=excluded.energy,
			valence=excluded.valence,
			tempo=excluded.tempo,
			instrumentalness=excluded.instrumentalness,
			acousticness=excluded.acousticness;
	`)
	if err != nil {
		return err
	}
	defer stmtTrack.Close()

	stmtLink, err := tx.PrepareContext(ctx, `
		INSERT INTO playlist_tracks (playlist_id, track_id)
		VALUES (?, ?)
		ON CONFLICT(playlist_id, track_id) DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmtLink.Close()

	for _, t := range p.Tracks {
		// Ensure track exists in the global 'tracks' table
		if _, err := stmtTrack.ExecContext(
			ctx,
			t.ID,
			t.Title,
			t.Artist,
			t.Album,
			t.DurationMs,
			t.ISRC,
			t.CoverURL,
			t.PreviewURL,
			t.Features.Danceability,
			t.Features.Energy,
			t.Features.Valence,
			t.Features.Tempo,
			t.Features.Instrumentalness,
			t.Features.Acousticness,
		); err != nil {
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
		cover_url TEXT,
		preview_url TEXT,
		danceability REAL,
		energy REAL,
		valence REAL,
		tempo REAL,
		instrumentalness REAL,
		acousticness REAL,
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
	if _, err := a.db.Exec(query); err != nil {
		return err
	}

	if _, err := a.db.Exec("ALTER TABLE tracks ADD COLUMN cover_url TEXT"); err != nil {
		if !isDuplicateColumnError(err) {
			return err
		}
	}
	if _, err := a.db.Exec("ALTER TABLE tracks ADD COLUMN preview_url TEXT"); err != nil {
		if !isDuplicateColumnError(err) {
			return err
		}
	}
	if _, err := a.db.Exec("ALTER TABLE tracks ADD COLUMN danceability REAL"); err != nil {
		if !isDuplicateColumnError(err) {
			return err
		}
	}
	if _, err := a.db.Exec("ALTER TABLE tracks ADD COLUMN energy REAL"); err != nil {
		if !isDuplicateColumnError(err) {
			return err
		}
	}
	if _, err := a.db.Exec("ALTER TABLE tracks ADD COLUMN valence REAL"); err != nil {
		if !isDuplicateColumnError(err) {
			return err
		}
	}
	if _, err := a.db.Exec("ALTER TABLE tracks ADD COLUMN tempo REAL"); err != nil {
		if !isDuplicateColumnError(err) {
			return err
		}
	}
	if _, err := a.db.Exec("ALTER TABLE tracks ADD COLUMN instrumentalness REAL"); err != nil {
		if !isDuplicateColumnError(err) {
			return err
		}
	}
	if _, err := a.db.Exec("ALTER TABLE tracks ADD COLUMN acousticness REAL"); err != nil {
		if !isDuplicateColumnError(err) {
			return err
		}
	}

	return nil
}

func isDuplicateColumnError(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "duplicate column") || strings.Contains(err.Error(), "already exists"))
}
