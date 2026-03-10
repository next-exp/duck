//go:build ignore

package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

const headerSize = 80

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: go run extract_events.go <input.rd> <output.rd> [num_events]\n")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]
	numEvents := 2
	if len(os.Args) > 3 {
		fmt.Sscanf(os.Args[3], "%d", &numEvents)
	}

	in, err := os.Open(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening input: %v\n", err)
		os.Exit(1)
	}
	defer in.Close()

	out, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	for i := 0; i < numEvents; i++ {
		header := make([]byte, headerSize)
		n, err := in.Read(header)
		if err != nil || n < headerSize {
			fmt.Fprintf(os.Stderr, "Error reading header: %v\n", err)
			break
		}

		eventSize := binary.LittleEndian.Uint32(header[0:4])
		fmt.Printf("Event %d: size=%d bytes\n", i+1, eventSize)

		eventData := make([]byte, eventSize)
		copy(eventData, header)

		remaining := int(eventSize) - headerSize
		if remaining > 0 {
			n, err := in.Read(eventData[headerSize:])
			if err != nil || n < remaining {
				fmt.Fprintf(os.Stderr, "Error reading event data: %v\n", err)
				break
			}
		}

		out.Write(eventData)
	}

	fmt.Printf("Wrote %d events to %s\n", numEvents, outputFile)
}
