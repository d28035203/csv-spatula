// takeout-csv-to-bookmark-folders.go — potential-spoon.
// Author: d28035203

package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	htmlFilename  = "playlist_bookmarks.html"
	fallbackTitle = "YouTube Video"
	ytDlpPath     = "yt-dlp" // should be in PATH after brew install
)

type Bookmark struct {
	URL     string
	Title   string
	AddDate int64
}

func iso8601ToUnix(isoStr string) int64 {
	if isoStr == "" {
		return 0
	}
	// Try standard RFC3339
	t, err := time.Parse(time.RFC3339, isoStr)
	if err == nil {
		return t.Unix()
	}
	// Fallback: replace +00:00 with Z
	isoStr = strings.ReplaceAll(isoStr, "+00:00", "Z")
	t, err = time.Parse(time.RFC3339, isoStr)
	if err == nil {
		return t.Unix()
	}
	return 0
}

func fetchTitle(videoID string) string {
	url := "https://www.youtube.com/watch?v=" + videoID
	cmd := exec.Command(ytDlpPath, "--get-title", "--quiet", "--no-warnings", url)

	output, err := cmd.Output()
	if err != nil {
		log.Printf("yt-dlp error for %s: %v", videoID, err)
		return fmt.Sprintf("%s %s", fallbackTitle, videoID)
	}

	title := strings.TrimSpace(string(output))
	if title == "" {
		return fmt.Sprintf("%s %s", fallbackTitle, videoID)
	}

	// Escape HTML entities
	title = strings.ReplaceAll(title, "&", "&amp;")
	title = strings.ReplaceAll(title, "<", "&lt;")
	title = strings.ReplaceAll(title, ">", "&gt;")
	title = strings.ReplaceAll(title, `"`, "&quot;")

	return title
}

func writeBookmarksHTML(folderPath string, bookmarks []Bookmark) error {
	filename := filepath.Join(folderPath, htmlFilename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	// Sort by timestamp (oldest first)
	sort.Slice(bookmarks, func(i, j int) bool {
		return bookmarks[i].AddDate < bookmarks[j].AddDate
	})

	header := fmt.Sprintf(`<!DOCTYPE NETSCAPE-Bookmark-file-1>
<!-- Generated from YouTube Takeout CSV - %s -->
<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
<TITLE>%s</TITLE>
<H1>%s</H1>
<DL><p>
    <DT><H3 ADD_DATE="0" FOLDED>%s</H3>
    <DL><p>
`, time.Now().Format("2006-01-02"), folderPath, folderPath, folderPath)

	_, err = w.WriteString(header)
	if err != nil {
		return err
	}

	for _, b := range bookmarks {
		fmt.Fprintf(w, `        <DT><A HREF="%s" ADD_DATE="%d">%s</A>
`, b.URL, b.AddDate, b.Title)
	}

	footer := `    </DL><p>
</DL><p>
`
	_, err = w.WriteString(footer)
	return err
}

func processCSV(csvPath string) error {
	base := filepath.Base(csvPath)
	folderName := strings.TrimSuffix(base, ".csv")
	if folderName == base { // no .csv extension
		return fmt.Errorf("%s does not end with .csv", csvPath)
	}

	outputFolder := filepath.Join(filepath.Dir(csvPath), folderName)
	if err := os.MkdirAll(outputFolder, 0755); err != nil {
		return err
	}

	fmt.Printf("Processing %s → %s/\n", base, folderName)

	f, err := os.Open(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	// Skip header row
	_, err = r.Read()
	if err != nil {
		return fmt.Errorf("cannot read header from %s: %w", csvPath, err)
	}

	var bookmarks []Bookmark
	videoCount := 0

	for {
		row, err := r.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			log.Printf("Error reading row in %s: %v", csvPath, err)
			continue
		}

		if len(row) < 1 {
			continue
		}

		videoID := strings.TrimSpace(row[0])
		if len(videoID) != 11 {
			continue // invalid ID
		}

		ts := ""
		if len(row) >= 2 {
			ts = strings.TrimSpace(row[1])
		}

		title := fetchTitle(videoID)
		url := "https://www.youtube.com/watch?v=" + videoID
		addDate := iso8601ToUnix(ts)

		bookmarks = append(bookmarks, Bookmark{
			URL:     url,
			Title:   title,
			AddDate: addDate,
		})

		videoCount++
		if videoCount%10 == 0 {
			fmt.Printf("  %d videos processed...\n", videoCount)
		}

		// Gentle delay to avoid rate limiting / blocks
		time.Sleep(700 * time.Millisecond)
	}

	if videoCount == 0 {
		fmt.Printf("  → No valid videos found in %s\n", base)
		return nil
	}

	if err := writeBookmarksHTML(outputFolder, bookmarks); err != nil {
		return err
	}

	fmt.Printf("  → Saved %d bookmarks to %s/%s\n\n", videoCount, folderName, htmlFilename)
	return nil
}

func main() {
	csvFiles, err := filepath.Glob("*.csv")
	if err != nil {
		log.Fatal(err)
	}

	if len(csvFiles) == 0 {
		fmt.Println("No .csv files found in current directory.")
		fmt.Println("Place this program in the folder with your Takeout CSV files.")
		return
	}

	fmt.Printf("Found %d CSV files. Starting...\n\n", len(csvFiles))

	for _, csvPath := range csvFiles {
		if err := processCSV(csvPath); err != nil {
			log.Printf("Failed to process %s: %v", csvPath, err)
		}
	}

	fmt.Println("All processing finished.")
	fmt.Println("You can now import playlist_bookmarks.html from each folder into Chrome:")
	fmt.Println("→ Bookmarks manager → three dots → Import bookmarks → choose HTML file")
}
