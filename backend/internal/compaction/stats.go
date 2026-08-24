package compaction

type Stats struct {
	Level           int    `json:"level"`
	OutputLevel     int    `json:"output_level"`
	InputFiles      []uint64 `json:"input_files"`
	OutputFiles     []uint64 `json:"output_files"`
	KeysRead        int64  `json:"keys_read"`
	KeysWritten     int64  `json:"keys_written"`
	DroppedVersions int64  `json:"dropped_versions"`
	DroppedTombs    int64  `json:"dropped_tombs"`
	BytesRead       int64  `json:"bytes_read"`
	BytesWritten    int64  `json:"bytes_written"`
	DurationMS      int64  `json:"duration_ms"`
}
