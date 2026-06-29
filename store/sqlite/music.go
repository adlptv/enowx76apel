package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/enowdev/enowx/store"
)

type musicStore struct{ db *sql.DB }

func (s *musicStore) ListPlaylists(ctx context.Context) ([]store.Playlist, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT p.id, p.name, p.description, p.share_code, p.created_at,
		        (SELECT COUNT(*) FROM playlist_tracks t WHERE t.playlist_id = p.id) AS cnt
		 FROM playlists p WHERE p.deleted = 0 ORDER BY p.created_at DESC, p.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Playlist
	for rows.Next() {
		var p store.Playlist
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.ShareCode, &p.CreatedAt, &p.Count); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *musicStore) GetPlaylist(ctx context.Context, id int64) (*store.Playlist, error) {
	var p store.Playlist
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, share_code, created_at FROM playlists WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.Description, &p.ShareCode, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	tracks, err := s.tracks(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	p.Tracks = tracks
	p.Count = len(tracks)
	return &p, nil
}

func (s *musicStore) PlaylistByShareCode(ctx context.Context, code string) (*store.Playlist, error) {
	var p store.Playlist
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, share_code, created_at FROM playlists WHERE share_code = ?`, code).
		Scan(&p.ID, &p.Name, &p.Description, &p.ShareCode, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	tracks, err := s.tracks(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	p.Tracks = tracks
	p.Count = len(tracks)
	return &p, nil
}

func (s *musicStore) tracks(ctx context.Context, playlistID int64) ([]store.MusicTrack, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT video_id, title, artist, album, duration, thumbnail
		 FROM playlist_tracks WHERE playlist_id = ? ORDER BY position ASC, id ASC`, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []store.MusicTrack{}
	for rows.Next() {
		var t store.MusicTrack
		if err := rows.Scan(&t.VideoID, &t.Title, &t.Artist, &t.Album, &t.Duration, &t.Thumbnail); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// touch bumps a playlist's sync metadata (unix-millis updated_at + version) so
// the change is picked up by two-way sync as a last-write-wins update.
func (s *musicStore) touch(ctx context.Context, playlistID int64) {
	_, _ = s.db.ExecContext(ctx,
		`UPDATE playlists SET sync_updated_at = ?, sync_version = sync_version + 1 WHERE id = ?`,
		time.Now().UnixMilli(), playlistID)
}

func (s *musicStore) CreatePlaylist(ctx context.Context, name, description, shareCode string) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO playlists (name, description, share_code, sync_updated_at, sync_version) VALUES (?, ?, ?, ?, 1)`,
		name, description, shareCode, time.Now().UnixMilli())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// DeletePlaylist soft-deletes: the row becomes a tombstone (deleted=1) and its
// tracks are removed, so the deletion propagates to other devices via sync
// instead of being resurrected on the next pull.
func (s *musicStore) DeletePlaylist(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM playlist_tracks WHERE playlist_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE playlists SET deleted = 1, sync_updated_at = ?, sync_version = sync_version + 1 WHERE id = ?`,
		time.Now().UnixMilli(), id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *musicStore) AddTrack(ctx context.Context, playlistID int64, t store.MusicTrack) error {
	// Append to the end; ignore if the track is already in the playlist.
	var pos int
	_ = s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(position)+1, 0) FROM playlist_tracks WHERE playlist_id = ?`, playlistID).Scan(&pos)
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO playlist_tracks
		   (playlist_id, video_id, title, artist, album, duration, thumbnail, position)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		playlistID, t.VideoID, t.Title, t.Artist, t.Album, t.Duration, t.Thumbnail, pos)
	if err == nil {
		s.touch(ctx, playlistID)
	}
	return err
}

func (s *musicStore) RemoveTrack(ctx context.Context, playlistID int64, videoID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM playlist_tracks WHERE playlist_id = ? AND video_id = ?`, playlistID, videoID)
	if err == nil {
		s.touch(ctx, playlistID)
	}
	return err
}

