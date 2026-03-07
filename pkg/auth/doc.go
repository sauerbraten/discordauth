// package auth implements Sauerbraten's player authentication mechanism.
//
// The mechanism relies on the associativity of scalar multiplication on elliptic curves: private keys are random (big)
// scalars, and the corresponding public key is created by multiplying the curve base point with the private key. (This
// means the public key is another point on the curve.)
//
// To check for possession of the private key belonging to a public key known to the server, the base point is
// multiplied with another random, big scalar (the "secret") and the resulting point is sent to the user as "challenge"
// (challenge = secret * base).
// The client multiplies the challenge curve point with his private key (a scalar), and sends the X coordinate of the
// resulting point back to the server (response = challenge * priv = secret * base * priv).
// The server instead multiplies the user's public key with the secret scalar. Since pub = base * priv,
// pub * secret = (base * priv) * secret = (secret * base) * priv = challenge * priv = response.
//
// Because of the curve's symmetry, there are exactly two points on the curve at any given X. For simplicity (and maybe
// performance), the server is satisfied when the client responds with the correct X.
package auth
