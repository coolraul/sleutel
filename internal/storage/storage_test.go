package storage

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/mms/sleutel/internal/model"
)

func makeVaultFile() VaultFile {
	salt := base64.StdEncoding.EncodeToString(make([]byte, 32))
	nonce := base64.StdEncoding.EncodeToString(make([]byte, 12))
	return VaultFile{
		Header: model.VaultHeader{
			KDF:     "argon2id",
			Time:    3,
			Memory:  65536,
			Threads: 4,
			Salt:    salt,
			Nonce:   nonce,
		},
		Payload: []byte("encryptedblob"),
	}
}

func TestWriteRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.slkv")

	vf := makeVaultFile()
	if err := Write(path, vf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if got.Header.KDF != vf.Header.KDF {
		t.Errorf("KDF mismatch: %q vs %q", got.Header.KDF, vf.Header.KDF)
	}
	if got.Header.Salt != vf.Header.Salt {
		t.Errorf("Salt mismatch")
	}
	if string(got.Payload) != string(vf.Payload) {
		t.Errorf("Payload mismatch: %q vs %q", got.Payload, vf.Payload)
	}
}

func TestReadInvalidMagic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.slkv")
	_ = os.WriteFile(path, []byte("XXXX\x01{}"), 0600)
	_, err := Read(path)
	if err == nil {
		t.Fatal("expected error for invalid magic, got nil")
	}
}

func TestReadUnsupportedVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.slkv")
	_ = os.WriteFile(path, []byte("SLKV\x99{}\n"), 0600)
	_, err := Read(path)
	if err == nil {
		t.Fatal("expected error for unsupported version, got nil")
	}
}

func TestReadMissingHeaderNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.slkv")
	_ = os.WriteFile(path, []byte("SLKV\x01no-newline-ever"), 0600)
	_, err := Read(path)
	if err == nil {
		t.Fatal("expected error for missing newline, got nil")
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.slkv")

	if Exists(path) {
		t.Fatal("expected Exists=false before creation")
	}
	_ = Write(path, makeVaultFile())
	if !Exists(path) {
		t.Fatal("expected Exists=true after creation")
	}
}

func TestWriteCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "dir", "vault.slkv")
	if err := Write(path, makeVaultFile()); err != nil {
		t.Fatalf("Write with nested dirs: %v", err)
	}
	if !Exists(path) {
		t.Fatal("vault file not found after write")
	}
}
