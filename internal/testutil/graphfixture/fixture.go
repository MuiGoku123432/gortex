// Package graphfixture provides immutable SQLite graph fixtures for tests.
// Production packages must not import it.
package graphfixture

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

const (
	OwnerKey       = "workspace:fixture/repo"
	NeighborOwner  = "workspace:neighbor/repo"
	ScopedNodeID   = "fixture/repo/main.go::Run"
	ScopedTargetID = "fixture/repo/main.go::Store"
)

// Fixture describes a seeded SQLite graph and its expected scope.
type Fixture struct {
	Path          string
	Owner         string
	NeighborOwner string
}

// Fingerprint records the state of the database and its sidecars.
type Fingerprint struct {
	Files     map[string]FileFingerprint
	Canonical string
}

type FileFingerprint struct {
	Exists bool
	Size   int64
	SHA256 string
}

// Create builds a deterministic scoped graph database under dir.
func Create(dir string) (Fixture, error) {
	path := filepath.Join(dir, "projection-fixture.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return Fixture{}, err
	}
	defer db.Close()

	const schema = `
PRAGMA journal_mode=WAL;
PRAGMA wal_autocheckpoint=0;
CREATE TABLE nodes (owner_key TEXT NOT NULL, id TEXT NOT NULL, kind TEXT NOT NULL, name TEXT NOT NULL, file_path TEXT NOT NULL, meta TEXT NOT NULL, PRIMARY KEY(owner_key,id));
CREATE TABLE edges (owner_key TEXT NOT NULL, from_id TEXT NOT NULL, to_id TEXT NOT NULL, kind TEXT NOT NULL, file_path TEXT NOT NULL, line INTEGER NOT NULL, origin TEXT NOT NULL, meta TEXT NOT NULL, PRIMARY KEY(owner_key,from_id,to_id,kind,file_path,line,origin));
CREATE TABLE projection_manifest (owner_key TEXT PRIMARY KEY, operation_id TEXT NOT NULL, pending_generation TEXT NOT NULL, active_generation TEXT NOT NULL, cleanup_complete INTEGER NOT NULL);
`
	if _, err := db.Exec(schema); err != nil {
		return Fixture{}, fmt.Errorf("create graph fixture: %w", err)
	}
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO nodes VALUES(?,?,?,?,?,?)`, []any{OwnerKey, ScopedNodeID, "function", "Run", "fixture/repo/main.go", `{"secret_canary":"fixture-secret"}`}},
		{`INSERT INTO nodes VALUES(?,?,?,?,?,?)`, []any{OwnerKey, ScopedTargetID, "type", "Store", "fixture/repo/main.go", `{}`}},
		{`INSERT INTO edges VALUES(?,?,?,?,?,?,?,?)`, []any{OwnerKey, ScopedNodeID, ScopedTargetID, "references", "fixture/repo/main.go", 12, "ast_resolved", `{}`}},
		{`INSERT INTO nodes VALUES(?,?,?,?,?,?)`, []any{NeighborOwner, "neighbor/repo/main.go::Run", "function", "Run", "neighbor/repo/main.go", `{}`}},
		{`INSERT INTO projection_manifest VALUES(?,?,?,?,?)`, []any{OwnerKey, "operation-fixture", "generation-pending", "generation-active", 0}},
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement.query, statement.args...); err != nil {
			return Fixture{}, fmt.Errorf("seed graph fixture: %w", err)
		}
	}
	return Fixture{Path: path, Owner: OwnerKey, NeighborOwner: NeighborOwner}, nil
}

// Snapshot fingerprints the DB/WAL/SHM files and canonical graph rows.
func Snapshot(path string) (Fingerprint, error) {
	canonical, err := Canonical(path)
	if err != nil {
		return Fingerprint{}, err
	}
	f := Fingerprint{Files: make(map[string]FileFingerprint), Canonical: canonical}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		name := path + suffix
		data, err := os.ReadFile(name)
		if os.IsNotExist(err) {
			f.Files[suffix] = FileFingerprint{}
			continue
		}
		if err != nil {
			return Fingerprint{}, err
		}
		sum := sha256.Sum256(data)
		f.Files[suffix] = FileFingerprint{Exists: true, Size: int64(len(data)), SHA256: hex.EncodeToString(sum[:])}
	}
	return f, nil
}

// Canonical returns sorted logical graph content, independent of SQLite bytes.
func Canonical(path string) (string, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return "", err
	}
	defer db.Close()
	var ownerColumn int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('nodes') WHERE name = 'owner_key'`).Scan(&ownerColumn); err != nil {
		return "", err
	}
	queries := []string{
		`SELECT printf('node\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s',owner_key,id,kind,name,file_path,hex(meta)) FROM nodes`,
		`SELECT printf('edge\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%d\x1f%s\x1f%s',owner_key,from_id,to_id,kind,file_path,line,origin,hex(meta)) FROM edges`,
		`SELECT printf('manifest\x1f%s\x1f%s\x1f%s\x1f%s\x1f%d',owner_key,operation_id,pending_generation,active_generation,cleanup_complete) FROM projection_manifest`,
	}
	if ownerColumn == 0 {
		queries = []string{
			`SELECT printf('node\x1f%s\x1f%d\x1f%s\x1f%s\x1f%s\x1f%s\x1f%d\x1f%d\x1f%d\x1f%d\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s',id,view_gen,kind,name,qual_name,file_path,start_line,end_line,start_column,end_column,language,repo_prefix,workspace_id,project_id,hex(meta)) FROM nodes`,
			`SELECT printf('edge\x1f%d\x1f%s\x1f%s\x1f%s\x1f%s\x1f%d\x1f%g\x1f%s\x1f%s\x1f%s\x1f%d\x1f%d\x1f%s',id,from_id,to_id,kind,file_path,line,confidence,confidence_label,origin,tier,cross_repo,view_gen,hex(meta)) FROM edges`,
		}
	}
	var records []string
	for _, query := range queries {
		rows, err := db.Query(query)
		if err != nil {
			return "", err
		}
		for rows.Next() {
			var record string
			if err := rows.Scan(&record); err != nil {
				rows.Close()
				return "", err
			}
			records = append(records, record)
		}
		if err := rows.Close(); err != nil {
			return "", err
		}
	}
	sort.Strings(records)
	return strings.Join(records, "\n"), nil
}

// Equal reports whether physical sidecars and canonical content are unchanged.
func Equal(a, b Fingerprint) bool {
	if a.Canonical != b.Canonical || len(a.Files) != len(b.Files) {
		return false
	}
	for name, left := range a.Files {
		if right, ok := b.Files[name]; !ok || left != right {
			return false
		}
	}
	return true
}
