package store

import (
	"fmt"
	"path"
	"testing"

	cosmostypes "cosmossdk.io/store/types"
	"github.com/stretchr/testify/require"
)

func TestNewMultiStore(t *testing.T) {
	// Will be deleted once the test completes.
	tempDBPath := path.Join(t.TempDir(), fmt.Sprintf("%s.db", t.Name()))

	// Mount a single "test" store.
	testStoreKey := cosmostypes.NewKVStoreKey("test")
	storeTypesByStoreKey := map[cosmostypes.StoreKey]cosmostypes.StoreType{
		testStoreKey: cosmostypes.StoreTypeDB,
	}

	multiStore, closeDB, err := NewMultiStore(tempDBPath, storeTypesByStoreKey)
	require.NoError(t, err)
	require.NotNil(t, multiStore)

	testStore := multiStore.GetKVStore(testStoreKey)
	require.NotNil(t, testStore)

	t.Run("basic operations", func(t *testing.T) {
		// The store should initially be empty.
		value1 := testStore.Get([]byte("key1"))
		require.Equal(t, []byte(nil), value1)

		// Store a value.
		testStore.Set([]byte("key1"), []byte("value1"))

		// The store should now have a value.
		gotValue1 := testStore.Get([]byte("key1"))
		require.Equal(t, []byte("value1"), gotValue1)

		// Store another value.
		testStore.Set([]byte("key2"), []byte("value2"))

		// The store should now have two values.
		gotValue1 = testStore.Get([]byte("key1"))
		require.Equal(t, []byte("value1"), gotValue1)
		gotValue2 := testStore.Get([]byte("key2"))
		require.Equal(t, []byte("value2"), gotValue2)
	})

	t.Run("load from persistence", func(t *testing.T) {
		// Close the existing multi-store/DB.
		err := closeDB()
		require.NoError(t, err)

		// Load the persisted multi-store.
		multiStore, closeDB, err = NewMultiStore(tempDBPath, storeTypesByStoreKey)
		require.NoError(t, err)
		require.NotNil(t, multiStore)

		persistedTestStore := multiStore.GetKVStore(testStoreKey)
		require.NotNil(t, persistedTestStore)

		// The store should have two values that were stored in the previous test.
		gotValue1 := persistedTestStore.Get([]byte("key1"))
		require.Equal(t, []byte("value1"), gotValue1)
		gotValue2 := persistedTestStore.Get([]byte("key2"))
		require.Equal(t, []byte("value2"), gotValue2)
	})
}
