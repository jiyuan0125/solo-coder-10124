package outputs

import (
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/bjesus/pipet/common"
)

func OutputCSV(app *common.PipetApp) (string, error) {
	var result strings.Builder
	headers := app.CSVHeaders
	headerCount := len(headers)

	trimmedHeaders := make([]string, headerCount)
	for i, h := range headers {
		trimmedHeaders[i] = strings.TrimSpace(h)
	}

	if len(app.Data) == 0 && headerCount > 0 {
		headerLine, err := formatCSVLine(trimmedHeaders)
		if err != nil {
			return "", err
		}
		return headerLine + "\n", nil
	}

	blocks := collectFinestRecords(app.Data)

	for bi, block := range blocks {
		if bi > 0 {
			result.WriteString("\n")
		}

		if headerCount > 0 && bi == 0 {
			headerLine, err := formatCSVLine(trimmedHeaders)
			if err != nil {
				return "", err
			}
			result.WriteString(headerLine)
			result.WriteString("\n")
		}

		var blockIdentifier string
		if bi < len(app.BlockNames) && app.BlockNames[bi] != "" {
			blockIdentifier = fmt.Sprintf("block %q", app.BlockNames[bi])
		} else {
			blockIdentifier = fmt.Sprintf("block #%d", bi+1)
		}

		for _, record := range block {
			fieldCount := len(record)
			if headerCount > 0 && fieldCount != headerCount {
				errMsg := fmt.Sprintf("%s: record has %d fields, but header specifies %d fields: %v", blockIdentifier, fieldCount, headerCount, record)
				fmt.Fprintln(os.Stderr, errMsg)
				return "", fmt.Errorf(errMsg)
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
	processed := make([]string, len(fields))
	for i, f := range fields {
		processed[i] = sanitizeCSVField(f)
	}

	var buf strings.Builder
	w := csv.NewWriter(&buf)
	err := w.Write(processed)
	if err != nil {
		return "", err
	}
	w.Flush()
	return strings.TrimRight(buf.String(), "\r\n"), nil
}

func sanitizeCSVField(field string) string {
	if field == "" {
		return field
	}

	firstChar := field[0]
	if firstChar == '=' || firstChar == '+' || firstChar == '-' || firstChar == '@' {
		return "\t" + field
	}

	return field
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
