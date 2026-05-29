package app

// In-process ABCI streaming listener writing per-block protobuf files.
// Workaround for the cosmos-sdk `[streaming.<svc>] plugin = "..."` path
// requiring an external go-plugin binary that poktroll does not ship.
//
// Per-block (height H), files are written via .partial + atomic rename:
//   {prefix}block-{H}-meta — varint-prefixed: RequestFinalizeBlock,
//                            ResponseFinalizeBlock, ResponseCommit.
//   {prefix}block-{H}-data — varint-prefixed: StoreKVPair per Set/Delete.
//
// The meta file is the completion marker: data is renamed to its final
// name before meta is. A crash mid-block leaves at most a `.partial`
// (and possibly a final data file with no meta), both of which downstream
// consumers must treat as incomplete.
//
// No-op unless `[streaming.file] write-dir` is set in app.toml.

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/spf13/cast"
)

const (
	streamingFileWriteDir   = "streaming.file.write-dir"
	streamingFileKeys       = "streaming.file.keys"
	streamingFilePrefix     = "streaming.file.prefix"
	streamingFileFsync      = "streaming.file.fsync"
	streamingFileStopOnErr  = "streaming.file.stop-node-on-err"
	streamingFileOutputMeta = "streaming.file.output-metadata"
)

type FileStreamer struct {
	mu         sync.Mutex
	writeDir   string
	prefix     string
	fsync      bool
	outputMeta bool
	stopOnErr  bool
	// hasListeners indicates whether any KV store is being listened on. When
	// false the data file is skipped entirely (avoids 0-byte inode churn in
	// meta-only mode).
	hasListeners bool

	// Per-block state, set in ListenFinalizeBlock, consumed in ListenCommit.
	// `prepared` gates the data write so a failed FinalizeBlock can't leave
	// an orphan data file when stop-node-on-err is false.
	curHeight          int64
	curMetaFile        *os.File
	curMetaPartialPath string
	curMetaFinalPath   string
	prepared           bool
}

// NewFileStreamer returns (nil, nil) when `streaming.file.write-dir` is unset.
// `hasListeners` MUST be set by the caller via RegisterInProcessFileStreamer
// based on whether any store keys are being exposed.
func NewFileStreamer(appOpts servertypes.AppOptions) (*FileStreamer, error) {
	writeDir := cast.ToString(appOpts.Get(streamingFileWriteDir))
	if writeDir == "" {
		return nil, nil
	}
	if err := os.MkdirAll(writeDir, 0o755); err != nil {
		return nil, fmt.Errorf("streaming.file: cannot create write-dir %q: %w", writeDir, err)
	}
	prefix := cast.ToString(appOpts.Get(streamingFilePrefix))
	if strings.ContainsAny(prefix, "/\\") || strings.Contains(prefix, "..") {
		return nil, fmt.Errorf("streaming.file.prefix must not contain path separators or '..': %q", prefix)
	}
	if err := sweepStalePartials(writeDir, prefix); err != nil {
		return nil, fmt.Errorf("streaming.file: sweep stale partials: %w", err)
	}
	fsync := true
	if raw := appOpts.Get(streamingFileFsync); raw != nil {
		fsync = cast.ToBool(raw)
	}
	outputMeta := true
	if raw := appOpts.Get(streamingFileOutputMeta); raw != nil {
		outputMeta = cast.ToBool(raw)
	}
	stopOnErr := true
	if raw := appOpts.Get(streamingFileStopOnErr); raw != nil {
		stopOnErr = cast.ToBool(raw)
	}
	return &FileStreamer{
		writeDir:   writeDir,
		prefix:     prefix,
		fsync:      fsync,
		outputMeta: outputMeta,
		stopOnErr:  stopOnErr,
	}, nil
}

// sweepStalePartials removes any leftover `{prefix}block-*.partial` files
// from a prior process crash so they don't accumulate forever.
func sweepStalePartials(writeDir, prefix string) error {
	pattern := filepath.Join(writeDir, prefix+"block-*.partial")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	for _, m := range matches {
		_ = os.Remove(m)
	}
	return nil
}

// fail honors the configured stop-on-err policy. Returning an error from a
// Listen method only causes baseapp to LOG it (SDK v0.53.7 does not consult
// StreamingManager.StopNodeOnErr for in-process listeners — that field is
// only read by the go-plugin gRPC wrapper). When `stop-node-on-err=true`
// the caller wants the node to halt on archival I/O failures, so we panic.
func (f *FileStreamer) fail(err error) error {
	if f.stopOnErr {
		panic(err)
	}
	return err
}

