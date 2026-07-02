package sync

import "testing"

func TestSealOpenRoundTrip(t *testing.T) {
	key, err := deriveKey("server-issued-key", "user-salt")
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte(`{"provider":"openai","api_key":"sk-secret"}`)
	payload, nonce, err := seal(key, plain)
	if err != nil {
		t.Fatal(err)
	}
	got, err := open(key, payload, nonce)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(plain) {
		t.Fatalf("round trip mismatch: %s", got)
	}
}

func TestDeriveKeyDiffersBySalt(t *testing.T) {
	a, _ := deriveKey("k", "salt-a")
	b, _ := deriveKey("k", "salt-b")
	if string(a) == string(b) {
		t.Fatal("different salts must derive different keys")
	}
}

func TestOpenWrongKeyFails(t *testing.T) {
	k1, _ := deriveKey("k", "s")
	k2, _ := deriveKey("other", "s")
	p, n, _ := seal(k1, []byte("x"))
	if _, err := open(k2, p, n); err == nil {
		t.Fatal("expected auth failure with wrong key")
	}
}
