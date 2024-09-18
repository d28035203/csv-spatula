# CSV Spatula

Go **CLI utilities** for batch file work: merge unique YouTube entries from CSV exports and turn takeout CSVs into bookmark folder structures.

## Tools

- `merge_unique_videos.go` — dedupe / merge video lists from CSV
- `takeout-csv-to-bookmark-folders.go` — CSV → bookmark folders

## Tech

Go · encoding/csv · os/path

## Run

```bash
git clone https://github.com/d28035203/csv-spatula.git
cd csv-spatula
go run merge_unique_videos.go
go run takeout-csv-to-bookmark-folders.go
```

## License

MIT
