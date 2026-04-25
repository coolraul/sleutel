package model

// VaultHeader is serialized as a newline-terminated JSON line in the vault file,
// after the magic bytes and version byte. It contains the parameters needed to
// derive the encryption key before the encrypted payload can be read.
type VaultHeader struct {
	KDF     string `json:"kdf"`      // always "argon2id"
	Time    uint32 `json:"time"`     // argon2 time parameter
	Memory  uint32 `json:"memory"`   // argon2 memory in KiB
	Threads uint8  `json:"threads"`  // argon2 parallelism
	Salt    string `json:"salt"`     // base64-encoded 32-byte salt
	Nonce   string `json:"nonce"`    // base64-encoded 12-byte GCM nonce
}

const (
	Magic          = "SLKV"
	FormatVersion  = byte(0x01)
)
