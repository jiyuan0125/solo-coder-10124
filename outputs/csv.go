package outputs

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/bjesus/pipet/common"
)

func OutputCSV(app *common.PipetApp) (string, error) {
	var result strings.Builder

	hasHeader := len(app.CSVHeader) > 0
	expectedFields := len(app.CSVHeader)

	if hasHeader {
		result.WriteString(formatCSVRow(app.CSVHeader))
		result.WriteString("\n")
	}

	for i, blockData := range app.Data {
		rows := extractCSVRows(blockData)

		for _, row := range rows {
			if hasHeader && len(row) != expectedFields {
				return "", fmt.Errorf("field count mismatch: header has %d fields, but record has %d fields (block %d)", expectedFields, len(row), i)
			}
			result.WriteString(formatCSVRow(row))
			result.WriteString("\n")
		}

		if i < len(app.Data)-1 {
			result.WriteString("\n")
		}
	}

	return result.String(), nil
}

func extractCSVRows(data interface{}) [][]string {
	v := reflect.ValueOf(data)

	if v.Kind() != reflect.Slice {
		return [][]string{{fmt.Sprint(data)}}
	}

	if v.Len() == 0 {
		return [][]string{}
	}

	firstElem := v.Index(0).Interface()
	firstVal := reflect.ValueOf(firstElem)

	if firstVal.Kind() == reflect.Slice {
		var result [][]string
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i).Interface()
			subRows := extractCSVRows(elem)
			result = append(result, subRows...)
		}
		return result
	}

	var row []string
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i).Interface()
		elemVal := reflect.ValueOf(elem)
		if elemVal.Kind() == reflect.Slice {
			subRows := extractCSVRows(elem)
			return subRows
		}
		row = append(row, fmt.Sprint(elem))
	}

	return [][]string{row}
}

func formatCSVRow(fields []string) string {
	var escaped []string
	for _, field := range fields {
		escaped = append(escaped, escapeCSVField(field))
	}
	return strings.Join(escaped, ",")
}

func escapeCSVField(field string) string {
	needsQuoting := strings.ContainsAny(field, ",\"\n\r")
	if !needsQuoting {
		return field
	}

	escaped := strings.ReplaceAll(field, "\"", "\"\"")
	return "\"" + escaped + "\""
}
