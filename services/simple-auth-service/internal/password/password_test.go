package password

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/crypto/argon2"
)

// newTestHasher returns an Argon2Hasher with cheap-but-valid params suitable for tests.
func newTestHasher(pepper string) *Argon2Hasher {
	return &Argon2Hasher{
		config: &Argon2Config{
			Pepper:  pepper,
			KeyLen:  32,
			Memory:  64 * 1024,
			Time:    1,
			Threads: 1,
		},
	}
}

// TestHashVerify_HappyPath ensures a freshly hashed password verifies successfully.
func TestHashVerify_HappyPath(t *testing.T) {
	const plaintext = "correct-horse-battery-staple"
	const pepper = "test-pepper"

	h := newTestHasher(pepper)
	encoded := h.Hash(plaintext)

	ok, err := h.Verify(plaintext, encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected verification to succeed but it failed")
	}
}

// TestVerify_WrongPassword ensures a mismatched password returns false.
func TestVerify_WrongPassword(t *testing.T) {
	const pepper = "test-pepper"
	h := newTestHasher(pepper)
	encoded := h.Hash("correct-password")

	ok, err := h.Verify("wrong-password", encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected verification to fail for wrong password but it succeeded")
	}
}

// TestVerify_WrongPepper ensures a mismatched pepper returns false.
func TestVerify_WrongPepper(t *testing.T) {
	h := newTestHasher("correct-pepper")
	encoded := h.Hash("my-password")
	wrongPepperHasher := newTestHasher("wrong-pepper")

	ok, err := wrongPepperHasher.Verify("my-password", encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected verification to fail for wrong pepper but it succeeded")
	}
}

// TestVerify_MalformedHash covers invalid PHC string structures.
func TestVerify_MalformedHash(t *testing.T) {
	validHash := func() string {
		return newTestHasher("p").Hash("pw")
	}

	// Build a valid hash once to use as the base for surgical mutations.
	base := validHash()
	parts := strings.Split(base, "$")
	// parts: ["", "argon2id", "v=19", "m=65536,t=1,p=1", "<salt>", "<hash>"]

	cases := []struct {
		name  string
		input string
	}{
		{
			name:  "too few segments",
			input: "$argon2id$v=19$m=65536,t=1,p=1$onlysalt",
		},
		{
			name:  "too many segments",
			input: base + "$extra",
		},
		{
			name:  "wrong algorithm version",
			input: strings.Replace(base, fmt.Sprintf("v=%d", argon2.Version), "v=1", 1),
		},
		{
			name:  "invalid version not a number",
			input: strings.Join([]string{parts[0], parts[1], "v=abc", parts[3], parts[4], parts[5]}, "$"),
		},
		{
			name:  "invalid salt base64",
			input: strings.Join([]string{parts[0], parts[1], parts[2], parts[3], "!!!bad-b64!!!", parts[5]}, "$"),
		},
		{
			name:  "invalid hash base64",
			input: strings.Join([]string{parts[0], parts[1], parts[2], parts[3], parts[4], "!!!bad-b64!!!"}, "$"),
		},
		{
			name:  "missing parameter key (no equals sign)",
			input: strings.Join([]string{parts[0], parts[1], parts[2], "65536,t=1,p=1", parts[4], parts[5]}, "$"),
		},
		{
			name:  "memory parameter not a number",
			input: strings.Join([]string{parts[0], parts[1], parts[2], "m=abc,t=1,p=1", parts[4], parts[5]}, "$"),
		},
		{
			name:  "time parameter not a number",
			input: strings.Join([]string{parts[0], parts[1], parts[2], "m=65536,t=abc,p=1", parts[4], parts[5]}, "$"),
		},
		{
			name:  "threads parameter not a number",
			input: strings.Join([]string{parts[0], parts[1], parts[2], "m=65536,t=1,p=abc", parts[4], parts[5]}, "$"),
		},
		{
			name:  "fewer than 3 parameter fields",
			input: strings.Join([]string{parts[0], parts[1], parts[2], "m=65536,t=1", parts[4], parts[5]}, "$"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newTestHasher("any-pepper").Verify("any-plaintext", tc.input)
			if err == nil {
				t.Errorf("expected an error for case %q but got nil", tc.name)
			}
		})
	}
}

// TestVerify_ParametersExceedMaximums ensures hashes with oversized parameters are rejected.
func TestVerify_ParametersExceedMaximums(t *testing.T) {
	base := newTestHasher("p").Hash("pw")
	parts := strings.Split(base, "$")
	// parts[3] is the parameter segment, e.g. "m=65536,t=1,p=1"

	cases := []struct {
		name   string
		params string
	}{
		{
			name:   "memory exceeds maximum",
			params: fmt.Sprintf("m=%d,t=1,p=1", maxAllowedMemory+1),
		},
		{
			name:   "time exceeds maximum",
			params: fmt.Sprintf("m=65536,t=%d,p=1", maxAllowedTime+1),
		},
		{
			name:   "threads exceeds maximum",
			params: fmt.Sprintf("m=65536,t=1,p=%d", maxAllowedThreads+1),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := strings.Join([]string{parts[0], parts[1], parts[2], tc.params, parts[4], parts[5]}, "$")
			_, err := newTestHasher("any-pepper").Verify("any-plaintext", mutated)
			if err == nil {
				t.Errorf("expected an error for case %q but got nil", tc.name)
			}
		})
	}
}
