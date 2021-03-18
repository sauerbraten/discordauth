package auth

import (
	"testing"
)

func TestChallenge(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Error("could not generate keys")
	}

	challenge, solution, err := GenerateChallenge(pub)
	if err != nil {
		t.Error("could not generate challenge from public key")
	}

	answer, err := Solve(challenge, priv)
	if err != nil {
		t.Error("could not solve challenge using private key")
	}

	if answer != solution {
		t.Errorf("challenge answer does not match solution (expected %v, got %v)", solution, answer)
	}
}
