package channel_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"pocket/pkg/observable/channel"
)

func TestMapWord_BzToPalindrome(t *testing.T) {
	tests := []struct {
		name    string
		wordBz  []byte
		isValid bool
	}{
		{
			name:    "valid palindrome",
			wordBz:  []byte("rotator"),
			isValid: true,
		},
		{
			name:    "invalid palindrome",
			wordBz:  []byte("spinner"),
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				wordCounter int32
				ctx, cancel = context.WithCancel(context.Background())
			)
			t.Cleanup(cancel)

			// set up source bytes observable
			bzObservable, bzPublishCh := channel.NewObservable[[]byte]()
			bytesToPalindrome := func(wordBz []byte) (palindrome, bool) {
				return newPalindrome(string(wordBz)), false
			}

			// map bytes observable to palindrome observable
			palindromeObservable := channel.Map(ctx, bzObservable, bytesToPalindrome)
			palindromeObserver := palindromeObservable.Subscribe(ctx)

			// publish a word in bytes
			bzPublishCh <- tt.wordBz

			// concurrently consume the palindrome observer's channel
			go func() {
				for word := range palindromeObserver.Ch() {
					atomic.AddInt32(&wordCounter, 1)

					// word.forwards should always match the original word
					require.Equal(t, string(tt.wordBz), word.forwards)

					if tt.isValid {
						require.Equal(t, string(tt.wordBz), word.backwards)
						require.Truef(t, word.IsValid(), "palindrome should be valid")
					} else {
						require.NotEmptyf(t, string(tt.wordBz), word.backwards)
						require.Falsef(t, word.IsValid(), "palindrome should be invalid")
					}
				}
			}()

			// wait a tick for the observer to receive the word
			time.Sleep(time.Millisecond)

			// ensure that the observer received the word
			require.Equal(t, int32(1), atomic.LoadInt32(&wordCounter))
		})
	}
}

// palindrome is a word that is spelled the same forwards and backwards.
type palindrome struct {
	forwards  string
	backwards string
}

func newPalindrome(word string) palindrome {
	return palindrome{
		forwards:  word,
		backwards: reverseString(word),
	}
}

// IsValid returns true if the word actually is a palindrome.
func (p *palindrome) IsValid() bool {
	return p.forwards == (p.backwards)
}

// reverseString reverses a string, character-by-character.
func reverseString(s string) string {
	runes := []rune(s)
	// use i & j as cursors to iteratively swap values on symmetrical indexes
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
