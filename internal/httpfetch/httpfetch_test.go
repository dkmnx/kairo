package httpfetch

import (
	"testing"
)

func TestVerifySHA256(t *testing.T) {
	t.Run("matching hash", func(t *testing.T) {
		data := []byte("hello world")
		// sha256("hello world") = b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9
		err := VerifySHA256(data, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9")
		if err != nil {
			t.Errorf("VerifySHA256() error = %v", err)
		}
	})
	t.Run("mismatch", func(t *testing.T) {
		data := []byte("hello world")
		err := VerifySHA256(data, "0000000000000000000000000000000000000000000000000000000000000000")
		if err == nil {
			t.Error("should return error on hash mismatch")
		}
	})
	t.Run("case insensitive", func(t *testing.T) {
		data := []byte("hello world")
		err := VerifySHA256(data, "B94D27B9934D3E08A52E52D7DA7DABFAC484EFE37A5380EE9088F7ACE2EFCDE9")
		if err != nil {
			t.Errorf("should be case-insensitive, error = %v", err)
		}
	})
	t.Run("empty data", func(t *testing.T) {
		// sha256("") = e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
		err := VerifySHA256([]byte{}, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
		if err != nil {
			t.Errorf("VerifySHA256() error = %v", err)
		}
	})
	t.Run("trims whitespace in expected hash", func(t *testing.T) {
		data := []byte("test")
		// sha256("test") = 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
		err := VerifySHA256(data, "  9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08\n")
		if err != nil {
			t.Errorf("VerifySHA256() should trim whitespace, error = %v", err)
		}
	})
}
