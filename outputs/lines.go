package outputs

import (
	"encoding/csv"
	"fmt"
	"reflect"
	"strings"

	"github.com/bjesus/pipet/common"
)

func OutputLines(app *common.PipetApp, headers []string) (string, error) {
	var result strings.Builder
	headerCount := len(headers)

	blocks := collectFinestRecords(app.Data)

	for bi, block := range blocks {
		if bi > 0 {
			result.WriteString("\n")
		}

		if headerCount > 0 && bi == 0 {
			headerLine, err := formatCSVLine(headers)
			if err != nil {
				return "", err
			}
			result.WriteString(headerLine)
			result.WriteString("\n")
		}

		for _, record := range block {
			fieldCount := len(record)
			if headerCount > 0 && fieldCount != headerCount {
				return "", fmt.Errorf("record has %d fields, but header specifies %d fields: %v", fieldCount, headerCount, record)
			}

			line, err := formatCSVLine(record)
			if err != nil {
				return "", err
			}
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	return result.String(), nil
}

func formatCSVLine(fields []string) (string, error) {
	var buf strings.Builder
	w := csv.NewWriter(&buf)
	err := w.Write(fields)
	if err != nil {
		return "", err
	}
	w.Flush()
	return strings.TrimRight(buf.String(), "\r\n"), nil
}

func collectFinestRecords(data interface{}) [][][]string {
	var result [][][]string

	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Slice {
		return result
	}

	for i := 0; i < val.Len(); i++ {
		blockData := val.Index(i).Interface()
		records := collectRecordsFromBlock(blockData)
		if len(records) > 0 {
			result = append(result, records)
		}
	}

	return result
}

func collectRecordsFromBlock(data interface{}) [][]string {
	var records [][]string
	collectRecords(data, &records)
	return records
}

func collectRecords(data interface{}, records *[][]string) {
	if data == nil {
		return
	}

	val := reflect.ValueOf(data)
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		if val.Len() == 0 {
			return
		}

		allStrings := true
		for i := 0; i < val.Len(); i++ {
			elem := val.Index(i).Interface()
			if _, ok := elem.(string); !ok {
				elemVal := reflect.ValueOf(elem)
				if elemVal.Kind() == reflect.Slice || elemVal.Kind() == reflect.Array {
					allStrings = false
					break
				}
			}
		}

		if allStrings {
			var record []string
			for i := 0; i < val.Len(); i++ {
				record = append(record, fmt.Sprintf("%v", val.Index(i).Interface()))
			}
			*records = append(*records, record)
		} else {
			for i := 0; i < val.Len(); i++ {
				collectRecords(val.Index(i).Interface(), records)
			}
		}
	default:
		*records = append(*records, []string{fmt.Sprintf("%v", data)})
	}
}
