package crypto

import "testing"

func TestBootstrapTokenHashing(t *testing.T) {
	token, err := NewBootstrapToken()
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("empty token")
	}
	if HashToken(token) == HashToken("other") {
		t.Fatal("hash collision for distinct test tokens")
	}
}
