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
	"errors"
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

func parsePoint(s string) (x, y *big.Int, err error) {
	if len(s) < 1 {
		return nil, nil, errors.New("auth: could not parse curve point: too short")
	}

	x = new(big.Int)
	y = new(big.Int)
	xxx := new(big.Int)
	threeX := new(big.Int)
	y2 := new(big.Int)

	_, ok := x.SetString(s[1:], 16)
	if !ok {
		return nil, nil, errors.New("auth: could not set X coordinate of curve point")
	}

	// the next steps find y using the formula y^2 = x^3 - 3*x + B
	xxx.Mul(x, x).Mul(xxx, x)           // x^3
	threeX.Add(x, x).Add(threeX, x)     // 3*x
	y2.Sub(xxx, threeX).Add(y2, p192.B) // x^3 - 3*x + B
	y.ModSqrt(y2, p192.P)               // find a square root

	if s[0] == '-' && y.Bit(0) == 1 {
		y.Neg(y)
	}

	return
}
