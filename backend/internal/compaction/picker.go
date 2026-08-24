package compaction

import (
	"gorocksdb/internal/config"
	"gorocksdb/internal/version"
)

type Job struct {
	Level      int
	Inputs     []*version.FileMeta
	Grandpa    []*version.FileMeta
	Score      float64
	OutputLevel int
}

func Pick(v *version.Version, p config.Profile) *Job {
	best := (*Job)(nil)
	bestScore := 0.0

	l0n := float64(v.NumFiles(0))
	l0score := l0n / float64(p.L0CompactionTrigger)
	if l0score >= 1 {
		inputs := append([]*version.FileMeta(nil), v.Files[0]...)
		var smallest, largest []byte
		for i, f := range inputs {
			if i == 0 || bytesLess(f.Smallest, smallest) {
				smallest = f.Smallest
			}
			if i == 0 || bytesGreater(f.Largest, largest) {
				largest = f.Largest
			}
		}
		l1 := v.Overlapping(1, smallest, largest)
		best = &Job{Level: 0, Inputs: append(inputs, l1...), Score: l0score, OutputLevel: 1}
		bestScore = l0score
	}

	for lv := 1; lv < config.NumLevels-1; lv++ {
		need := p.MaxBytesForLevel(lv)
		have := v.Bytes(lv)
		score := float64(have) / float64(need)
		if score < 1 || score <= bestScore {
			continue
		}
		if len(v.Files[lv]) == 0 {
			continue
		}
		pick := v.Files[lv][0]
		// pick largest file
		for _, f := range v.Files[lv] {
			if f.Size > pick.Size {
				pick = f
			}
		}
		lnext := v.Overlapping(lv+1, pick.Smallest, pick.Largest)
		inputs := append([]*version.FileMeta{pick}, lnext...)
		best = &Job{Level: lv, Inputs: inputs, Score: score, OutputLevel: lv + 1}
		bestScore = score
	}
	return best
}

func bytesLess(a, b []byte) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

func bytesGreater(a, b []byte) bool {
	return bytesLess(b, a)
}
