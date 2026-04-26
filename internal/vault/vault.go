package vault

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mms/sleutel/internal/crypto"
	"github.com/mms/sleutel/internal/model"
	"github.com/mms/sleutel/internal/storage"
)

// Vault holds the in-memory decrypted state of an open vault.
type Vault struct {
	path    string
	entries []model.Entry
	key     []byte // derived from master password; zeroed on close
	header  model.VaultHeader
}

// Create initialises a new vault file at path with the given master password.
// Fails if the file already exists.
func Create(path string, password []byte) (*Vault, error) {
	if storage.Exists(path) {
		return nil, fmt.Errorf("vault already exists at %s", path)
	}

	salt, err := crypto.RandBytes(crypto.SaltLen)
	if err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	nonce, err := crypto.RandBytes(crypto.NonceLen)
	if err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	p := crypto.DefaultParams()
	key := crypto.DeriveKey(password, salt, p)

	hdr := model.VaultHeader{
		KDF:     "argon2id",
		Time:    p.Time,
		Memory:  p.Memory,
		Threads: p.Threads,
		Salt:    base64.StdEncoding.EncodeToString(salt),
		Nonce:   base64.StdEncoding.EncodeToString(nonce),
	}

	v := &Vault{path: path, key: key, header: hdr}
	if err := v.flush(nonce); err != nil {
		crypto.Zero(key)
		return nil, fmt.Errorf("initial flush: %w", err)
	}
	return v, nil
}

// Open decrypts the vault at path using the given master password.
func Open(path string, password []byte) (*Vault, error) {
	vf, err := storage.Read(path)
	if err != nil {
		return nil, fmt.Errorf("read vault: %w", err)
	}

	salt, err := base64.StdEncoding.DecodeString(vf.Header.Salt)
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(vf.Header.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}

	p := crypto.Params{
		Time:    vf.Header.Time,
		Memory:  vf.Header.Memory,
		Threads: vf.Header.Threads,
	}
	key := crypto.DeriveKey(password, salt, p)

	plain, err := crypto.Decrypt(key, nonce, vf.Payload)
	if err != nil {
		crypto.Zero(key)
		return nil, err // already a user-friendly message from crypto package
	}

	var entries []model.Entry
	if err := json.Unmarshal(plain, &entries); err != nil {
		crypto.Zero(key)
		return nil, fmt.Errorf("unmarshal entries: %w", err)
	}

	return &Vault{path: path, entries: entries, key: key, header: vf.Header}, nil
}

// Close zeros the key material in memory.
func (v *Vault) Close() {
	crypto.Zero(v.key)
	v.key = nil
	v.entries = nil
}

// Add inserts a new entry and saves the vault.
func (v *Vault) Add(e model.Entry) (model.Entry, error) {
	now := time.Now().UTC()
	e.ID = uuid.New().String()
	e.CreatedAt = now
	e.UpdatedAt = now
	v.entries = append(v.entries, e)
	if err := v.save(); err != nil {
		v.entries = v.entries[:len(v.entries)-1]
		return model.Entry{}, err
	}
	return e, nil
}

// Get returns the entry with the given ID, or an error if not found.
func (v *Vault) Get(id string) (model.Entry, error) {
	for _, e := range v.entries {
		if e.ID == id {
			return e, nil
		}
	}
	return model.Entry{}, fmt.Errorf("entry not found: %s", id)
}

// Edit replaces the entry with the given ID. Only non-empty fields in patch are applied.
func (v *Vault) Edit(id string, patch model.Entry) (model.Entry, error) {
	for i, e := range v.entries {
		if e.ID != id {
			continue
		}
		orig := e
		applyPatch(&v.entries[i], patch)
		v.entries[i].UpdatedAt = time.Now().UTC()
		if err := v.save(); err != nil {
			v.entries[i] = orig
			return model.Entry{}, err
		}
		return v.entries[i], nil
	}
	return model.Entry{}, fmt.Errorf("entry not found: %s", id)
}

// Delete removes the entry with the given ID.
func (v *Vault) Delete(id string) error {
	for i, e := range v.entries {
		if e.ID != id {
			continue
		}
		orig := make([]model.Entry, len(v.entries))
		copy(orig, v.entries)
		v.entries = append(v.entries[:i], v.entries[i+1:]...)
		if err := v.save(); err != nil {
			v.entries = orig
			return err
		}
		return nil
	}
	return fmt.Errorf("entry not found: %s", id)
}

// AddSecurityQuestion appends a security question to the entry and saves.
func (v *Vault) AddSecurityQuestion(entryID string, sq model.SecurityQuestion) (model.Entry, error) {
	for i, e := range v.entries {
		if e.ID != entryID {
			continue
		}
		orig := v.entries[i]
		v.entries[i].SecurityQuestions = append(v.entries[i].SecurityQuestions, sq)
		v.entries[i].UpdatedAt = time.Now().UTC()
		if err := v.save(); err != nil {
			v.entries[i] = orig
			return model.Entry{}, err
		}
		return v.entries[i], nil
	}
	return model.Entry{}, fmt.Errorf("entry not found: %s", entryID)
}

