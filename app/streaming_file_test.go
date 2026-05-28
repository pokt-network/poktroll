package app

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

type fakeOpts map[string]any

func (f fakeOpts) Get(k string) any { return f[k] }

// newTestStreamer constructs a FileStreamer with stop-on-err disabled so
// tests can inspect returned errors without panicking. Set `hasListeners`
// explicitly per test.
func newTestStreamer(t *testing.T, dir string, hasListeners bool, extra fakeOpts) *FileStreamer {
	t.Helper()
	opts := fakeOpts{
		"streaming.file.write-dir":        dir,
		"streaming.file.fsync":            false,
		"streaming.file.stop-node-on-err": false,
	}
	for k, v := range extra {
		opts[k] = v
	}
	s, err := NewFileStreamer(opts)
	require.NoError(t, err)
	require.NotNil(t, s)
	s.hasListeners = hasListeners
	return s
}

// ── construction ──────────────────────────────────────────────────────────

func TestFileStreamer_NoOpWhenUnconfigured(t *testing.T) {
	s, err := NewFileStreamer(fakeOpts{})
	require.NoError(t, err)
	require.Nil(t, s)
}

func TestFileStreamer_RejectsPrefixWithPathTraversal(t *testing.T) {
	for _, bad := range []string{"../etc/", "foo/bar", "..\\baz"} {
		_, err := NewFileStreamer(fakeOpts{
			"streaming.file.write-dir": t.TempDir(),
			"streaming.file.prefix":    bad,
		})
		require.Error(t, err, "prefix=%q must be rejected", bad)
	}
}

func TestFileStreamer_RejectsUncreatableWriteDir(t *testing.T) {
	// Use a path under a read-only parent → MkdirAll fails.
	parent := t.TempDir()
	require.NoError(t, os.Chmod(parent, 0o500))
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

	_, err := NewFileStreamer(fakeOpts{
		"streaming.file.write-dir": filepath.Join(parent, "subdir"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "write-dir")
}

func TestFileStreamer_SweepStalePartialsOnStartup(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "block-5-meta.partial"), []byte("stale"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "block-5-data.partial"), []byte("stale"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "block-5-meta"), []byte("keep"), 0o644))

	_, err := NewFileStreamer(fakeOpts{"streaming.file.write-dir": dir})
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "block-5-meta.partial"))
	require.True(t, os.IsNotExist(err), "stale meta.partial must be removed")
	_, err = os.Stat(filepath.Join(dir, "block-5-data.partial"))
	require.True(t, os.IsNotExist(err), "stale data.partial must be removed")
	_, err = os.Stat(filepath.Join(dir, "block-5-meta"))
	require.NoError(t, err, "final file must be preserved")
}

func TestFileStreamer_SweepRespectsPrefix(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shard0-block-5-meta.partial"), []byte("stale"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "other-block-5-meta.partial"), []byte("keep"), 0o644))

	_, err := NewFileStreamer(fakeOpts{
		"streaming.file.write-dir": dir,
		"streaming.file.prefix":    "shard0-",
	})
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "shard0-block-5-meta.partial"))
	require.True(t, os.IsNotExist(err), "matching prefix swept")
	_, err = os.Stat(filepath.Join(dir, "other-block-5-meta.partial"))
	require.NoError(t, err, "non-matching prefix preserved")
}

// ── happy paths ───────────────────────────────────────────────────────────

func TestFileStreamer_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := newTestStreamer(t, dir, true, nil)

	ctx := context.Background()
	req := abci.RequestFinalizeBlock{Height: 7, Hash: []byte("hashhashhashhash")}
	res := abci.ResponseFinalizeBlock{AppHash: []byte("apphash")}
	cmt := abci.ResponseCommit{RetainHeight: 7}
	kvs := []*storetypes.StoreKVPair{
		{StoreKey: "bank", Key: []byte("k1"), Value: []byte("v1")},
		{StoreKey: "bank", Key: []byte("k2"), Delete: true},
	}

	require.NoError(t, s.ListenFinalizeBlock(ctx, req, res))
	require.NoError(t, s.ListenCommit(ctx, cmt, kvs))

	metaPath := filepath.Join(dir, "block-7-meta")
	dataPath := filepath.Join(dir, "block-7-data")
	requireFileNonEmpty(t, metaPath)
	requireFileNonEmpty(t, dataPath)
	requireNotExist(t, metaPath+".partial")
	requireNotExist(t, dataPath+".partial")

	metaMsgs := readVarintMessages(t, metaPath, 3)
	gotReq := &abci.RequestFinalizeBlock{}
	require.NoError(t, proto.Unmarshal(metaMsgs[0], gotReq))
	require.Equal(t, int64(7), gotReq.Height)
	gotRes := &abci.ResponseFinalizeBlock{}
	require.NoError(t, proto.Unmarshal(metaMsgs[1], gotRes))
	require.Equal(t, []byte("apphash"), gotRes.AppHash)
	gotCmt := &abci.ResponseCommit{}
	require.NoError(t, proto.Unmarshal(metaMsgs[2], gotCmt))
	require.Equal(t, int64(7), gotCmt.RetainHeight)

	dataMsgs := readVarintMessages(t, dataPath, 2)
	for i, kv := range kvs {
		got := &storetypes.StoreKVPair{}
		require.NoError(t, proto.Unmarshal(dataMsgs[i], got))
		require.Equal(t, kv.StoreKey, got.StoreKey)
		require.Equal(t, kv.Key, got.Key)
		require.Equal(t, kv.Value, got.Value)
		require.Equal(t, kv.Delete, got.Delete)
	}
}

