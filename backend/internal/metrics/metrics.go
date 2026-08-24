package metrics

import "sync/atomic"

type Metrics struct {
	Puts            atomic.Int64
	Gets            atomic.Int64
	Deletes         atomic.Int64
	Scans           atomic.Int64
	GetHits         atomic.Int64
	GetMisses       atomic.Int64
	BytesWritten    atomic.Int64
	BytesRead       atomic.Int64
	Flushes         atomic.Int64
	Compactions     atomic.Int64
	WriteStalls     atomic.Int64
	BloomRejects    atomic.Int64
	SSTTouched      atomic.Int64
	DroppedVersions atomic.Int64
	DroppedTombs    atomic.Int64
	WALSyncs        atomic.Int64
	RotateCount     atomic.Int64
	WriteNanos      atomic.Int64
	WriteSamples    atomic.Int64
}

func (m *Metrics) ObserveWrite(ns int64) {
	m.WriteNanos.Add(ns)
	m.WriteSamples.Add(1)
}

func (m *Metrics) Snapshot() map[string]any {
	samples := m.WriteSamples.Load()
	var avg float64
	if samples > 0 {
		avg = float64(m.WriteNanos.Load()) / float64(samples) / 1e6
	}
	return map[string]any{
		"puts":             m.Puts.Load(),
		"gets":             m.Gets.Load(),
		"deletes":          m.Deletes.Load(),
		"scans":            m.Scans.Load(),
		"get_hits":         m.GetHits.Load(),
		"get_misses":       m.GetMisses.Load(),
		"bytes_written":    m.BytesWritten.Load(),
		"bytes_read":       m.BytesRead.Load(),
		"flushes":          m.Flushes.Load(),
		"compactions":      m.Compactions.Load(),
		"write_stalls":     m.WriteStalls.Load(),
		"bloom_rejects":    m.BloomRejects.Load(),
		"sst_touched":      m.SSTTouched.Load(),
		"dropped_versions": m.DroppedVersions.Load(),
		"dropped_tombs":    m.DroppedTombs.Load(),
		"wal_syncs":        m.WALSyncs.Load(),
		"rotates":          m.RotateCount.Load(),
		"avg_write_ms":     avg,
	}
}
