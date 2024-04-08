package rings

import (
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	ring "github.com/noot/ring-go"
)

// newRingFromPoints creates a new ring from points (i.e. public keys) on the secp256k1 curve
func newRingFromPoints(points []ringtypes.Point) (*ring.Ring, error) {
	return ring.NewFixedKeyRingFromPublicKeys(ring_secp256k1.NewCurve(), points)
}

// pointsFromPublicKeys returns the secp256k1 points for the given public keys.
// It returns an error if any of the keys is not on the secp256k1 curve.
func pointsFromPublicKeys(
	publicKeys ...cryptotypes.PubKey,
) (points []ringtypes.Point, err error) {
	curve := ring_secp256k1.NewCurve()

	for _, key := range publicKeys {
		// Check if the key is a secp256k1 public key
		if _, ok := key.(*secp256k1.PubKey); !ok {
			return nil, ErrRingsNotSecp256k1Curve.Wrapf("got %T", key)
		}

		// Convert the public key to the point on the secp256k1 curve
		point, err := curve.DecodeToPoint(key.Bytes())
		if err != nil {
			return nil, err
		}

		// Insert the point into the slice of points
		points = append(points, point)
	}

	return points, nil
}
