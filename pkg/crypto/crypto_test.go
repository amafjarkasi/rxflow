package crypto

import (
	"crypto/rand"
	"testing"
)

func TestEncryptor_EncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	encryptor, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	plaintext := "sensitive_patient_data_12345"

	ciphertext, err := encryptor.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt string: %v", err)
	}

	if plaintext == ciphertext {
		t.Error("ciphertext matches plaintext")
	}

	decrypted, err := encryptor.DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("failed to decrypt string: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("expected %q, got %q", plaintext, decrypted)
	}

	// Tampering test
	tamperedCiphertext := ciphertext[:len(ciphertext)-5] + "AAAAA"
	_, err = encryptor.DecryptString(tamperedCiphertext)
	if err == nil {
		t.Error("expected decryption failure for tampered ciphertext")
	}
}

func TestHasher_HashPatientData(t *testing.T) {
	salt := []byte("a_very_secure_random_salt_for_hmac")

	hasher, err := NewHasher(salt)
	if err != nil {
		t.Fatalf("failed to create hasher: %v", err)
	}

	id := "MRN-12345"
	dob := "1990-01-01"
	name := "John Doe"

	hash1 := hasher.HashPatientData(id, dob, name)

	if hash1 == "" {
		t.Error("hash cannot be empty")
	}

	if !hasher.VerifyPatientData(id, dob, name, hash1) {
		t.Error("expected verification to pass")
	}

	if hasher.VerifyPatientData("MRN-99999", dob, name, hash1) {
		t.Error("expected verification to fail for different identifier")
	}
}
