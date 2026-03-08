package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/FialaMoises/go-web-crawler/internal/crawler"
)

// ResultData represents exported crawl data
type ResultData struct {
	URL       string        `json:"url"`
	Depth     int           `json:"depth"`
	Success   bool          `json:"success"`
	LinksFound int          `json:"links_found"`
	Duration  string        `json:"duration"`
	Timestamp string        `json:"timestamp"`
	Error     string        `json:"error,omitempty"`
}

// SummaryData represents crawler summary statistics
type SummaryData struct {
	StartURL       string        `json:"start_url"`
	PagesVisited   int64         `json:"pages_visited"`
	PagesFailed    int64         `json:"pages_failed"`
	LinksFound     int64         `json:"links_found"`
	TotalDuration  string        `json:"total_duration"`
	AverageDuration string       `json:"average_duration"`
	StartTime      string        `json:"start_time"`
	EndTime        string        `json:"end_time"`
}

// ExportData holds all data to be exported
type ExportData struct {
	Summary SummaryData  `json:"summary"`
	Results []ResultData `json:"results"`
}

// Exporter handles exporting crawl results
type Exporter struct {
	outputPath string
}

// NewExporter creates a new Exporter
func NewExporter(outputPath string) *Exporter {
	return &Exporter{
		outputPath: outputPath,
	}
}

// Export exports results in the specified format
func (e *Exporter) Export(results []crawler.WorkerResult, stats crawler.Stats, startURL string, format string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(e.outputPath, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	// Prepare export data
	exportData := e.prepareExportData(results, stats, startURL)

	// Export based on format
	switch format {
	case "json":
		return e.exportJSON(exportData)
	case "csv":
		return e.exportCSV(exportData)
	case "both":
		if err := e.exportJSON(exportData); err != nil {
			return err
		}
		return e.exportCSV(exportData)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

// prepareExportData converts crawler results to export format
func (e *Exporter) prepareExportData(results []crawler.WorkerResult, stats crawler.Stats, startURL string) ExportData {
	resultData := make([]ResultData, 0, len(results))

	for _, r := range results {
		errMsg := ""
		if r.Error != nil {
			errMsg = r.Error.Error()
		}

		resultData = append(resultData, ResultData{
			URL:        r.URL,
			Depth:      r.Depth,
			Success:    r.Success,
			LinksFound: len(r.Links),
			Duration:   r.Duration.String(),
			Timestamp:  r.Timestamp.Format(time.RFC3339),
			Error:      errMsg,
		})
	}

	summary := SummaryData{
		StartURL:        startURL,
		PagesVisited:    stats.PagesVisited,
		PagesFailed:     stats.PagesFailed,
		LinksFound:      stats.LinksFound,
		TotalDuration:   stats.EndTime.Sub(stats.StartTime).String(),
		AverageDuration: stats.AverageDuration.String(),
		StartTime:       stats.StartTime.Format(time.RFC3339),
		EndTime:         stats.EndTime.Format(time.RFC3339),
	}

	return ExportData{
		Summary: summary,
		Results: resultData,
	}
}

// exportJSON exports data to JSON format
func (e *Exporter) exportJSON(data ExportData) error {
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(e.outputPath, fmt.Sprintf("crawl_results_%s.json", timestamp))

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create json file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Printf("✓ Results exported to JSON: %s\n", filename)
	return nil
}

// exportCSV exports data to CSV format
func (e *Exporter) exportCSV(data ExportData) error {
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(e.outputPath, fmt.Sprintf("crawl_results_%s.csv", timestamp))

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create csv file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"URL", "Depth", "Success", "Links Found", "Duration", "Timestamp", "Error"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	// Write results
	for _, r := range data.Results {
		row := []string{
			r.URL,
			fmt.Sprintf("%d", r.Depth),
			fmt.Sprintf("%t", r.Success),
			fmt.Sprintf("%d", r.LinksFound),
			r.Duration,
			r.Timestamp,
			r.Error,
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}

	fmt.Printf("✓ Results exported to CSV: %s\n", filename)
	return nil
}

// ExportURLList exports just the list of visited URLs
func (e *Exporter) ExportURLList(urls []string) error {
	if err := os.MkdirAll(e.outputPath, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(e.outputPath, fmt.Sprintf("visited_urls_%s.txt", timestamp))

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create urls file: %w", err)
	}
	defer file.Close()

	for _, url := range urls {
		if _, err := fmt.Fprintln(file, url); err != nil {
			return fmt.Errorf("write url: %w", err)
		}
	}

	fmt.Printf("✓ URL list exported: %s\n", filename)
	return nil
}