func TestFileStreamer_MultipleBlocksContiguous(t *testing.T) {
	dir := t.TempDir()
	s := newTestStreamer(t, dir, true, nil)
	ctx := context.Background()
	for h := int64(100); h <= 105; h++ {
		require.NoError(t, s.ListenFinalizeBlock(ctx, abci.RequestFinalizeBlock{Height: h}, abci.ResponseFinalizeBlock{}))
		require.NoError(t, s.ListenCommit(ctx, abci.ResponseCommit{}, nil))
	}
	for h := int64(100); h <= 105; h++ {
		requireFileExists(t, filepath.Join(dir, "block-"+fmt.Sprintf("%d", h)+"-meta"))
		requireFileExists(t, filepath.Join(dir, "block-"+fmt.Sprintf("%d", h)+"-data"))
	}
}

func TestFileStreamer_MetaDisabledWritesOnlyData(t *testing.T) {
	dir := t.TempDir()
	s := newTestStreamer(t, dir, true, fakeOpts{
		"streaming.file.output-metadata": false,
	})
	ctx := context.Background()
	require.NoError(t, s.ListenFinalizeBlock(ctx, abci.RequestFinalizeBlock{Height: 3}, abci.ResponseFinalizeBlock{}))
	require.NoError(t, s.ListenCommit(ctx, abci.ResponseCommit{}, nil))

	requireNotExist(t, filepath.Join(dir, "block-3-meta"))
	requireFileExists(t, filepath.Join(dir, "block-3-data"))
}

func TestFileStreamer_NoListenersSkipsDataFile(t *testing.T) {
	dir := t.TempDir()
	s := newTestStreamer(t, dir, false, nil)
	ctx := context.Background()
	require.NoError(t, s.ListenFinalizeBlock(ctx, abci.RequestFinalizeBlock{Height: 9}, abci.ResponseFinalizeBlock{}))
	require.NoError(t, s.ListenCommit(ctx, abci.ResponseCommit{}, nil))

	requireFileExists(t, filepath.Join(dir, "block-9-meta"))
	requireNotExist(t, filepath.Join(dir, "block-9-data"))
}

func TestFileStreamer_EmptyChangeSetStillWritesDataFile(t *testing.T) {
	dir := t.TempDir()
	s := newTestStreamer(t, dir, true, nil) // hasListeners=true but empty changeSet
	ctx := context.Background()
	require.NoError(t, s.ListenFinalizeBlock(ctx, abci.RequestFinalizeBlock{Height: 12}, abci.ResponseFinalizeBlock{}))
	require.NoError(t, s.ListenCommit(ctx, abci.ResponseCommit{}, nil))

	// Block existed and was empty → 0-byte but present.
	st, err := os.Stat(filepath.Join(dir, "block-12-data"))
	require.NoError(t, err)
	require.EqualValues(t, 0, st.Size(), "empty change set produces 0-byte data file")
}

func TestFileStreamer_PrefixApplied(t *testing.T) {
	dir := t.TempDir()
	s := newTestStreamer(t, dir, true, fakeOpts{"streaming.file.prefix": "shard0-"})
	ctx := context.Background()
	require.NoError(t, s.ListenFinalizeBlock(ctx, abci.RequestFinalizeBlock{Height: 42}, abci.ResponseFinalizeBlock{}))
	require.NoError(t, s.ListenCommit(ctx, abci.ResponseCommit{}, nil))

	requireFileExists(t, filepath.Join(dir, "shard0-block-42-meta"))
	requireFileExists(t, filepath.Join(dir, "shard0-block-42-data"))
}

// ── completion-marker semantics ───────────────────────────────────────────

func TestFileStreamer_MetaIsCompletionMarker(t *testing.T) {
	// On a successful block, data must be renamed to its final name BEFORE
	// meta is. We verify by inspecting that meta exists last in modtime order.
	dir := t.TempDir()
	s := newTestStreamer(t, dir, true, nil)
	ctx := context.Background()
	require.NoError(t, s.ListenFinalizeBlock(ctx, abci.RequestFinalizeBlock{Height: 50}, abci.ResponseFinalizeBlock{}))
	require.NoError(t, s.ListenCommit(ctx, abci.ResponseCommit{}, nil))

	metaInfo, err := os.Stat(filepath.Join(dir, "block-50-meta"))
	require.NoError(t, err)
	dataInfo, err := os.Stat(filepath.Join(dir, "block-50-data"))
	require.NoError(t, err)
	require.False(t, metaInfo.ModTime().Before(dataInfo.ModTime()),
		"meta must be created after data (completion marker)")
}

