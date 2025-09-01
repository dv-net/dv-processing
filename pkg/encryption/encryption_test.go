package encryption_test

import (
	"fmt"
	"testing"

	"github.com/dv-net/dv-processing/pkg/encryption"
)

func TestEncryptDecrypt(t *testing.T) {
	password := "testpassword"
	plaintext := "primary unique toss tuition defense alone artefact tube chalk wrist plunge gym vast sail boost junk fancy forest plastic raise hundred swallow weasel pepper"
	cipher, err := encryption.Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}
	if !encryption.IsEncrypted(cipher) {
		t.Errorf("Expected IsEncrypted to return true for %q", cipher)
	}
	decrypted, err := encryption.Decrypt(cipher, password)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("Decrypted text %q does not match original %q", decrypted, plaintext)
	}

	fmt.Println("Encrypted:", cipher)
	fmt.Println("Decrypted:", decrypted)
}

func TestIsEncryptedFalse(t *testing.T) {
	if encryption.IsEncrypted("not encrypted data") {
		t.Errorf("Expected IsEncrypted to return false")
	}
}

func TestDecryptInvalid(t *testing.T) {
	_, err := encryption.Decrypt("ENCv1:invalidbase64", "password")
	if err == nil {
		t.Errorf("Expected error when decrypting invalid data")
	}
}

func BenchmarkEncrypt(b *testing.B) {
	password := "benchpassword"
	plaintext := "benchmark data to encrypt"
	for i := 0; i < b.N; i++ {
		_, err := encryption.Encrypt(plaintext, password)
		if err != nil {
			b.Fatalf("Encrypt error: %v", err)
		}
	}
}

func BenchmarkDecrypt(b *testing.B) {
	password := "benchpassword"
	plaintext := "benchmark data to encrypt"
	cipher, err := encryption.Encrypt(plaintext, password)
	if err != nil {
		b.Fatalf("Encrypt error: %v", err)
	}
	for i := 0; i < b.N; i++ {
		_, err := encryption.Decrypt(cipher, password)
		if err != nil {
			b.Fatalf("Decrypt error: %v", err)
		}
	}
}
