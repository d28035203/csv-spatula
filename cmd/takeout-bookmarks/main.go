// Command takeout-bookmarks turns YouTube Takeout playlist CSVs into Chrome bookmark HTML.
package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/d28035203/csv-spatula/internal/ytcsv"
)

const (
	htmlFilename  = "playlist_bookmarks.html"
	fallbackTitle = "YouTube Video"
)

type bookmark struct {
	URL     string
	Title   string
	AddDate int64
}

func main() {
	dir := flag.String("dir", ".", "directory containing Takeout playlist CSV files")
	ytDlp := flag.String("yt-dlp", "yt-dlp", "path to yt-dlp binary (for video titles)")
	delay := flag.Duration("delay", 700*time.Millisecond, "pause between yt-dlp title fetches")
	skipTitles := flag.Bool("skip-titles", false, "do not call yt-dlp; use fallback titles")
	cachePath := flag.String("cache", ".title-cache.tsv", "path to title cache (id\\ttitle); empty disables")
	flag.Parse()

	pattern := filepath.Join(*dir, "*.csv")
	csvFiles, err := filepath.Glob(pattern)
	if err != nil {
		log.Fatal(err)
	}
	// Skip merge tool output and other non-playlist dumps if present in the same folder.
	filtered := csvFiles[:0]
	for _, f := range csvFiles {
		base := strings.ToLower(filepath.Base(f))
		if base == "merged_unique_videos.csv" || strings.HasPrefix(base, "merged_") {
			continue
		}
		filtered = append(filtered, f)
	}
	csvFiles = filtered
	if len(csvFiles) == 0 {
		fmt.Printf("No playlist .csv files in %s\n", *dir)
		fmt.Println("Place Takeout playlist CSVs there, or pass -dir.")
		os.Exit(1)
	}

	cache := newTitleCache(*cachePath)
	defer cache.Save()

	fmt.Printf("Found %d CSV file(s). Starting...\n\n", len(csvFiles))

	for _, csvPath := range csvFiles {
		if err := processCSV(csvPath, *ytDlp, *delay, *skipTitles, cache); err != nil {
			log.Printf("Failed %s: %v", csvPath, err)
		}
	}

	fmt.Println("Done.")
	fmt.Println("Import each folder's playlist_bookmarks.html in Chrome:")
	fmt.Println("  Bookmarks manager → ⋮ → Import bookmarks → choose the HTML file")
}

