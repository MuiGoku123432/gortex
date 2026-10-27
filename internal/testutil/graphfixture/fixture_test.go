package graphfixture

import (
	"database/sql"
	"strings"
	"testing"
)

func TestProjectionFixture(t *testing.T) {
	fixture, err := Create(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	before, err := Snapshot(fixture.Path)
	if err != nil {
		t.Fatal(err)
	}
	after, err := Snapshot(fixture.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !Equal(before, after) {
		t.Fatal("read-only snapshot changed fixture")
	}
	if !before.Files[""].Exists || !before.Files["-wal"].Exists || !before.Files["-shm"].Exists {
		t.Fatalf("expected DB/WAL/SHM fingerprints: %#v", before.Files)
	}
	if !strings.Contains(before.Canonical, fixture.Owner) || !strings.Contains(before.Canonical, fixture.NeighborOwner) {
		t.Fatalf("canonical graph does not distinguish owners: %s", before.Canonical)
	}

	db, err := sql.Open("sqlite", "file:"+fixture.Path+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var scoped, neighboring int
	if err := db.QueryRow(`SELECT count(*) FROM nodes WHERE owner_key=?`, fixture.Owner).Scan(&scoped); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM nodes WHERE owner_key=?`, fixture.NeighborOwner).Scan(&neighboring); err != nil {
		t.Fatal(err)
	}
	if scoped != 2 || neighboring != 1 {
		t.Fatalf("unexpected scope counts: scoped=%d neighboring=%d", scoped, neighboring)
	}
}
