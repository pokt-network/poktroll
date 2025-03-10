package session

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/pokt-network/poktroll/pkg/relayer"
)

const (
	bufferSize = 1 * 1024 * 1024 // 1MB
	lengthSize = 4               // Represent the size of the relays as uint32
)

var _ relayer.RelayStore = (*relayStore)(nil)

// relayStore is an implementation of the RelayStore interface that stores all
// the mined relays corresponding to a single session in a file.
// It is optimized for fast writes by:
// - Using a fixed-size buffer to reduce the number of system calls.
// - Using a single file for all the relays to reduce file system overhead.
// - Having an always open file to avoid the overhead of opening and closing the file.
// - Storing the offsets of the relays in memory to avoid seeking the file.
type relayStore struct {
	// relayStorePath is the path to the file where the relays are stored.
	relayStorePath string

	// relaysFile is the file where the relays are stored.
	// To avoid the overhead of opening and closing the file, it is always open
	// and only closed prior to the relay store deletion.
	relaysFile *os.File

	// bufferedWriter is a buffered writer that writes to the relaysFile.
	bufferedWriter *bufio.Writer

	// offsets is a map from the relay hash to the offset in the relaysFile where
	// the relay is stored.
	// This map is used to avoid seeking the file when reading a relay.
	offsets map[string]int64

	// mu is a mutex to protect the relay store from concurrent writes.
	mu sync.Mutex
}

func NewRelayStore(relayStorePath string) (relayer.RelayStore, error) {
	relaysFile, err := os.OpenFile(relayStorePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	// Initialize the buffered bufferedWriter with a fixed-size buffer.
	bufferedWriter := bufio.NewWriterSize(relaysFile, bufferSize)

	return &relayStore{
		relaysFile:     relaysFile,
		relayStorePath: relayStorePath,
		bufferedWriter: bufferedWriter,
		offsets:        make(map[string]int64, 0),
	}, nil
}

// Write writes a relay to the relay store.
func (rs *relayStore) Write(relayHash, relayBz []byte) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// The length of the relay value is stored as a uint32.
	// The total space required to write the relay is the sum of the length of the relay value
	relaySize := uint32(len(relayBz))
	requiredSpace := lengthSize + relaySize

	// Get the current offset in the file to store the relay.
	offset, err := rs.relaysFile.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}

	// Default to file writer if the required space is greater than the buffer size.
	var writer io.Writer = rs.relaysFile

	// If the relay fits in the buffer then use the buffered writer to write the relay.
	if requiredSpace <= bufferSize {
		// If the buffered writer does not have enough space to write the relay, flush it.
		// This may happen if the buffer is full or if the relay is larger than the
		// remaining space in the buffer.
		// Due to the check above, the relay will fit in the buffer after the flush.
		if rs.bufferedWriter.Available() < int(requiredSpace) {
			if err := rs.bufferedWriter.Flush(); err != nil {
				return err
			}
		}

		// Use the buffered writer to write the relay.
		writer = rs.bufferedWriter

		// Update the offset to account for the buffered data that has not been written to the file.
		offset += int64(rs.bufferedWriter.Buffered())
	}

	// Store the offset of the relay in memory.
	// TODO_MAINNET(@red-0ne): Ensure that the store can recover from a restart by
	// persisting the offsets to disk or regenerating it from the relays file.
	// Consider storing the relays hashes in the file too for faster recovery.
	rs.offsets[string(relayHash)] = offset

	// Write the relay size to the file to determine the length of the relay when reading it.
	if err := binary.Write(writer, binary.LittleEndian, relaySize); err != nil {
		return err
	}

	// Write the relay to the file.
	if _, err := writer.Write(relayBz); err != nil {
		return err
	}

	return nil
}

// Get reads a relay from the relay store given its hash.
func (rs *relayStore) Get(relayHash []byte) ([]byte, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Flush the buffered writer to ensure that all the relays are written to the
	// file and the whole relay is read from the file.
	if err := rs.bufferedWriter.Flush(); err != nil {
		return nil, err
	}

	// Get the offset of the relay in the file.
	offset, ok := rs.offsets[string(relayHash)]
	if !ok {
		return nil, fmt.Errorf("relay file offset not found for relay hash %x", relayHash)
	}

	// Seek to the offset of the relay in the file.
	if _, err := rs.relaysFile.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	// Read the length of the relay which is stored as a uint32.
	var length uint32
	if err := binary.Read(rs.relaysFile, binary.LittleEndian, &length); err != nil {
		return nil, err
	}

	// Read the relay bytes.
	relayBz := make([]byte, length)
	if _, err := rs.relaysFile.Read(relayBz); err != nil {
		return nil, err
	}

	return relayBz, nil
}

// Delete deletes the relay store file.
// This method should be called when the relay store is no longer needed.
func (rs *relayStore) Delete() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Close the relay store file.
	if err := rs.relaysFile.Close(); err != nil {
		return err
	}

	// Delete the relay store file.
	// TODO_CONSIDERATION: Consider removing the parent directories if they are empty
	// after deleting the relay store file.
	if err := os.Remove(rs.relayStorePath); err != nil {
		return err
	}

	return nil
}
