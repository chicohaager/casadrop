package storage

import (
	"database/sql"
	"fmt"
	"log"

	"casadrop/internal/utils"
)

// migrateMimeTypes corrects mime_type values written by versions that derived
// the type from http.DetectContentType alone.
//
// Two classes of row are wrong:
//
//   - The sniffer recognised nothing and stored application/octet-stream (or the
//     too-coarse application/ogg). getMediaType classifies by MIME prefix, so
//     those shares rendered no <audio>/<video> element at all — a .flac or an
//     untagged .mp3 could only be downloaded, never previewed.
//   - Matroska stored as video/webm, because .mkv and .webm share the same EBML
//     magic bytes.
//
// The stored value *is* the old sniff result, so re-running the same
// disambiguation over it produces exactly what a fresh upload would produce
// today — no need to re-read the uploaded files. Rows whose type was recognised
// (text/html, image/png, …) are non-committal-free and left untouched, which is
// what keeps this from becoming a content-type-confusion vector.
func (s *SQLiteStorage) migrateMimeTypes() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	updated := 0
	for _, t := range []struct{ table, nameCol string }{
		{"shares", "original_name"},
		{"received_files", "original_name"},
		{"folder_contents", "file_name"},
	} {
		n, err := refineMimeColumn(tx, t.table, t.nameCol)
		if err != nil {
			return err
		}
		updated += n
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	if updated > 0 {
		log.Printf("Corrected mime_type on %d row(s) written by an older version", updated)
	}
	return nil
}

// refineMimeColumn rewrites the mime_type of every row whose stored value the
// sniffer could not commit to, using the file name to disambiguate.
func refineMimeColumn(tx *sql.Tx, table, nameCol string) (int, error) {
	// Only the non-committal/ambiguous values are candidates — the set is fixed
	// and small, so the table name and column are the only interpolated parts
	// and both come from a hard-coded list above, never from input.
	//
	// Candidates are exactly the sniff results RefineMimeType can still change:
	// the two non-committal answers, plus the two committal ones the sniffer
	// provably cannot tell apart from something else (video/webm ↔ Matroska/WebM
	// audio, video/mp4 ↔ M4A/M4B). Everything else is left alone, which is what
	// keeps the migration from becoming a content-type-confusion vector.
	rows, err := tx.Query(
		`SELECT rowid, ` + nameCol + `, mime_type FROM ` + table + `
		 WHERE mime_type IN ('application/octet-stream', 'application/ogg',
		                     'video/webm', 'video/mp4')`,
	)
	if err != nil {
		// Report it. This used to return (0, nil) on the theory that the table
		// might not exist yet on a fresh database — which cannot happen:
		// initBaseSchema creates shares, received_files and folder_contents and
		// is fatal on failure (sqlite.go), and it runs before this migration.
		// So the only errors this could ever swallow were real ones, and it
		// swallowed them perfectly: with updated == 0 the summary line is
		// skipped too, leaving a failed migration indistinguishable from a
		// clean no-op in the startup log.
		return 0, fmt.Errorf("query %s: %w", table, err)
	}

	type fix struct {
		rowid int64
		mime  string
	}
	var fixes []fix
	for rows.Next() {
		var rowid int64
		var name, mime sql.NullString
		if err := rows.Scan(&rowid, &name, &mime); err != nil {
			rows.Close()
			return 0, err
		}
		if !name.Valid || !mime.Valid {
			continue
		}
		if refined := utils.RefineMimeType(mime.String, name.String); refined != mime.String {
			fixes = append(fixes, fix{rowid, refined})
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	for _, f := range fixes {
		if _, err := tx.Exec(
			`UPDATE `+table+` SET mime_type = ? WHERE rowid = ?`, f.mime, f.rowid,
		); err != nil {
			return 0, err
		}
	}
	return len(fixes), nil
}