func processCSV(csvPath, ytDlp string, delay time.Duration, skipTitles bool, cache *titleCache) error {
	base := filepath.Base(csvPath)
	folderName := strings.TrimSuffix(base, filepath.Ext(base))
	if folderName == base {
		return fmt.Errorf("%s is not a .csv file", csvPath)
	}

	outputFolder := filepath.Join(filepath.Dir(csvPath), folderName)
	if err := os.MkdirAll(outputFolder, 0o755); err != nil {
		return err
	}

	fmt.Printf("Processing %s → %s/\n", base, folderName)

	f, err := os.Open(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true

	if _, err := r.Read(); err != nil {
		return fmt.Errorf("read header: %w", err)
	}

	var bookmarks []bookmark
	videoCount := 0

	for {
		row, err := r.Read()
		if ytcsv.IsEOF(err) {
			break
		}
		if err != nil {
			log.Printf("  row error in %s: %v", base, err)
			continue
		}
		if len(row) < 1 {
			continue
		}

		videoID := strings.TrimSpace(row[0])
		if !ytcsv.IsVideoID(videoID) {
			continue
		}

		ts := ""
		if len(row) >= 2 {
			ts = strings.TrimSpace(row[1])
		}

		title := resolveTitle(videoID, ytDlp, skipTitles, cache)
		bookmarks = append(bookmarks, bookmark{
			URL:     ytcsv.WatchURL(videoID),
			Title:   title,
			AddDate: ytcsv.ParseTimeUnix(ts),
		})

		videoCount++
		if videoCount%10 == 0 {
			fmt.Printf("  %d videos...\n", videoCount)
		}

		if !skipTitles {
			time.Sleep(delay)
		}
	}

	if videoCount == 0 {
		fmt.Printf("  → no valid videos in %s\n", base)
		return nil
	}

	if err := writeBookmarksHTML(outputFolder, folderName, bookmarks); err != nil {
		return err
	}
	fmt.Printf("  → %d bookmarks → %s/%s\n\n", videoCount, folderName, htmlFilename)
	return nil
}

func resolveTitle(videoID, ytDlp string, skip bool, cache *titleCache) string {
	if t, ok := cache.Get(videoID); ok {
		return ytcsv.EscapeHTML(t)
	}
	if skip {
		t := fmt.Sprintf("%s %s", fallbackTitle, videoID)
		cache.Set(videoID, t)
		return ytcsv.EscapeHTML(t)
	}

	raw := fetchTitle(ytDlp, videoID)
	cache.Set(videoID, raw)
	return ytcsv.EscapeHTML(raw)
}

func fetchTitle(ytDlp, videoID string) string {
	url := ytcsv.WatchURL(videoID)
	cmd := exec.Command(ytDlp, "--get-title", "--quiet", "--no-warnings", url)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("  yt-dlp %s: %v", videoID, err)
		return fmt.Sprintf("%s %s", fallbackTitle, videoID)
	}
	title := strings.TrimSpace(string(output))
	if title == "" {
		return fmt.Sprintf("%s %s", fallbackTitle, videoID)
	}
	return title
}

func writeBookmarksHTML(folderPath, folderLabel string, bookmarks []bookmark) error {
	filename := filepath.Join(folderPath, htmlFilename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	sort.Slice(bookmarks, func(i, j int) bool {
		return bookmarks[i].AddDate < bookmarks[j].AddDate
	})

	label := ytcsv.EscapeHTML(folderLabel)
	header := fmt.Sprintf(`<!DOCTYPE NETSCAPE-Bookmark-file-1>
<!-- Generated by csv-spatula takeout-bookmarks — %s -->
<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
<TITLE>%s</TITLE>
<H1>%s</H1>
<DL><p>
    <DT><H3 ADD_DATE="0" FOLDED>%s</H3>
    <DL><p>
`, time.Now().Format("2006-01-02"), label, label, label)

	if _, err := w.WriteString(header); err != nil {
		return err
	}
	for _, b := range bookmarks {
		fmt.Fprintf(w, `        <DT><A HREF="%s" ADD_DATE="%d">%s</A>
`, b.URL, b.AddDate, b.Title)
	}
	_, err = w.WriteString(`    </DL><p>
</DL><p>
`)
	return err
}

// titleCache is a simple id → title map persisted as TSV.
type titleCache struct {
	path string
	mu   sync.Mutex
	m    map[string]string
	dirty bool
}

func newTitleCache(path string) *titleCache {
	c := &titleCache{path: path, m: make(map[string]string)}
	if path == "" {
		return c
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return c
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		c.m[parts[0]] = parts[1]
	}
	return c
}

func (c *titleCache) Get(id string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	t, ok := c.m[id]
	return t, ok
}

func (c *titleCache) Set(id, title string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if prev, ok := c.m[id]; ok && prev == title {
		return
	}
	c.m[id] = title
	c.dirty = true
}

func (c *titleCache) Save() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.path == "" || !c.dirty {
		return
	}
	var b strings.Builder
	b.WriteString("# youtube video id\ttitle\n")
	ids := make([]string, 0, len(c.m))
	for id := range c.m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		fmt.Fprintf(&b, "%s\t%s\n", id, c.m[id])
	}
	if err := os.WriteFile(c.path, []byte(b.String()), 0o644); err != nil {
		log.Printf("title cache save: %v", err)
	}
}
