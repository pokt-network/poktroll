package rings

import (
	"bytes"
	"crypto/sha256"
	"testing"

	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	ring "github.com/pokt-network/ring-go"
)

func TestRingPointsContain_True(t *testing.T) {
	curve := ring_secp256k1.NewCurve()

	// Build a ring with random members; put the signer at index 0.
	priv := curve.NewRandomScalar()
	const size = 8
	r, err := ring.NewKeyRing(curve, size, priv, 0)
	if err != nil {
		t.Fatalf("NewKeyRing: %v", err)
	}

	msg := sha256.Sum256([]byte("ring-points-contain:ok"))
	sig, err := r.Sign(msg, priv)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	// Build the expected map[[32]byte]Point from the ring’s public keys.
	points := sig.PublicKeysRef()
	m := make(map[[32]byte]ringtypes.Point, len(points))
	for _, p := range points {
		m[pointKey32(p)] = p
	}

	if !ringPointsContain(m, sig) {
		t.Fatalf("expected ringPointsContain to return true")
	}
}

func TestRingPointsContain_FalseWhenMissingKey(t *testing.T) {
	curve := ring_secp256k1.NewCurve()

	priv := curve.NewRandomScalar()
	const size = 4
	r, err := ring.NewKeyRing(curve, size, priv, 0)
	if err != nil {
		t.Fatalf("NewKeyRing: %v", err)
	}

	msg := sha256.Sum256([]byte("ring-points-contain:missing"))
	sig, err := r.Sign(msg, priv)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	points := sig.PublicKeysRef()
	m := make(map[[32]byte]ringtypes.Point, len(points))
	for _, p := range points {
		m[pointKey32(p)] = p
	}
	// Remove one member to force a miss.
	delete(m, pointKey32(points[1]))

	if ringPointsContain(m, sig) {
		t.Fatalf("expected ringPointsContain to return false when a key is missing")
	}
}

func TestPointKey32_StableAndDistinct(t *testing.T) {
	curve := ring_secp256k1.NewCurve()

	// Two different points
	p1 := curve.ScalarBaseMul(curve.NewRandomScalar())
	p2 := curve.ScalarBaseMul(curve.NewRandomScalar())

	k11 := pointKey32(p1)
	k12 := pointKey32(p1) // same point, should match
	k2 := pointKey32(p2)  // different point, should differ

	if !bytes.Equal(k11[:], k12[:]) {
		t.Fatalf("pointKey32 not stable for the same point")
	}
	if bytes.Equal(k11[:], k2[:]) {
		t.Fatalf("pointKey32 not distinct for different points")
	}
}
