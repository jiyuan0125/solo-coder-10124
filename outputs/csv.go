package outputs

import (
	"encoding/csv"
	"fmt"
	"reflect"
	"strings"

	"github.com/bjesus/pipet/common"
)

func OutputCSV(app *common.PipetApp) (string, error) {
	var sb strings.Builder
	w := csv.NewWriter(&sb)

	hasHeader := len(app.CSVHeader) > 0

	if hasHeader {
		if err := w.Write(app.CSVHeader); err != nil {
			return "", fmt.Errorf("failed to write CSV header: %w", err)
		}
	}

	for i, blockData := range app.Data {
		records := flattenToRecords(blockData)
		for _, record := range records {
			if hasHeader && len(record) != len(app.CSVHeader) {
				return "", fmt.Errorf("CSV field count mismatch: header has %d fields, but record has %d fields: %v",
					len(app.CSVHeader), len(record), record)
			}
			if err := w.Write(record); err != nil {
				return "", fmt.Errorf("failed to write CSV record: %w", err)
			}
		}

		if i < len(app.Data)-1 && len(records) > 0 {
			w.Flush()
			sb.WriteString("\n")
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return "", fmt.Errorf("CSV writing error: %w", err)
	}

	return sb.String(), nil
}

func flattenToRecords(data interface{}) [][]string {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		return [][]string{{fmt.Sprint(data)}}
	}

	if v.Len() == 0 {
		return nil
	}

	firstElem := v.Index(0)
	if firstElem.Kind() == reflect.Interface {
		firstElem = firstElem.Elem()
	}

	if firstElem.Kind() == reflect.Slice {
		var result [][]string
		for i := 0; i < v.Len(); i++ {
			row := flattenToRow(v.Index(i).Interface())
			if len(row) > 0 {
				result = append(result, row)
			}
		}
		return result
	}

	singleRow := flattenToRow(data)
	if len(singleRow) > 0 {
		return [][]string{singleRow}
	}
	return nil
}

func flattenToRow(data interface{}) []string {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		return []string{fmt.Sprint(data)}
	}

	var row []string
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i).Interface()
		elemV := reflect.ValueOf(elem)
		if elemV.Kind() == reflect.Slice {
			row = append(row, flattenToRow(elem)...)
		} else {
			row = append(row, fmt.Sprint(elem))
		}
	}
	return row
}