func (f *FileStreamer) ListenFinalizeBlock(ctx context.Context, req abci.RequestFinalizeBlock, res abci.ResponseFinalizeBlock) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	f.curHeight = req.Height
	f.prepared = false

	if !f.outputMeta {
		f.prepared = true
		return nil
	}

	finalPath := f.metaPath(req.Height)
	partialPath := finalPath + ".partial"
	mf, err := os.OpenFile(partialPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return f.fail(fmt.Errorf("streaming.file: open meta %q: %w", partialPath, err))
	}
	if err := writeProto(mf, &req); err != nil {
		mf.Close()
		_ = os.Remove(partialPath)
		return f.fail(fmt.Errorf("streaming.file: write FinalizeBlock req h=%d: %w", req.Height, err))
	}
	if err := writeProto(mf, &res); err != nil {
		mf.Close()
		_ = os.Remove(partialPath)
		return f.fail(fmt.Errorf("streaming.file: write FinalizeBlock res h=%d: %w", req.Height, err))
	}
	f.curMetaFile = mf
	f.curMetaPartialPath = partialPath
	f.curMetaFinalPath = finalPath
	f.prepared = true
	return nil
}

func (f *FileStreamer) ListenCommit(ctx context.Context, res abci.ResponseCommit, changeSet []*storetypes.StoreKVPair) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.prepared {
		return f.fail(fmt.Errorf("streaming.file: ListenCommit called without successful ListenFinalizeBlock (h=%d)", f.curHeight))
	}
	f.prepared = false

	// 1) Append ResponseCommit to meta.partial; fsync; close. Keep .partial.
	if f.outputMeta && f.curMetaFile != nil {
		if err := writeProto(f.curMetaFile, &res); err != nil {
			f.cleanupMeta()
			return f.fail(fmt.Errorf("streaming.file: write Commit res h=%d: %w", f.curHeight, err))
		}
		if f.fsync {
			if err := f.curMetaFile.Sync(); err != nil {
				f.cleanupMeta()
				return f.fail(fmt.Errorf("streaming.file: fsync meta h=%d: %w", f.curHeight, err))
			}
		}
		if err := f.curMetaFile.Close(); err != nil {
			_ = os.Remove(f.curMetaPartialPath)
			f.resetMetaState()
			return f.fail(fmt.Errorf("streaming.file: close meta h=%d: %w", f.curHeight, err))
		}
		f.curMetaFile = nil
	}

	// 2) Data file. Skip entirely when no stores are exposed (meta-only mode);
	//    writing 0-byte files every block creates inode churn for no value.
	var dataFinal string
	if f.hasListeners {
		dataFinal = f.dataPath(f.curHeight)
		dataPartial := dataFinal + ".partial"
		df, err := os.OpenFile(dataPartial, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			if f.outputMeta {
				_ = os.Remove(f.curMetaPartialPath)
				f.resetMetaState()
			}
			return f.fail(fmt.Errorf("streaming.file: open data %q: %w", dataPartial, err))
		}
		for _, kv := range changeSet {
			if err := writeProto(df, kv); err != nil {
				df.Close()
				_ = os.Remove(dataPartial)
				if f.outputMeta {
					_ = os.Remove(f.curMetaPartialPath)
					f.resetMetaState()
				}
				return f.fail(fmt.Errorf("streaming.file: write KV h=%d: %w", f.curHeight, err))
			}
		}
		if f.fsync {
			if err := df.Sync(); err != nil {
				df.Close()
				_ = os.Remove(dataPartial)
				if f.outputMeta {
					_ = os.Remove(f.curMetaPartialPath)
					f.resetMetaState()
				}
				return f.fail(fmt.Errorf("streaming.file: fsync data h=%d: %w", f.curHeight, err))
			}
		}
		if err := df.Close(); err != nil {
			_ = os.Remove(dataPartial)
			if f.outputMeta {
				_ = os.Remove(f.curMetaPartialPath)
				f.resetMetaState()
			}
			return f.fail(fmt.Errorf("streaming.file: close data h=%d: %w", f.curHeight, err))
		}
		// 3) Rename data first.
		if err := os.Rename(dataPartial, dataFinal); err != nil {
			_ = os.Remove(dataPartial)
			if f.outputMeta {
				_ = os.Remove(f.curMetaPartialPath)
				f.resetMetaState()
			}
			return f.fail(fmt.Errorf("streaming.file: rename data h=%d: %w", f.curHeight, err))
		}
	}

	// 4) Rename meta last — it is the completion marker.
	if f.outputMeta && f.curMetaPartialPath != "" {
		if err := os.Rename(f.curMetaPartialPath, f.curMetaFinalPath); err != nil {
			_ = os.Remove(f.curMetaPartialPath)
			// data already exists; downstream will discard it (no meta).
			if dataFinal != "" {
				_ = os.Remove(dataFinal)
			}
			f.resetMetaState()
			return f.fail(fmt.Errorf("streaming.file: rename meta h=%d: %w", f.curHeight, err))
		}
		f.resetMetaState()
	}

	// 5) fsync the write-dir so the rename(s) survive a crash.
	if f.fsync {
		if err := fsyncDir(f.writeDir); err != nil {
			return f.fail(fmt.Errorf("streaming.file: fsync write-dir h=%d: %w", f.curHeight, err))
		}
	}
	return nil
}

