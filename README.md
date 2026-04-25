# sleutel

A local-first, encrypted password manager CLI written in Go.

---

## Threat Model

**Protected against:**
- Vault file theft (disk image, backup exfiltration) — AES-256-GCM encryption with Argon2id-derived key means an attacker needs the master password to read anything.
- Offline brute-force attacks — Argon2id parameters (time=3, memory=64MiB, threads=4) make each guess expensive (~100ms on modern hardware).
- Ciphertext tampering — GCM authentication tag detects any modification to the encrypted payload.
- Accidental plaintext writes — entries are never written to disk unencrypted; export requires explicit opt-in and a warning prompt.
- Nonce reuse — a fresh random 96-bit nonce is generated on every vault save.

**Not protected against:**
- Compromise of the machine while the vault is open — a keylogger or memory dump can extract the master password or derived key.
- Weak master passwords — sleutel does not enforce complexity; the user is responsible.
- Side-channel attacks on the KDF — out of scope for a local CLI tool.
- Clipboard sniffing — clipboard support is not implemented in phase 1.

---

## Installation

```sh
go install github.com/mms/sleutel/cmd/sleutel@latest
```

Or build from source:

```sh
git clone https://github.com/mms/sleutel
cd sleutel
go build -o sleutel ./cmd/sleutel
```

---

## Commands

All commands accept `--vault <path>` to override the default vault location (`~/.sleutel/vault.slkv`).

### Initialize a new vault

```sh
sleutel init
# Master password:
# Confirm master password:
# Vault created at /Users/alice/.sleutel/vault.slkv
```

### Add an entry

```sh
sleutel add --title "GitHub" --username alice --url https://github.com --tags dev,work

# Generate a password automatically:
sleutel add --title "AWS" --username alice --generate --gen-length 32 --gen-symbols
```

### Get an entry

```sh
sleutel get <id>
sleutel get <id> --show     # reveal password
```

### List all entries

```sh
sleutel list
sleutel list --tag dev
```

### Edit an entry

```sh
sleutel edit <id> --title "New Title"
sleutel edit <id> --password newpassword --tags updated,tags
```

### Delete an entry

```sh
sleutel delete <id>
sleutel delete <id> --force   # skip confirmation
```

### Search

```sh
sleutel search github
sleutel search production     # matches notes, URL, title, tags
```

### Generate a password (without storing)

```sh
sleutel generate
sleutel generate --length 32 --symbols=false
```

### Export (plaintext — handle with care)

```sh
sleutel export --file backup.json
sleutel export --file backup.json --yes   # skip confirmation
```

### Import

```sh
sleutel import --file backup.json
```

---

## Vault File Format

```
[4 bytes]  magic: "SLKV"
[1 byte]   version: 0x01
[N bytes]  JSON header (newline-terminated):
           { "kdf":"argon2id", "time":3, "memory":65536, "threads":4,
             "salt":"<base64-32-bytes>", "nonce":"<base64-12-bytes>" }
[rest]     AES-256-GCM ciphertext of gzip(JSON([]Entry))
           — GCM tag appended by standard seal operation
```

The header is readable without the password (it holds the KDF parameters needed to derive the key). The encrypted payload is opaque and authenticated — any modification to either the payload or the header-derived key causes decryption to fail.

---

## Project Structure

```
cmd/sleutel/          entry point
internal/cli/         command handlers (no business logic)
internal/crypto/      argon2id KDF, AES-256-GCM
internal/model/       Entry and VaultHeader types
internal/storage/     vault file serialisation / deserialisation
internal/vault/       vault service: open, add, edit, delete, search, generate
```

---

## Phase 2 Notes

- **Session unlock** — cache derived key in a protected in-memory session (or OS keychain) to avoid prompting for the password on every command.
- **Rekey** — re-derive key with new password and re-encrypt; design is ready (salt/params are in the header).
- **Clipboard** — `sleutel get <id> --clip` to write password to clipboard and clear after 30s.
- **TUI** — interactive fuzzy search and selection.
- **OTP** — TOTP secret storage and code generation.
- **Multiple vaults** — already supported via `--vault`; add named profiles in config.
- **Audit log** — append-only local log of vault operations (no secrets).
