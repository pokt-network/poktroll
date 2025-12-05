//go:build test

package keys

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

func TestFileKeyProvider_NewWithNonExistentDir(t *testing.T) {
	logger := polyzero.NewLogger()
	tempDir := t.TempDir()
	keysDir := filepath.Join(tempDir, "new_keys_dir")

	// Directory doesn't exist - should be created
	provider, err := NewFileKeyProvider(logger, keysDir)
	require.NoError(t, err)
	require.NotNil(t, provider)
	defer provider.Close()

	// Verify directory was created
	info, err := os.Stat(keysDir)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestFileKeyProvider_NewWithExistingDir(t *testing.T) {
	logger := polyzero.NewLogger()
	tempDir := t.TempDir()

	provider, err := NewFileKeyProvider(logger, tempDir)
	require.NoError(t, err)
	require.NotNil(t, provider)
	defer provider.Close()

	require.Equal(t, "file:"+tempDir, provider.Name())
}

func TestFileKeyProvider_NewWithFile(t *testing.T) {
	logger := polyzero.NewLogger()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "not_a_dir")

	// Create a file instead of directory
	err := os.WriteFile(filePath, []byte("test"), 0644)
	require.NoError(t, err)

	// Should fail - not a directory
	_, err = NewFileKeyProvider(logger, filePath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a directory")
}

func TestFileKeyProvider_LoadKeys_EmptyDir(t *testing.T) {
	logger := polyzero.NewLogger()
	tempDir := t.TempDir()

	provider, err := NewFileKeyProvider(logger, tempDir)
	require.NoError(t, err)
	defer provider.Close()

	ctx := context.Background()
	keys, err := provider.LoadKeys(ctx)
	require.NoError(t, err)
	require.Empty(t, keys)
}

func TestFileKeyProvider_LoadKeys_ValidKeyFile(t *testing.T) {
	logger := polyzero.NewLogger()
	tempDir := t.TempDir()

	// Create a valid key file
	// This is a test private key - never use in production!
	keyContent := `
operator_address: pokt1test123456789
private_key_hex: 0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
`
	keyPath := filepath.Join(tempDir, "test_key.yaml")
	err := os.WriteFile(keyPath, []byte(keyContent), 0600)
	require.NoError(t, err)

	provider, err := NewFileKeyProvider(logger, tempDir)
	require.NoError(t, err)
	defer provider.Close()

	ctx := context.Background()
	keys, err := provider.LoadKeys(ctx)
	require.NoError(t, err)
	require.Len(t, keys, 1)

	// Key should be indexed by operator address
	key, exists := keys["pokt1test123456789"]
	require.True(t, exists)
	require.NotNil(t, key)
}

func TestFileKeyProvider_LoadKeys_InvalidKeyFile(t *testing.T) {
	logger := polyzero.NewLogger()
	tempDir := t.TempDir()

	// Create an invalid key file (missing fields)
	keyContent := `
operator_address: pokt1test
`
	keyPath := filepath.Join(tempDir, "invalid_key.yaml")
	err := os.WriteFile(keyPath, []byte(keyContent), 0600)
	require.NoError(t, err)

	provider, err := NewFileKeyProvider(logger, tempDir)
	require.NoError(t, err)
	defer provider.Close()

	ctx := context.Background()
	keys, err := provider.LoadKeys(ctx)
	require.NoError(t, err) // Should not error, just skip invalid file
	require.Empty(t, keys)
}

func TestFileKeyProvider_LoadKeys_InvalidHexKey(t *testing.T) {
	logger := polyzero.NewLogger()
	tempDir := t.TempDir()

	// Create a key file with invalid hex
	keyContent := `
operator_address: pokt1test
private_key_hex: not_valid_hex
`
	keyPath := filepath.Join(tempDir, "bad_hex.yaml")
	err := os.WriteFile(keyPath, []byte(keyContent), 0600)
	require.NoError(t, err)

	provider, err := NewFileKeyProvider(logger, tempDir)
	require.NoError(t, err)
	defer provider.Close()

	ctx := context.Background()
	keys, err := provider.LoadKeys(ctx)
	require.NoError(t, err)
	require.Empty(t, keys) // Invalid file skipped
}

func TestFileKeyProvider_LoadKeys_WrongKeyLength(t *testing.T) {
	logger := polyzero.NewLogger()
	tempDir := t.TempDir()

	// Create a key file with wrong key length (16 bytes instead of 32)
	keyContent := `
operator_address: pokt1test
private_key_hex: 0123456789abcdef0123456789abcdef
`
	keyPath := filepath.Join(tempDir, "short_key.yaml")
	err := os.WriteFile(keyPath, []byte(keyContent), 0600)
	require.NoError(t, err)

	provider, err := NewFileKeyProvider(logger, tempDir)
	require.NoError(t, err)
	defer provider.Close()

	ctx := context.Background()
	keys, err := provider.LoadKeys(ctx)
	require.NoError(t, err)
	require.Empty(t, keys) // Invalid file skipped
}

func TestFileKeyProvider_LoadKeys_SkipsNonYamlFiles(t *testing.T) {
	logger := polyzero.NewLogger()
	tempDir := t.TempDir()

	// Create a non-yaml file
	err := os.WriteFile(filepath.Join(tempDir, "readme.txt"), []byte("not a key"), 0644)
	require.NoError(t, err)

	// Create a subdirectory (should be skipped)
	err = os.MkdirAll(filepath.Join(tempDir, "subdir"), 0755)
	require.NoError(t, err)

	provider, err := NewFileKeyProvider(logger, tempDir)
	require.NoError(t, err)
	defer provider.Close()

	ctx := context.Background()
	keys, err := provider.LoadKeys(ctx)
	require.NoError(t, err)
	require.Empty(t, keys)
}

func TestFileKeyProvider_SupportsHotReload(t *testing.T) {
	logger := polyzero.NewLogger()
	tempDir := t.TempDir()

	provider, err := NewFileKeyProvider(logger, tempDir)
	require.NoError(t, err)
	defer provider.Close()

	require.True(t, provider.SupportsHotReload())
}

func TestFileKeyProvider_Close(t *testing.T) {
	logger := polyzero.NewLogger()
	tempDir := t.TempDir()

	provider, err := NewFileKeyProvider(logger, tempDir)
	require.NoError(t, err)

	// Close should not error
	err = provider.Close()
	require.NoError(t, err)

	// Double close should be safe
	err = provider.Close()
	require.NoError(t, err)
}

func TestFileKeyProvider_MultipleKeyFiles(t *testing.T) {
	logger := polyzero.NewLogger()
	tempDir := t.TempDir()

	// Create multiple key files
	keys := []struct {
		filename string
		address  string
		key      string
	}{
		{
			"key1.yaml",
			"pokt1operator1",
			"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		{
			"key2.yml",
			"pokt1operator2",
			"fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		},
		{
			"key3.json",
			"pokt1operator3",
			"1111111111111111111111111111111111111111111111111111111111111111",
		},
	}

	for _, k := range keys {
		content := `
operator_address: ` + k.address + `
private_key_hex: ` + k.key + `
`
		err := os.WriteFile(filepath.Join(tempDir, k.filename), []byte(content), 0600)
		require.NoError(t, err)
	}

	provider, err := NewFileKeyProvider(logger, tempDir)
	require.NoError(t, err)
	defer provider.Close()

	ctx := context.Background()
	loadedKeys, err := provider.LoadKeys(ctx)
	require.NoError(t, err)
	require.Len(t, loadedKeys, 3)

	// All addresses should be present
	for _, k := range keys {
		_, exists := loadedKeys[k.address]
		require.True(t, exists, "key for %s should exist", k.address)
	}
}