func (s *musicStore) RecordPlay(ctx context.Context, e store.PlayEvent) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO play_history (video_id, title, artist, album) VALUES (?, ?, ?, ?)`,
		e.VideoID, e.Title, e.Artist, e.Album)
	return err
}

func (s *musicStore) RecentPlays(ctx context.Context, limit int) ([]store.MusicTrack, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	// Most recent distinct tracks (latest play wins).
	rows, err := s.db.QueryContext(ctx,
		`SELECT video_id, title, artist, album FROM play_history
		 WHERE id IN (SELECT MAX(id) FROM play_history GROUP BY video_id)
		 ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []store.MusicTrack{}
	for rows.Next() {
		var t store.MusicTrack
		if err := rows.Scan(&t.VideoID, &t.Title, &t.Artist, &t.Album); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *musicStore) TopArtists(ctx context.Context, limit int) ([]store.ArtistCount, error) {
	if limit <= 0 || limit > 50 {
		limit = 8
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT artist, COUNT(*) AS plays FROM play_history
		 WHERE artist <> '' GROUP BY artist ORDER BY plays DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []store.ArtistCount{}
	for rows.Next() {
		var a store.ArtistCount
		if err := rows.Scan(&a.Artist, &a.Plays); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *musicStore) ClearHistory(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM play_history`)
	return err
}

// PlaylistsForSync returns every playlist (including tombstones) as a sync item.
func (s *musicStore) PlaylistsForSync(ctx context.Context) ([]store.SyncedPlaylist, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, description, share_code, sync_updated_at, sync_version, deleted FROM playlists`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type row struct {
		id int64
		sp store.SyncedPlaylist
	}
	var rowsOut []row
	for rows.Next() {
		var r row
		var del int
		if err := rows.Scan(&r.id, &r.sp.Name, &r.sp.Description, &r.sp.ShareCode, &r.sp.UpdatedAt, &r.sp.Version, &del); err != nil {
			return nil, err
		}
		r.sp.Deleted = del != 0
		rowsOut = append(rowsOut, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]store.SyncedPlaylist, 0, len(rowsOut))
	for _, r := range rowsOut {
		if !r.sp.Deleted {
			tracks, err := s.tracks(ctx, r.id)
			if err != nil {
				return nil, err
			}
			r.sp.Tracks = tracks
		}
		out = append(out, r.sp)
	}
	return out, nil
}

// ApplySyncedPlaylist upserts a remote playlist by share code. The syncer has
// already decided this version wins (LWW), so we write its metadata verbatim.
func (s *musicStore) ApplySyncedPlaylist(ctx context.Context, p store.SyncedPlaylist) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var id int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM playlists WHERE share_code = ?`, p.ShareCode).Scan(&id)
	switch {
	case err == sql.ErrNoRows:
		res, e := tx.ExecContext(ctx,
			`INSERT INTO playlists (name, description, share_code, sync_updated_at, sync_version, deleted)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			p.Name, p.Description, p.ShareCode, p.UpdatedAt, p.Version, boolToInt(p.Deleted))
		if e != nil {
			return e
		}
		id, _ = res.LastInsertId()
	case err != nil:
		return err
	default:
		if _, e := tx.ExecContext(ctx,
			`UPDATE playlists SET name=?, description=?, sync_updated_at=?, sync_version=?, deleted=? WHERE id=?`,
			p.Name, p.Description, p.UpdatedAt, p.Version, boolToInt(p.Deleted), id); e != nil {
			return e
		}
	}

	// Replace the track set to match the remote exactly.
	if _, e := tx.ExecContext(ctx, `DELETE FROM playlist_tracks WHERE playlist_id = ?`, id); e != nil {
		return e
	}
	if !p.Deleted {
		for i, t := range p.Tracks {
			if _, e := tx.ExecContext(ctx,
				`INSERT INTO playlist_tracks (playlist_id, video_id, title, artist, album, duration, thumbnail, position)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				id, t.VideoID, t.Title, t.Artist, t.Album, t.Duration, t.Thumbnail, i); e != nil {
				return e
			}
		}
	}
	return tx.Commit()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
