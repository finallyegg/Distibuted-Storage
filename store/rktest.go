package store

import (
	"os"
)

func rkChunk(buf []byte, len uint64) uint {
	HASHLEN := 32
	THE_PRIME := 31
	MINCHUNK := 2048
	TARGETCHUNK := 4096
	MAXCHUNK := 8192

	var i int
	var hash uint64
	var off uint64
	var b uint64
	var b_n uint64
	var saved [256]uint64

	if b == 0 {
		b = uint64(THE_PRIME)
		b_n = 1
		for i = 0; i < (HASHLEN - 1); i++ {
			b_n *= b
		}
		for i = 0; i < 256; i++ {
			saved[i] = uint64(i) * b_n
		}
	}

	for off = 0; (off < uint64(HASHLEN)) && (off < len); off++ {
		hash = hash*b + uint64(buf[off])
	}

	for off < len {
		hash = (hash-saved[buf[off-uint64(HASHLEN)]])*b + uint64(buf[off])
		off++

		if ((off >= uint64(MINCHUNK)) && ((hash % uint64(TARGETCHUNK)) == 1)) || (off >= uint64(MAXCHUNK)) {
			return uint(off)
		}
	}

	return uint(off)
}

func rkMain(path string) []int {
	lenSize := []int{}
	file, _ := os.Open(path)
	fileInfo, _ := file.Stat()
	fileSize := fileInfo.Size()
	bufferChunk := make([]byte, int(fileSize))
	file.Read(bufferChunk)
	// fmt.Printf("read '%s' %d bytes\n", path, bytesRead)

	var chunks uint64 = 0
	var chunkLen uint64 = 0

	var off uint64 = 0
	for int64(off) < fileSize {
		ret := rkChunk(bufferChunk[off:], uint64(fileSize)-off)
		chunks++
		chunkLen += uint64(ret)
		// fmt.Printf("Chunk at offset %d, len %d\n", off, ret)
		lenSize = append(lenSize, int(ret))
		off += uint64(ret)
	}
	file.Close()
	return lenSize
}

// func main(){
// 	args := os.Args
// 	rkMain(args[1])
// }
