package encoding

import (
	"encoding/binary"
	"hash/crc32"
)

var crcTable = crc32.MakeTable(crc32.Castagnoli)

func CRC32C(data []byte) uint32 {
	return crc32.Checksum(data, crcTable)
}

func CRC32CUpdate(crc uint32, data []byte) uint32 {
	return crc32.Update(crc, crcTable, data)
}

func PutCRC32C(buf []byte, crc uint32) {
	binary.LittleEndian.PutUint32(buf, crc)
}

func GetCRC32C(buf []byte) uint32 {
	return binary.LittleEndian.Uint32(buf)
}

// MaskCRC is the LevelDB-style masked CRC so a zeroed buffer is not a valid checksum.
func MaskCRC(crc uint32) uint32 {
	return ((crc >> 15) | (crc << 17)) + 0xa282ead8
}

func UnmaskCRC(masked uint32) uint32 {
	rot := masked - 0xa282ead8
	return (rot >> 17) | (rot << 15)
}
