package excel

import (
	"bytes"
	"encoding/csv"
	"errors"
	"strings"

	"github.com/xuri/excelize/v2"
)

func ParseFile(data []byte, ext string) (headers []string, rows [][]string, err error) {
	switch strings.ToLower(ext) {
	case ".xlsx", ".xlsm", ".xltx", ".xltm":
		return parseXLSX(data)
	case ".csv":
		return parseCSV(data)
	default:
		return nil, nil, errors.New("unsupported file format: " + ext)
	}
}

func parseXLSX(data []byte) (headers []string, rows [][]string, err error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, errors.New("failed to parse xlsx file")
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, nil
	}

	sheetName := sheets[0]
	rowIter, err := f.Rows(sheetName)
	if err != nil {
		return nil, nil, errors.New("failed to read xlsx rows")
	}
	defer rowIter.Close()

	isFirst := true
	for rowIter.Next() {
		cells, err := rowIter.Columns()
		if err != nil {
			continue
		}

		if len(cells) == 0 {
			continue
		}

		if isFirst {
			headers = cells
			isFirst = false
			continue
		}

		row := make([]string, len(headers))
		for i := range headers {
			if i < len(cells) {
				row[i] = strings.TrimSpace(cells[i])
			}
		}
		rows = append(rows, row)
	}

	return headers, rows, nil
}

func parseCSV(data []byte) (headers []string, rows [][]string, err error) {
	// Strip UTF-8 BOM if present (Excel exports often include this)
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	delimiter := detectDelimiter(data)

	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = delimiter
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, errors.New("failed to parse csv file: " + err.Error())
	}

	if len(records) == 0 {
		return nil, nil, nil
	}

	headers = records[0]
	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
	}

	for _, record := range records[1:] {
		if isEmptyRow(record) {
			continue
		}
		row := make([]string, len(headers))
		for i := range headers {
			if i < len(record) {
				row[i] = strings.TrimSpace(record[i])
			}
		}
		rows = append(rows, row)
	}

	return headers, rows, nil
}

func isEmptyRow(record []string) bool {
	for _, cell := range record {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func detectDelimiter(data []byte) rune {
	firstLine := ""
	lines := bytes.Split(data, []byte("\n"))
	if len(lines) > 0 {
		firstLine = string(lines[0])
	}

	tabCount := strings.Count(firstLine, "\t")
	semicolCount := strings.Count(firstLine, ";")
	commaCount := strings.Count(firstLine, ",")

	if tabCount > commaCount && tabCount > semicolCount {
		return '\t'
	}
	if semicolCount > commaCount {
		return ';'
	}
	return ','
}
