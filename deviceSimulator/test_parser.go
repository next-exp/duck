// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test_parser.go <file.rd>")
		os.Exit(1)
	}

	filePath := os.Args[1]

	fmt.Printf("Testing parser with file: %s\n", filePath)

	// Create parser
	parser, err := NewFileParser(filePath)
	if err != nil {
		log.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	// Parse all events
	events, err := parser.ParseAll()
	if err != nil {
		log.Fatalf("Failed to parse events: %v", err)
	}

	fmt.Printf("Successfully parsed %d events\n", len(events))

	// Show summary of first few events
	for i, event := range events {
		if i >= 5 {
			break
		}

		fmt.Printf("\nEvent %d:\n", i+1)
		fmt.Printf("  Event ID: %d\n", event.EventID)
		fmt.Printf("  Timestamp: %d µs\n", event.Timestamp)
		fmt.Printf("  Equipment count: %d\n", len(event.Equipments))

		for equipID, data := range event.Equipments {
			fmt.Printf("    Equipment %d: %d bytes\n", equipID, len(data))
		}
	}

	// Show statistics
	totalEquipmentData := make(map[uint32]int)
	for _, event := range events {
		for equipID, data := range event.Equipments {
			totalEquipmentData[equipID] += len(data)
		}
	}

	fmt.Printf("\nStatistics:\n")
	fmt.Printf("Total events: %d\n", len(events))
	fmt.Printf("Equipment data summary:\n")
	for equipID, totalBytes := range totalEquipmentData {
		eventsWithEquip := 0
		for _, event := range events {
			if _, ok := event.Equipments[equipID]; ok {
				eventsWithEquip++
			}
		}
		fmt.Printf("  Equipment %d: %d bytes across %d events (avg: %d bytes/event)\n",
			equipID, totalBytes, eventsWithEquip, totalBytes/eventsWithEquip)
	}
}