func (f *FileStreamer) metaPath(h int64) string {
	return filepath.Join(f.writeDir, fmt.Sprintf("%sblock-%d-meta", f.prefix, h))
}

func (f *FileStreamer) dataPath(h int64) string {
	return filepath.Join(f.writeDir, fmt.Sprintf("%sblock-%d-data", f.prefix, h))
}

func (f *FileStreamer) cleanupMeta() {
	if f.curMetaFile != nil {
		_ = f.curMetaFile.Close()
	}
	if f.curMetaPartialPath != "" {
		_ = os.Remove(f.curMetaPartialPath)
	}
	f.resetMetaState()
}

func (f *FileStreamer) resetMetaState() {
	f.curMetaFile = nil
	f.curMetaPartialPath = ""
	f.curMetaFinalPath = ""
}

// writeProto emits one varint-length-prefixed proto message.
func writeProto(w io.Writer, m proto.Message) error {
	buf, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	var hdr [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(hdr[:], uint64(len(buf)))
	if _, err = w.Write(hdr[:n]); err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}

// fsyncDir flushes the directory entry so a recent rename survives a crash.
// Best-effort on platforms where directory fsync is not supported.
func fsyncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	syncErr := d.Sync()
	closeErr := d.Close()
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}

// RegisterInProcessFileStreamer wires the FileStreamer into the BaseApp.
// `streaming.file.keys`: ["*"] exposes all stores; an explicit list exposes
// only the named ones — unknown names cause startup to FAIL with the full
// list of available stores. Empty or unset = no stores (meta-only mode if
// output-metadata is true).
func RegisterInProcessFileStreamer(
	app *baseapp.BaseApp,
	appOpts servertypes.AppOptions,
	keys map[string]*storetypes.KVStoreKey,
) error {
	streamer, err := NewFileStreamer(appOpts)
	if err != nil {
		return err
	}
	if streamer == nil {
		return nil
	}

	exposed, err := selectExposedKeys(cast.ToStringSlice(appOpts.Get(streamingFileKeys)), keys)
	if err != nil {
		return err
	}
	streamer.hasListeners = len(exposed) > 0
	if streamer.hasListeners {
		app.CommitMultiStore().AddListeners(exposed)
	}

	// Merge with the existing StreamingManager (preserves any other listeners,
	// e.g., the go-plugin gRPC path). NOTE: StopNodeOnErr is intentionally NOT
	// merged from our config — it has no effect on in-process listeners in
	// SDK v0.53.7 (see fail()). We honor stop-on-err inside the streamer.
	mgr := app.StreamingManager()
	mgr.ABCIListeners = append(mgr.ABCIListeners, streamer)
	app.SetStreamingManager(mgr)
	return nil
}

func selectExposedKeys(requested []string, all map[string]*storetypes.KVStoreKey) ([]storetypes.StoreKey, error) {
	for _, r := range requested {
		if r == "*" {
			out := make([]storetypes.StoreKey, 0, len(all))
			for _, v := range all {
				out = append(out, v)
			}
			return out, nil
		}
	}
	var unknown []string
	out := make([]storetypes.StoreKey, 0, len(requested))
	for _, name := range requested {
		if k, ok := all[name]; ok {
			out = append(out, k)
		} else {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		available := make([]string, 0, len(all))
		for k := range all {
			available = append(available, k)
		}
		sort.Strings(available)
		return nil, fmt.Errorf(
			"streaming.file.keys: unknown store(s) %v; available: %v (or [\"*\"] for all)",
			unknown, available,
		)
	}
	return out, nil
}