// ── sad paths ─────────────────────────────────────────────────────────────

func TestFileStreamer_CommitWithoutFinalizeReturnsError(t *testing.T) {
	dir := t.TempDir()
	s := newTestStreamer(t, dir, true, nil)
	err := s.ListenCommit(context.Background(), abci.ResponseCommit{}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ListenCommit called without successful ListenFinalizeBlock")
}

func TestFileStreamer_CommitWithoutFinalize_PanicsWhenStopOnErr(t *testing.T) {
	dir := t.TempDir()
	s, err := NewFileStreamer(fakeOpts{
		"streaming.file.write-dir":        dir,
		"streaming.file.stop-node-on-err": true,
		"streaming.file.fsync":            false,
	})
	require.NoError(t, err)
	s.hasListeners = true
	require.Panics(t, func() {
		_ = s.ListenCommit(context.Background(), abci.ResponseCommit{}, nil)
	})
}

func TestFileStreamer_CanceledContext(t *testing.T) {
	dir := t.TempDir()
	s := newTestStreamer(t, dir, true, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, s.ListenFinalizeBlock(ctx, abci.RequestFinalizeBlock{Height: 1}, abci.ResponseFinalizeBlock{}), context.Canceled)
	require.ErrorIs(t, s.ListenCommit(ctx, abci.ResponseCommit{}, nil), context.Canceled)
}

func TestFileStreamer_RenameFailure_LeavesNoOrphan(t *testing.T) {
	// After FinalizeBlock writes meta.partial, revoke write perms on the dir
	// so the data file open fails. Verify no final files exist after the error.
	dir := t.TempDir()
	s := newTestStreamer(t, dir, true, nil)
	ctx := context.Background()
	require.NoError(t, s.ListenFinalizeBlock(ctx, abci.RequestFinalizeBlock{Height: 60}, abci.ResponseFinalizeBlock{}))

	require.NoError(t, os.Chmod(dir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := s.ListenCommit(ctx, abci.ResponseCommit{}, nil)
	require.Error(t, err)

	// Restore perms so we can inspect.
	require.NoError(t, os.Chmod(dir, 0o755))
	requireNotExist(t, filepath.Join(dir, "block-60-meta"))
	requireNotExist(t, filepath.Join(dir, "block-60-data"))
}

// ── selectExposedKeys ─────────────────────────────────────────────────────

func TestSelectExposedKeys(t *testing.T) {
	all := map[string]*storetypes.KVStoreKey{
		"bank":    storetypes.NewKVStoreKey("bank"),
		"staking": storetypes.NewKVStoreKey("staking"),
		"gov":     storetypes.NewKVStoreKey("gov"),
	}
	got, err := selectExposedKeys(nil, all)
	require.NoError(t, err)
	require.Empty(t, got, "nil → none")

	got, err = selectExposedKeys([]string{}, all)
	require.NoError(t, err)
	require.Empty(t, got, "[] → none")

	got, err = selectExposedKeys([]string{"*"}, all)
	require.NoError(t, err)
	require.Len(t, got, 3, `["*"] → all`)

	got, err = selectExposedKeys([]string{"bank", "gov"}, all)
	require.NoError(t, err)
	require.Len(t, got, 2, "subset")

	_, err = selectExposedKeys([]string{"bank", "unknown"}, all)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown")
	require.Contains(t, err.Error(), "bank")
	require.Contains(t, err.Error(), "staking")
}

// ── helpers ───────────────────────────────────────────────────────────────

func requireFileNonEmpty(t *testing.T, p string) {
	t.Helper()
	st, err := os.Stat(p)
	require.NoError(t, err, "stat %s", p)
	require.Greater(t, st.Size(), int64(0), "%s is empty", p)
}

func requireFileExists(t *testing.T, p string) {
	t.Helper()
	_, err := os.Stat(p)
	require.NoError(t, err, "expected %s to exist", p)
}

func requireNotExist(t *testing.T, p string) {
	t.Helper()
	_, err := os.Stat(p)
	require.True(t, os.IsNotExist(err), "%s should not exist (err=%v)", p, err)
}

func readVarintMessages(t *testing.T, path string, n int) [][]byte {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	r := bytes.NewReader(b)
	out := make([][]byte, 0, n)
	for i := 0; i < n; i++ {
		size, err := binary.ReadUvarint(r)
		require.NoError(t, err, "msg %d varint", i)
		payload := make([]byte, size)
		_, err = io.ReadFull(r, payload)
		require.NoError(t, err, "msg %d payload", i)
		out = append(out, payload)
	}
	require.Equal(t, 0, r.Len(), "trailing bytes after %d messages", n)
	return out
}
