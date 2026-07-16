// Command merge-unique merges YouTube playlist CSVs and writes unique video IDs.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/d28035203/csv-spatula/internal/ytcsv"
)

type videoEntry struct {
	ID        string
	Timestamp string
}

func main() {
	dir := flag.String("dir", ".", "directory containing playlist CSV files")
	out := flag.String("out", "merged_unique_videos.csv", "output CSV path")
	flag.Parse()

	pattern := filepath.Join(*dir, "*.csv")
	files, err := filepath.Glob(pattern)
	if err != nil {
		log.Fatalf("list CSV files: %v", err)
	}
	// Never merge our own output back in if it lives next to inputs.
	filtered := files[:0]
	outBase := filepath.Base(*out)
	for _, f := range files {
		if filepath.Base(f) == outBase {
			continue
		}
		filtered = append(filtered, f)
	}
	files = filtered

	if len(files) == 0 {
		fmt.Printf("No .csv files found in %s\n", *dir)
		fmt.Println("Usage: merge-unique -dir path/to/csvs -out merged.csv")
		os.Exit(1)
	}

	fmt.Printf("Found %d CSV file(s) in %s\n", len(files), *dir)

	seen := make(map[string]videoEntry)
	for _, filename := range files {
		fmt.Printf("  Reading: %s\n", filepath.Base(filename))
		n, err := processFile(filename, seen)
		if err != nil {
			log.Printf("Error processing %s: %v", filename, err)
			continue
		}
		fmt.Printf("    → %d new unique IDs\n", n)
	}

	if len(seen) == 0 {
		fmt.Println("No valid video entries found.")
		os.Exit(1)
	}

	entries := make([]videoEntry, 0, len(seen))
	for _, e := range seen {
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })

	if err := writeMerged(*out, entries); err != nil {
		log.Fatalf("write output: %v", err)
	}

	fmt.Printf("\nSuccess → %d unique videos → %s\n", len(entries), *out)
	fmt.Println("Columns: Video ID, Playlist video creation timestamp")
}

func processFile(filename string, seen map[string]videoEntry) (int, error) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1 // Takeout rows can vary slightly

	header, err := r.Read()
	if err != nil {
		return 0, fmt.Errorf("read header: %w", err)
	}
	if len(header) < 1 || !strings.Contains(strings.ToLower(header[0]), "video id") {
		fmt.Printf("    Warning: header does not look like Takeout (first col: %q)\n", first(header))
	}

	added := 0
	for {
		row, err := r.Read()
		if ytcsv.IsEOF(err) {
			break
		}
		if err != nil {
			return added, fmt.Errorf("read row: %w", err)
		}
		if len(row) < 1 {
			continue
		}
		videoID := strings.TrimSpace(row[0])
		if !ytcsv.IsVideoID(videoID) {
			continue
		}
		timestamp := ""
		if len(row) >= 2 {
			timestamp = strings.TrimSpace(row[1])
		}
		if _, exists := seen[videoID]; exists {
			continue
		}
		seen[videoID] = videoEntry{ID: videoID, Timestamp: timestamp}
		added++
	}
	return added, nil
}

func writeMerged(path string, entries []videoEntry) error {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"Video ID", "Playlist video creation timestamp"}); err != nil {
		return err
	}
	for _, e := range entries {
		if err := w.Write([]string{e.ID, e.Timestamp}); err != nil {
			return err
		}
	}
	return w.Error()
}

func first(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	return ss[0]
}
