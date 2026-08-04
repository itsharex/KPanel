//go:build linux

package systeminfo

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var storageRoots = map[string]bool{
	"/": true, "/home": true, "/home/docker": true, "/home/web": true,
	"/opt": true, "/root": true, "/srv": true, "/tmp": true,
	"/usr": true, "/var": true, "/var/lib/docker": true, "/var/log": true,
}

type inodeIdentity struct{ device, inode uint64 }

func (c *Collector) collectStorageUsage(ctx context.Context, requested string) (StorageUsage, error) {
	root := filepath.Clean(requested)
	if !filepath.IsAbs(root) || !storageRoots[root] {
		return StorageUsage{}, errors.New("storage path is not an allowed analysis root")
	}
	return collectStorageRoot(ctx, root, c.Now().UTC())
}

func collectStorageRoot(ctx context.Context, root string, collectedAt time.Time) (StorageUsage, error) {
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() {
		return StorageUsage{}, errors.New("storage analysis root is unavailable")
	}
	rootStat, ok := rootInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return StorageUsage{}, errors.New("storage metadata is unavailable")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return StorageUsage{}, err
	}
	result := StorageUsage{Path: root, Items: make([]StorageUsageItem, 0, len(entries)), CollectedAt: collectedAt}
	jobs := make(chan fs.DirEntry)
	items := make(chan StorageUsageItem, len(entries))
	workers := min(4, max(1, runtime.NumCPU()), len(entries))
	perChildLimit := max(2_000, min(25_000, maxStorageEntries/max(1, len(entries))))
	var scanned atomic.Int64
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			for entry := range jobs {
				items <- scanStorageChild(ctx, root, entry, uint64(rootStat.Dev), perChildLimit, &scanned)
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, entry := range entries {
			child := filepath.Join(root, entry.Name())
			if root == "/" && (child == "/proc" || child == "/sys" || child == "/dev" || child == "/run") {
				continue
			}
			jobs <- entry
		}
	}()
	group.Wait()
	close(items)
	for item := range items {
		result.Items = append(result.Items, item)
		result.Scanned += item.Scanned
		result.Truncated = result.Truncated || item.Truncated
	}
	sort.Slice(result.Items, func(i, j int) bool {
		if result.Items[i].SizeBytes != result.Items[j].SizeBytes {
			return result.Items[i].SizeBytes > result.Items[j].SizeBytes
		}
		return result.Items[i].Path < result.Items[j].Path
	})
	if len(result.Items) > maxStorageItems {
		result.Items = result.Items[:maxStorageItems]
		result.Truncated = true
	}
	return result, nil
}

func scanStorageChild(ctx context.Context, root string, entry fs.DirEntry, rootDevice uint64, entryLimit int, scanned *atomic.Int64) StorageUsageItem {
	child := filepath.Join(root, entry.Name())
	item := StorageUsageItem{Path: child, Kind: "file"}
	if entry.IsDir() {
		item.Kind = "directory"
	}
	seen := make(map[inodeIdentity]struct{}, min(entryLimit, 4096))
	walkErr := filepath.WalkDir(child, func(_ string, value fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			item.Truncated = true
			if value != nil && value.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if ctx.Err() != nil || item.Scanned >= entryLimit || scanned.Load() >= maxStorageEntries {
			item.Truncated = true
			return fs.SkipAll
		}
		info, infoErr := value.Info()
		if infoErr != nil {
			item.Truncated = true
			return nil
		}
		stat, statOK := info.Sys().(*syscall.Stat_t)
		if !statOK || uint64(stat.Dev) != rootDevice {
			if value.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if scanned.Add(1) > maxStorageEntries {
			scanned.Add(-1)
			item.Truncated = true
			return fs.SkipAll
		}
		item.Scanned++
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		identity := inodeIdentity{device: uint64(stat.Dev), inode: stat.Ino}
		if _, duplicate := seen[identity]; duplicate {
			return nil
		}
		seen[identity] = struct{}{}
		if stat.Blocks > 0 {
			item.SizeBytes += uint64(stat.Blocks) * 512
		}
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, context.Canceled) {
		item.Truncated = true
	}
	if ctx.Err() != nil {
		item.Truncated = true
	}
	return item
}
