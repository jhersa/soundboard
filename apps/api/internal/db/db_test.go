package db

import (
	"context"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestMigrateEsIdempotenteYActivaForeignKeys(t *testing.T) {
	ctx := context.Background()
	database, err := Open(filepath.Join(t.TempDir(), "x.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	migrations := fstest.MapFS{
		"0001_a.sql": {Data: []byte("CREATE TABLE a (id TEXT PRIMARY KEY); CREATE TABLE b (a_id TEXT REFERENCES a(id) ON DELETE CASCADE);")},
		"0002_b.sql": {Data: []byte("INSERT INTO a VALUES ('x'); INSERT INTO b VALUES ('x');")},
	}
	applied, err := Migrate(ctx, database, migrations)
	if err != nil || len(applied) != 2 {
		t.Fatalf("primera vez: %v %v", applied, err)
	}
	if applied, err := Migrate(ctx, database, migrations); err != nil || len(applied) != 0 {
		t.Fatalf("segunda vez no debe aplicar nada: %v %v", applied, err)
	}
	database.Exec("DELETE FROM a")
	var count int
	database.QueryRow("SELECT COUNT(*) FROM b").Scan(&count)
	if count != 0 {
		t.Fatal("foreign_keys debe estar activo (ON DELETE CASCADE)")
	}
}
