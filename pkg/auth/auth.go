// package auth implements server-side functionality for Sauerbraten's player authentication mechanism.
//
// The mechanism relies on the associativity of scalar multiplication on elliptic curves: private keys
// are random (big) scalars, and the corresponding public key is created by multiplying the curve base point
// with the private key. (This means the public key is another point on the curve.)
// To check for posession of the private key belonging to a public key known to the server, the base point is
// multiplied with another random, big scalar (the "secret") and the resulting point is sent to the user as
// "challenge". The user multiplies the challenge curve point with his private key (a scalar), and sends the
// X coordinate of the resulting point back to the server.
// The server instead multiplies the user's public key with the secret scalar. Since pub = base * priv,
// pub * secret = (base * priv) * secret = (base * secret) * priv = challenge * priv. Because of the curve's
// symmetry, there are exactly two points on the curve at any given X. For simplicity (and maybe performance),
// the server is satisfied when the user responds with the correct X.
package auth

import (
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
)

// from ecjacobian::print() in shared/crypto.cpp
func encodePoint(x, y *big.Int) (s string) {
	if y.Bit(0) == 1 {
		s += "-"
	} else {
		s += "+"
	}
	s += x.Text(16)
	return
}

func GenerateChallenge(pub PublicKey) (challenge, solution string, err error) {
	secret, x, y, err := elliptic.GenerateKey(p192, rand.Reader)

	// what we send to the client
	challenge = encodePoint(x, y)

	// what the client should return if she applies her private key to the challenge
	solX, _ := p192.ScalarMult(pub.x, pub.y, secret)
	solution = solX.Text(16)

	return
}
