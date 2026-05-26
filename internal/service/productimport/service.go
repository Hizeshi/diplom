package productimportsvc

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"iq-home/backend/internal/domain/productimport"
)

// Expected column headers (case-insensitive). Order in the file doesn't matter.
var columnAliases = map[string]string{
	"article":      "article",
	"артикул":      "article",
	"name":         "name",
	"название":     "name",
	"наименование": "name",
	"price":        "price",
	"цена":         "price",
	"type":         "type",
	"product_type": "type",
	"тип":          "type",
	"brand":        "brand",
	"бренд":        "brand",
	"series":       "series",
	"серия":        "series",
	"color":        "color",
	"цвет":         "color",
	"description":  "description",
	"описание":     "description",
}

type repo interface {
	Upsert(ctx context.Context, row productimport.Row) error
}

type Service struct {
	repo repo
}

func New(repo repo) *Service {
	return &Service{repo: repo}
}

// ImportCSV parses a CSV file and upserts products. Delimiter is detected automatically.
func (s *Service) ImportCSV(ctx context.Context, data []byte) (*productimport.Result, error) {
	rows, err := parseCSV(data)
	if err != nil {
		return nil, fmt.Errorf("productimport: parse csv: %w", err)
	}
	return s.importRows(ctx, rows)
}

// ImportExcel parses the first sheet of an xlsx file and upserts products.
func (s *Service) ImportExcel(ctx context.Context, data []byte) (*productimport.Result, error) {
	rows, err := parseExcel(data)
	if err != nil {
		return nil, fmt.Errorf("productimport: parse excel: %w", err)
	}
	return s.importRows(ctx, rows)
}

func (s *Service) importRows(ctx context.Context, rows []productimport.Row) (*productimport.Result, error) {
	result := &productimport.Result{Total: len(rows)}
	for i, row := range rows {
		if row.Article == "" || row.Name == "" {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: article and name are required", i+1))
			continue
		}
		if err := s.repo.Upsert(ctx, row); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("row %d (%s): %v", i+1, row.Article, err))
			continue
		}
		result.Upserted++
	}
	return result, nil
}

// ─── Parsers ─────────────────────────────────────────────────────────────────

func parseCSV(data []byte) ([]productimport.Row, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	// Detect delimiter by scanning the first line
	if first := bytes.IndexByte(data, '\n'); first > 0 {
		line := string(data[:first])
		if strings.Count(line, ";") > strings.Count(line, ",") {
			r.Comma = ';'
		}
	}

	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("file is empty or has no data rows")
	}
	return mapRecords(records[0], records[1:])
}

func parseExcel(data []byte) ([]productimport.Row, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("excel file has no sheets")
	}

	rawRows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}
	if len(rawRows) < 2 {
		return nil, fmt.Errorf("sheet is empty or has no data rows")
	}
	return mapRecords(rawRows[0], rawRows[1:])
}

// mapRecords converts raw string rows into domain rows using the header map.
func mapRecords(header []string, rows [][]string) ([]productimport.Row, error) {
	// Build column index: canonical_name → column index
	idx := make(map[string]int)
	for i, h := range header {
		key := strings.ToLower(strings.TrimSpace(h))
		if canonical, ok := columnAliases[key]; ok {
			idx[canonical] = i
		}
	}
	if _, ok := idx["article"]; !ok {
		return nil, fmt.Errorf("column 'article' (артикул) not found in header")
	}
	if _, ok := idx["name"]; !ok {
		return nil, fmt.Errorf("column 'name' (название) not found in header")
	}

	col := func(row []string, name string) string {
		i, ok := idx[name]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	var result []productimport.Row
	for _, row := range rows {
		// Skip fully empty rows
		empty := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}

		price, _ := strconv.ParseFloat(strings.ReplaceAll(col(row, "price"), " ", ""), 64)

		result = append(result, productimport.Row{
			Article:     col(row, "article"),
			Name:        col(row, "name"),
			Price:       price,
			ProductType: col(row, "type"),
			Brand:       col(row, "brand"),
			Series:      col(row, "series"),
			Color:       col(row, "color"),
			Description: col(row, "description"),
		})
	}
	return result, nil
}
