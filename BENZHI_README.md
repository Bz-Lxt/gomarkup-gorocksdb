# GoRocksDB 评测说明

本项目是基于Go语言的基于 LSM-Tree（Log-Structured Merge-Tree），旨在解决基于 LSM-Tree（Log-Structured Merge-Tree）相关的工程问题，使用了Go、React，功能有LSM-Tree 内存与磁盘全景墙、磁盘 Compaction 实时动画大屏、内存跳表（SkipList）与 WAL、SSTable 磁盘多级索引与分层归并。

Go 模块位于 `backend/`。评测入口：在该目录执行 `go test ./...`。
