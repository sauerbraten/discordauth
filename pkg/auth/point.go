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
