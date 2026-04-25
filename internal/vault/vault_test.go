package vault

import (
	"path/filepath"
	"testing"

	"github.com/mms/sleutel/internal/model"
)

func tempVaultPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "vault.slkv")
}

var testPassword = []byte("correct-horse-battery-staple")

// fastParams overrides Argon2 params for fast tests.
// Tests call Create/Open via unexported helpers that accept a path directly.
// We patch the default params by using low-cost params in the test binary.
// Since Create uses crypto.DefaultParams(), we create the vault normally but
// accept that tests run slightly slower (~100ms for KDF) — acceptable since
// the tests are not benchmarks and the KDF is the security primitive.

func TestCreateOpen(t *testing.T) {
	path := tempVaultPath(t)

	v, err := Create(path, testPassword)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	v.Close()

	v2, err := Open(path, testPassword)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer v2.Close()

	if v2.Path() != path {
		t.Errorf("path mismatch: %q vs %q", v2.Path(), path)
	}
}

func TestCreateAlreadyExists(t *testing.T) {
	path := tempVaultPath(t)
	v, _ := Create(path, testPassword)
	v.Close()

	_, err := Create(path, testPassword)
	if err == nil {
		t.Fatal("expected error creating vault that already exists")
	}
}

func TestOpenWrongPassword(t *testing.T) {
	path := tempVaultPath(t)
	v, _ := Create(path, testPassword)
	v.Close()

	_, err := Open(path, []byte("wrongpassword"))
	if err == nil {
		t.Fatal("expected error with wrong password")
	}
}

func TestAddGetEntry(t *testing.T) {
	path := tempVaultPath(t)
	v, _ := Create(path, testPassword)
	defer v.Close()

	e, err := v.Add(model.Entry{Title: "GitHub", Username: "alice", Password: "s3cr3t"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if e.ID == "" {
		t.Error("entry ID should be set")
	}
	if e.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	got, err := v.Get(e.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "GitHub" {
		t.Errorf("title: %q", got.Title)
	}
}

func TestEditEntry(t *testing.T) {
	path := tempVaultPath(t)
	v, _ := Create(path, testPassword)
	defer v.Close()

	e, _ := v.Add(model.Entry{Title: "old title", Password: "pw"})

	updated, err := v.Edit(e.ID, model.Entry{Title: "new title"})
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if updated.Title != "new title" {
		t.Errorf("title not updated: %q", updated.Title)
	}
	// Password should be unchanged
	if updated.Password != "pw" {
		t.Errorf("password changed unexpectedly: %q", updated.Password)
	}
}

func TestDeleteEntry(t *testing.T) {
	path := tempVaultPath(t)
	v, _ := Create(path, testPassword)
	defer v.Close()

	e, _ := v.Add(model.Entry{Title: "to delete"})
	if err := v.Delete(e.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := v.Get(e.ID); err == nil {
		t.Fatal("expected error getting deleted entry")
	}
}

func TestDeleteNotFound(t *testing.T) {
	path := tempVaultPath(t)
	v, _ := Create(path, testPassword)
	defer v.Close()

	if err := v.Delete("nonexistent-id"); err == nil {
		t.Fatal("expected error deleting nonexistent entry")
	}
}

func TestSearch(t *testing.T) {
	path := tempVaultPath(t)
	v, _ := Create(path, testPassword)
	defer v.Close()

	v.Add(model.Entry{Title: "GitHub", URL: "https://github.com", Tags: []string{"dev"}})
	v.Add(model.Entry{Title: "Google", URL: "https://google.com"})
	v.Add(model.Entry{Title: "AWS Console", Notes: "production account"})

	cases := []struct {
		query string
		want  int
	}{
		{"github", 1},
		{"google", 1},
		{"production", 1},
		{"dev", 1},     // tag match
		{"com", 2},     // URL match for github and google
		{"nothing", 0},
	}

	for _, tc := range cases {
		results := v.Search(tc.query)
		if len(results) != tc.want {
			t.Errorf("Search(%q): got %d results, want %d", tc.query, len(results), tc.want)
		}
	}
}

func TestPersistence(t *testing.T) {
	path := tempVaultPath(t)

	// Create vault and add entry.
	v, _ := Create(path, testPassword)
	e, _ := v.Add(model.Entry{Title: "Persistent"})
	v.Close()

	// Reopen and verify entry survived.
	v2, err := Open(path, testPassword)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer v2.Close()

	got, err := v2.Get(e.ID)
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}
	if got.Title != "Persistent" {
		t.Errorf("title: %q", got.Title)
	}
}

func TestList(t *testing.T) {
	path := tempVaultPath(t)
	v, _ := Create(path, testPassword)
	defer v.Close()

	v.Add(model.Entry{Title: "A"})
	v.Add(model.Entry{Title: "B"})
	v.Add(model.Entry{Title: "C"})

	list := v.List()
	if len(list) != 3 {
		t.Errorf("List: got %d entries, want 3", len(list))
	}
}

func TestImportEntries(t *testing.T) {
	path := tempVaultPath(t)
	v, _ := Create(path, testPassword)
	defer v.Close()

	imported := []model.Entry{
		{Title: "Imported A", Password: "pa"},
		{Title: "Imported B", Password: "pb"},
	}
	if err := v.ImportEntries(imported); err != nil {
		t.Fatalf("ImportEntries: %v", err)
	}

	list := v.List()
	if len(list) != 2 {
		t.Errorf("expected 2 entries after import, got %d", len(list))
	}
	for _, e := range list {
		if e.ID == "" {
			t.Error("imported entry missing ID")
		}
	}
}
