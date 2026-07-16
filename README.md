# CSV Spatula

> Repository: `csv-spatula` · https://github.com/d28035203/csv-spatula

Small **Go CLIs** for YouTube playlist / Google Takeout CSV cleanup:

1. **Merge & dedupe** video IDs across many playlist CSVs  
2. **Export bookmark HTML** (Chrome-importable) per playlist, with titles via `yt-dlp`

Personal utility — not a full YouTube client.

## Tools

| Command | Purpose |
|---------|---------|
| `merge-unique` | Merge all `*.csv` in a folder → one CSV of unique video IDs |
| `takeout-bookmarks` | Each playlist CSV → folder + `playlist_bookmarks.html` |

## Expected CSV format

YouTube Takeout playlist exports typically look like:

```text
Video ID,Playlist video creation timestamp
dQw4w9WgXcQ,2024-01-15T12:34:56+00:00
xxxxxxxxxxx,2024-02-01T08:00:00+00:00
```

- Column 0: 11-character video ID  
- Column 1 (optional for merge; used for bookmark dates): timestamp  

## Prerequisites

- **Go 1.22+**
- For titles: **[yt-dlp](https://github.com/yt-dlp/yt-dlp)** on `PATH`  
  (`brew install yt-dlp` on macOS)

## Install / run

```bash
git clone https://github.com/d28035203/csv-spatula.git
cd csv-spatula
go test ./...
```

### Merge unique videos

```bash
# CSVs in current directory
go run ./cmd/merge-unique

# explicit paths
go run ./cmd/merge-unique -dir ~/Takeout/playlists -out ~/merged_unique_videos.csv
```

Output columns: `Video ID`, `Playlist video creation timestamp` (first-seen timestamp kept).

### Takeout → Chrome bookmarks

```bash
# slow first run (yt-dlp per video); titles cached in .title-cache.tsv
go run ./cmd/takeout-bookmarks -dir ~/Takeout/playlists

# faster re-runs (cache hits)
go run ./cmd/takeout-bookmarks -dir ~/Takeout/playlists

# no network / no yt-dlp
go run ./cmd/takeout-bookmarks -dir ~/Takeout/playlists -skip-titles
```

Then in Chrome: **Bookmarks manager → ⋮ → Import bookmarks →** pick each folder’s `playlist_bookmarks.html`.

| Flag | Default | Meaning |
|------|---------|---------|
| `-dir` | `.` | Folder with `*.csv` |
| `-yt-dlp` | `yt-dlp` | Binary path |
| `-delay` | `700ms` | Pause between title fetches |
| `-skip-titles` | false | Skip yt-dlp; use `YouTube Video <id>` |
| `-cache` | `.title-cache.tsv` | Title cache path (`""` disables) |

## Layout

```
csv-spatula/
├── cmd/
│   ├── merge-unique/          # dedupe/merge CLIs
│   └── takeout-bookmarks/     # CSV → bookmark HTML
├── internal/ytcsv/            # shared ID/time/HTML helpers + tests
├── go.mod
└── README.md
```

## Notes

- `merge-unique` skips its own output filename if present in `-dir`.  
- `takeout-bookmarks` is intentionally polite (`-delay`) so YouTube is less likely to throttle yt-dlp.  
- Large libraries: first title fetch can take a long time; rely on the cache afterward.

## License

MIT
