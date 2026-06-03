package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Argon2Config struct {
	Pepper  string `env:"PASSWORD_PEPPER,required"`
	KeyLen  uint32 `env:"KEY_LENGTH" envDefault:"32"`
	Memory  uint32 `env:"MEMORY" envDefault:"65536"`
	Time    uint32 `env:"TIME" envDefault:"3"`    // Iterations
	Threads uint8  `env:"THREADS" envDefault:"4"` // Parallelism
}

type Argon2Hasher struct {
	config *Argon2Config
}

func NewArgon2Hasher(config *Argon2Config) *Argon2Hasher {
	return &Argon2Hasher{
		config: config,
	}
}

const (
	// prevent resource exhaustion
	maxAllowedMemory  = 1024 * 1024 // 1GB
	maxAllowedThreads = 16
	maxAllowedTime    = 10
)

func (h *Argon2Hasher) Hash(plaintext string) string {
	salt := make([]byte, 16)
	rand.Read(salt)

	hash := h.hashWithSalt(salt, plaintext)

	return h.toPHCString(salt, hash)
}

func (h *Argon2Hasher) hashWithSalt(salt []byte, plainttext string) []byte {
	return argon2.IDKey([]byte(plainttext+h.config.Pepper), salt, h.config.Time, h.config.Memory, h.config.Threads, h.config.KeyLen)
}

func (h *Argon2Hasher) toPHCString(salt []byte, hash []byte) string {
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.config.Memory, h.config.Time, h.config.Threads, b64Salt, b64Hash)
}

func (h *Argon2Hasher) Verify(plaintext, encodedHash string) (bool, error) {
	// $argon2id$v=19$m=65536,t=3,p=2$c29tZXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG
	// <empty (leading $)>:<algorithm>:<version>:<parameters>:<salt>:<hash>

	// avoid excessive mem allocation on header parsing
	// https://github.com/golang-jwt/jwt/security/advisories/GHSA-mh63-6h87-95cp
	parts := strings.SplitN(encodedHash, "$", 6)

	if len(parts) != 6 {
		return false, fmt.Errorf("Invalid format: received %d parts", len(parts))
	}

	// rfc says version string is 32bit
	version, err := parseParameter(parts[2], 32)
	if err != nil {
		return false, err
	} else if int(version) != argon2.Version {
		return false, fmt.Errorf("Argon2 Version Mismatch: hash using v%d, verifying with v%d", version, argon2.Version)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	if len(hash) > math.MaxUint32 {
		return false, fmt.Errorf("KeyLen exceeds maximum size uint32")
	}

	//<parameters> in form <m=?>,<t=?>,<p=?>
	paramParts := strings.Split(parts[3], ",")

	if len(paramParts) != 3 {
		return false, fmt.Errorf("Invalid format: expected 3 parameters, received %d", len(paramParts))
	}

	memory, err := parseParameter(paramParts[0], 32)
	if err != nil {
		return false, err
	}
	if memory > maxAllowedMemory {
		return false, fmt.Errorf("Invalid Hash: memory parameter '%d' exceeds max allowed '%d'", memory, maxAllowedMemory)
	}

	time, err := parseParameter(paramParts[1], 32)
	if err != nil {
		return false, err
	}
	if time > maxAllowedTime {
		return false, fmt.Errorf("Invalid Hash: time parameter '%d' exceeds max allowed '%d'", time, maxAllowedTime)
	}

	threads, err := parseParameter(paramParts[2], 32)
	if err != nil {
		return false, err
	}
	if threads > maxAllowedThreads {
		return false, fmt.Errorf("Invalid Hash: threads parameter '%d' exceeds max allowed '%d'", threads, maxAllowedThreads)
	}

	v := NewArgon2Hasher(&Argon2Config{
		Pepper:  h.config.Pepper,
		KeyLen:  uint32(len(hash)),
		Memory:  uint32(memory),
		Time:    uint32(time),
		Threads: uint8(threads),
	})

	computed := v.hashWithSalt(salt, plaintext)

	return subtle.ConstantTimeCompare(computed, hash) == 1, nil

}

func parseParameter(s string, bitSize int) (uint64, error) {
	parts := strings.Split(s, "=")

	if len(parts) != 2 {
		return 0, fmt.Errorf("Invalid parameter: received %d parts", len(parts))
	}

	value, err := strconv.ParseUint(parts[1], 10, bitSize)

	if err != nil {
		return 0, err
	}

	return value, nil
}
