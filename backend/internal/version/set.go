package version

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"gorocksdb/internal/encoding"
	"gorocksdb/internal/logger"
)

type VersionSet struct {
	mu          sync.Mutex
	dir         string
	current     atomic.Pointer[Version]
	nextFile    uint64
	lastSeq     uint64
	logNumber   uint64
	manifestNum uint64
	mf          *os.File
	obsolete    []*FileMeta
}

func Open(dir string) (*VersionSet, error) {
	vs := &VersionSet{dir: dir}
	v := NewVersion()
	vs.current.Store(v)
	vs.nextFile = 1
	cur := filepath.Join(dir, "CURRENT")
	data, err := os.ReadFile(cur)
	if err != nil {
		if os.IsNotExist(err) {
			if err := vs.createNew(); err != nil {
				return nil, err
			}
			return vs, nil
		}
		return nil, err
	}
	name := strings.TrimSpace(string(data))
	if err := vs.recoverManifest(filepath.Join(dir, name)); err != nil {
		return nil, err
	}
	return vs, nil
}

func (vs *VersionSet) createNew() error {
	vs.manifestNum = 1
	vs.nextFile = 2
	path := vs.manifestPath(vs.manifestNum)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	vs.mf = f
	e := &VersionEdit{}
	e.SetNextFile(vs.nextFile)
	e.SetLogNumber(0)
	e.SetLastSeq(0)
	if err := vs.writeEdit(e); err != nil {
		return err
	}
	return vs.writeCurrent()
}

func (vs *VersionSet) manifestPath(n uint64) string {
	return filepath.Join(vs.dir, fmt.Sprintf("MANIFEST-%06d", n))
}

func (vs *VersionSet) writeCurrent() error {
	tmp := filepath.Join(vs.dir, "CURRENT.tmp")
	name := fmt.Sprintf("MANIFEST-%06d\n", vs.manifestNum)
	if err := os.WriteFile(tmp, []byte(name), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(vs.dir, "CURRENT"))
}

func (vs *VersionSet) recoverManifest(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	base := NewVersion()
	off := 0
	for off < len(data) {
		payload, n, err := DecodeRecord(data[off:])
		if err != nil {
			if err == encoding.ErrShortRecord {
				logger.L().Warn("manifest tail truncated", "path", path)
				break
			}
			return err
		}
		e, err := DecodeEdit(payload)
		if err != nil {
			return err
		}
		base = applyEdit(base, e)
		if e.HasNextFile && e.NextFile > vs.nextFile {
			vs.nextFile = e.NextFile
		}
		if e.HasLastSeq && e.LastSeq > vs.lastSeq {
			vs.lastSeq = e.LastSeq
		}
		if e.HasLogNumber {
			vs.logNumber = e.LogNumber
		}
		off += n
	}
	num := parseManifestNum(path)
	vs.manifestNum = num
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	vs.mf = f
	vs.current.Store(base)
	if vs.nextFile < 1 {
		vs.nextFile = 1
	}
	return nil
}

func parseManifestNum(path string) uint64 {
	base := filepath.Base(path)
	s := strings.TrimPrefix(base, "MANIFEST-")
	n, _ := strconv.ParseUint(s, 10, 64)
	return n
}

func (vs *VersionSet) writeEdit(e *VersionEdit) error {
	rec := EncodeRecord(e.Encode())
	if _, err := vs.mf.Write(rec); err != nil {
		return err
	}
	return vs.mf.Sync()
}

func (vs *VersionSet) Current() *Version {
	return vs.current.Load()
}

func (vs *VersionSet) LastSequence() uint64 { return atomic.LoadUint64(&vs.lastSeq) }

func (vs *VersionSet) SetLastSequence(n uint64) {
	for {
		cur := atomic.LoadUint64(&vs.lastSeq)
		if n <= cur {
			return
		}
		if atomic.CompareAndSwapUint64(&vs.lastSeq, cur, n) {
			return
		}
	}
}

func (vs *VersionSet) NewFileNumber() uint64 {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	n := vs.nextFile
	vs.nextFile++
	return n
}

func (vs *VersionSet) LogNumber() uint64 { return vs.logNumber }

func (vs *VersionSet) Apply(e *VersionEdit) (*Version, []*FileMeta, error) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	if !e.HasNextFile {
		e.SetNextFile(vs.nextFile)
	} else if e.NextFile > vs.nextFile {
		vs.nextFile = e.NextFile
	}
	if !e.HasLastSeq {
		e.SetLastSeq(atomic.LoadUint64(&vs.lastSeq))
	}
	if err := vs.writeEdit(e); err != nil {
		return nil, nil, err
	}
	if e.HasLogNumber {
		vs.logNumber = e.LogNumber
	}
	if e.HasLastSeq {
		vs.SetLastSequence(e.LastSeq)
	}
	old := vs.current.Load()
	nv := applyEdit(old, e)
	vs.current.Store(nv)
	old.Unref()

	var obsolete []*FileMeta
	live := map[uint64]struct{}{}
	for _, f := range nv.AllFiles() {
		live[f.Number] = struct{}{}
	}
	for _, d := range e.Deleted {
		if _, ok := live[d.Number]; !ok {
			obsolete = append(obsolete, &FileMeta{Number: d.Number, Level: d.Level})
		}
	}
	return nv, obsolete, nil
}

func (vs *VersionSet) Close() error {
	if vs.mf != nil {
		return vs.mf.Close()
	}
	return nil
}

func (vs *VersionSet) Dir() string { return vs.dir }
