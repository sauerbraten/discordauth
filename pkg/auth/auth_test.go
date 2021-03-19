package auth

import (
	"testing"
)

var keys = []struct {
	priv, pub string
}{
	{
		"f373de2d49584e7a16166e76b1bb925f24f0130c63ac9332",
		"+2c1fb1dd4f2a7b9d81320497c64983e92cda412ed50f33aa",
	},
	{
		"0978245f7f243e61fca787f53bf9c82eabf87e2eeffbbe77",
		"-afe5929327bd76371626cce7585006067603daf76f09c27e",
	},
	{
		"935f7b951c132951527ab541ffc5b8bff258c1e88414ab2a",
		"-d954ee56eddf2b71e206e67d48aaf4afe1cc70f8ca9d1058",
	},
	{
		"f6295aa51aca7f511c441e754830cf0d951a2078cbf881d9",
		"-454c98466c45fce242724e6e989bdd9f841304a1fcba4787",
	},
	{
		"e9ee7bf32f60110b2a0355ccbf120404307de5ee72a41417",
		"+15fda493cb1095ca40f652b0d208769bd42b9e234e48d1a8",
	},
	{
		"8ef7537b1e631ca7c30a4fe8f70d61b7f2589c9ba1f97b0f",
		"+643d99cb21178557f4e965eb6dc1ec1e4f57b3b05375fafb",
	},
}

func TestAuth(t *testing.T) {
	for i, pair := range keys {
		priv, err := ParsePrivateKey(pair.priv)
		if err != nil {
			t.Errorf("parsing private part of key pair %d: %v", i+1, err)
		}
		pub, err := ParsePublicKey(pair.pub)
		if err != nil {
			t.Errorf("parsing public part of key pair %d: %v", i+1, err)
		}

		challenge, solution, err := GenerateChallenge(pub)
		if err != nil {
			t.Errorf("could not generate challenge from public key of pair %d: %v", i+1, err)
		}

		answer, err := Solve(challenge, priv)
		if err != nil {
			t.Errorf("could not solve challenge using private key of pair %d: %v", i+1, err)
		}

		if answer != solution {
			t.Errorf("challenge answer does not match solution (expected %v, got %v)", solution, answer)
		}
	}
}

func TestPrivateKey(t *testing.T) {
	for i, pair := range keys {
		priv, err := ParsePrivateKey(pair.priv)
		if err != nil {
			t.Errorf("parsing private part of key pair %d: %v", i+1, err)
		}
		if priv.String() != pair.priv {
			t.Errorf("encoding of private key %d does not match (expected %v, got %v)", i+1, pair.priv, priv.String())
		}
	}
}

func TestPublicKey(t *testing.T) {
	for i, pair := range keys {
		pub, err := ParsePublicKey(pair.pub)
		if err != nil {
			t.Errorf("parsing public part of key pair %d: %v", i+1, err)
		}
		if pub.String() != pair.pub {
			t.Errorf("encoding of public key %d does not match (expected %v, got %v)", i+1, pair.pub, pub.String())
		}
	}
}

func TestGenerateKeyPair(t *testing.T) {
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
