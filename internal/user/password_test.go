package user

import "testing"

func TestPasswordHashAndVerify(t *testing.T) {
	salt, err := GenerateSaltHex()
	if err != nil {
		t.Fatalf("GenerateSaltHex: %v", err)
	}
	hash, err := HashPassword("p@ssw0rd", salt)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Fatalf("expected non-empty hash")
	}
	if !VerifyPassword("p@ssw0rd", salt, hash) {
		t.Fatalf("expected verify ok")
	}
	if VerifyPassword("wrong", salt, hash) {
		t.Fatalf("expected verify fail")
	}
}
