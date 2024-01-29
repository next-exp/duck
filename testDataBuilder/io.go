package main

import (
	"encoding/binary"
	"os"
)

func writeBinaryData(data []byte, fd *os.File) error {
	//start := time.Now()
	err := binary.Write(fd, binary.LittleEndian, data)
	//duration := time.Since(start)
	//size := len(data)
	//speed := float64(size) / float64(duration.Milliseconds()) * 1000 / 1024 / 1024
	//fmt.Printf("Duration: %d, bytes: %d, speed %f MB/s\n", duration.Milliseconds(), size, speed)
	return err
}
