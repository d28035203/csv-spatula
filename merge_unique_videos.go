// merge_unique_videos.go — csv-spatula.
// Author: d28035203

package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	outputFile = "merged_unique_videos.csv"
)

type VideoEntry struct {
	ID        string
	Timestamp string
}

func main() {
	// Find all .csv files in current directory
	files, err := filepath.Glob("*.csv")
	if err != nil {
		log.Fatalf("Failed to list CSV files: %v", err)
	}
	if len(files) == 0 {
		fmt.Println("No .csv files found in the current directory.")
		return
	}

	fmt.Printf("Found %d CSV file(s)\n", len(files))

	// Use map to deduplicate: key = video ID, value = entry
	seen := make(map[string]VideoEntry)

	for _, filename := range files {
		fmt.Printf("  Reading: %s\n", filename)
		if err := processFile(filename, seen); err != nil {
			log.Printf("Error processing %s: %v", filename, err)
		}
	}

	if len(seen) == 0 {
		fmt.Println("No valid video entries found.")
		return
	}

	// Convert map to slice + sort by video ID
	entries := make([]VideoEntry, 0, len(seen))
	for _, entry := range seen {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})

	// Write merged result
	if err := writeMerged(entries); err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}

	fmt.Printf("\nSuccess!\n")
	fmt.Printf("→ Wrote %d unique videos to: %s\n", len(entries), outputFile)
	fmt.Println("Columns kept: Video ID,Playlist video creation timestamp")
}

func processFile(filename string, seen map[string]VideoEntry) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true

	// Read and skip header (we assume first row is header)
	header, err := r.Read()
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}
	if len(header) < 2 || !strings.Contains(strings.ToLower(header[0]), "video id") {
		fmt.Printf("  Warning: %s does not look like expected format (skipping header check)\n", filename)
	}

	count := 0
	for {
		row, err := r.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("error reading row: %w", err)
		}

		if len(row) < 2 {
			continue
		}

		videoID := strings.TrimSpace(row[0])
		if videoID == "" || len(videoID) != 11 { // typical YouTube ID length
			continue
		}

		timestamp := strings.TrimSpace(row[1])

		// Keep first occurrence only
		if _, exists := seen[videoID]; !exists {
			seen[videoID] = VideoEntry{
				ID:        videoID,
				Timestamp: timestamp,
			}
			count++
		}
	}

	fmt.Printf("    → Added %d new unique entries\n", count)
	return nil
}

func writeMerged(entries []VideoEntry) error {
	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Write header
	if err := w.Write([]string{"Video ID", "Playlist video creation timestamp"}); err != nil {
		return err
	}

	// Write sorted rows
	for _, e := range entries {
		if err := w.Write([]string{e.ID, e.Timestamp}); err != nil {
			return err
		}
	}

	return nil
}
