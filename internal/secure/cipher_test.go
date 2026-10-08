package secure

import (
	"strings"
	"testing"
)

func TestCipherRoundTrip(t *testing.T) {
	c, err := NewCipher(strings.Repeat("ab", 32))
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := c.Encrypt("super-secret")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "super-secret" {
		t.Fatal("ciphertext must not equal plaintext")
	}
	plain, err := c.Decrypt(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "super-secret" {
		t.Fatalf("got %q", plain)
	}
}

func TestCipherRejectsBadKey(t *testing.T) {
	if _, err := NewCipher("short"); err == nil {
		t.Fatal("expected invalid key error")
	}
}