// UpdateSecurityQuestion replaces the security question at idx and saves.
func (v *Vault) UpdateSecurityQuestion(entryID string, idx int, sq model.SecurityQuestion) (model.Entry, error) {
	for i, e := range v.entries {
		if e.ID != entryID {
			continue
		}
		if idx < 0 || idx >= len(v.entries[i].SecurityQuestions) {
			return model.Entry{}, fmt.Errorf("security question index out of range: %d", idx)
		}
		orig := v.entries[i]
		v.entries[i].SecurityQuestions[idx] = sq
		v.entries[i].UpdatedAt = time.Now().UTC()
		if err := v.save(); err != nil {
			v.entries[i] = orig
			return model.Entry{}, err
		}
		return v.entries[i], nil
	}
	return model.Entry{}, fmt.Errorf("entry not found: %s", entryID)
}

// DeleteSecurityQuestion removes the security question at idx and saves.
func (v *Vault) DeleteSecurityQuestion(entryID string, idx int) (model.Entry, error) {
	for i, e := range v.entries {
		if e.ID != entryID {
			continue
		}
		if idx < 0 || idx >= len(v.entries[i].SecurityQuestions) {
			return model.Entry{}, fmt.Errorf("security question index out of range: %d", idx)
		}
		orig := v.entries[i]
		sqs := v.entries[i].SecurityQuestions
		v.entries[i].SecurityQuestions = append(sqs[:idx], sqs[idx+1:]...)
		v.entries[i].UpdatedAt = time.Now().UTC()
		if err := v.save(); err != nil {
			v.entries[i] = orig
			return model.Entry{}, err
		}
		return v.entries[i], nil
	}
	return model.Entry{}, fmt.Errorf("entry not found: %s", entryID)
}

// List returns all entries (no passwords exposed).
func (v *Vault) List() []model.Entry {
	out := make([]model.Entry, len(v.entries))
	copy(out, v.entries)
	return out
}

// Search returns entries whose title, URL, notes, or tags contain query (case-insensitive).
func (v *Vault) Search(query string) []model.Entry {
	q := strings.ToLower(query)
	var out []model.Entry
	for _, e := range v.entries {
		if matchEntry(e, q) {
			out = append(out, e)
		}
	}
	return out
}

// Entries returns a copy of all entries including passwords — used for export only.
func (v *Vault) Entries() []model.Entry {
	out := make([]model.Entry, len(v.entries))
	copy(out, v.entries)
	return out
}

// ImportEntries bulk-imports entries, assigning new IDs and timestamps.
func (v *Vault) ImportEntries(entries []model.Entry) error {
	now := time.Now().UTC()
	added := make([]model.Entry, 0, len(entries))
	for _, e := range entries {
		e.ID = uuid.New().String()
		e.CreatedAt = now
		e.UpdatedAt = now
		added = append(added, e)
	}
	orig := v.entries
	v.entries = append(v.entries, added...)
	if err := v.save(); err != nil {
		v.entries = orig
		return err
	}
	return nil
}

// Path returns the vault file path.
func (v *Vault) Path() string { return v.path }

// --- internal helpers ---

func (v *Vault) save() error {
	// Generate a fresh nonce on every save — reusing nonce+key is a GCM footgun.
	nonce, err := crypto.RandBytes(crypto.NonceLen)
	if err != nil {
		return fmt.Errorf("generate nonce: %w", err)
	}
	return v.flush(nonce)
}

func (v *Vault) flush(nonce []byte) error {
	plain, err := json.Marshal(v.entries)
	if err != nil {
		return fmt.Errorf("marshal entries: %w", err)
	}

	ct, err := crypto.Encrypt(v.key, nonce, plain)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	// Update the nonce in the header for the new ciphertext.
	v.header.Nonce = base64.StdEncoding.EncodeToString(nonce)

	vf := storage.VaultFile{
		Header:  v.header,
		Payload: ct,
	}
	return storage.Write(v.path, vf)
}

func applyPatch(e *model.Entry, patch model.Entry) {
	if patch.Title != "" {
		e.Title = patch.Title
	}
	if patch.Username != "" {
		e.Username = patch.Username
	}
	if patch.Password != "" {
		e.Password = patch.Password
	}
	if patch.URL != "" {
		e.URL = patch.URL
	}
	if patch.Notes != "" {
		e.Notes = patch.Notes
	}
	if len(patch.Tags) > 0 {
		e.Tags = patch.Tags
	}
}

func matchEntry(e model.Entry, q string) bool {
	if strings.Contains(strings.ToLower(e.Title), q) {
		return true
	}
	if strings.Contains(strings.ToLower(e.URL), q) {
		return true
	}
	if strings.Contains(strings.ToLower(e.Notes), q) {
		return true
	}
	for _, t := range e.Tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	return false
}

// ErrNotFound is returned when an entry lookup fails.
var ErrNotFound = errors.New("entry not found")
