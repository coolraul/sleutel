package storage

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mms/sleutel/internal/model"
)

// VaultFile is the on-disk representation of the vault.
type VaultFile struct {
	Header  model.VaultHeader
	Payload []byte // encrypted, opaque
}

// Write serialises the vault file to path atomically.
// Format: [4-byte magic][1-byte version][JSON header\n][encrypted payload]
func Write(path string, vf VaultFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	var buf bytes.Buffer

	buf.WriteString(model.Magic)
	buf.WriteByte(model.FormatVersion)

	hdr, err := json.Marshal(vf.Header)
	if err != nil {
		return fmt.Errorf("marshal header: %w", err)
	}
	buf.Write(hdr)
	buf.WriteByte('\n')
	buf.Write(vf.Payload)

	// Write to a temp file then rename for atomicity.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// Read deserialises a vault file from path.
func Read(path string) (VaultFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return VaultFile{}, fmt.Errorf("read file: %w", err)
	}
	return parse(data)
}

func parse(data []byte) (VaultFile, error) {
	if len(data) < 6 {
		return VaultFile{}, fmt.Errorf("file too short")
	}

	if string(data[:4]) != model.Magic {
		return VaultFile{}, fmt.Errorf("invalid magic bytes")
	}

	if data[4] != model.FormatVersion {
		return VaultFile{}, fmt.Errorf("unsupported vault version: %d", data[4])
	}

	rest := data[5:]
	nl := bytes.IndexByte(rest, '\n')
	if nl < 0 {
		return VaultFile{}, fmt.Errorf("missing header newline")
	}

	var hdr model.VaultHeader
	if err := json.Unmarshal(rest[:nl], &hdr); err != nil {
		return VaultFile{}, fmt.Errorf("unmarshal header: %w", err)
	}

	// Validate base64 fields are present and decodable.
	if _, err := base64.StdEncoding.DecodeString(hdr.Salt); err != nil {
		return VaultFile{}, fmt.Errorf("invalid salt encoding: %w", err)
	}
	if _, err := base64.StdEncoding.DecodeString(hdr.Nonce); err != nil {
		return VaultFile{}, fmt.Errorf("invalid nonce encoding: %w", err)
	}

	payload := rest[nl+1:]

	return VaultFile{Header: hdr, Payload: payload}, nil
}

// Exists reports whether the vault file exists at path.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
