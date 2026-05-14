package cut

import (
	"strconv"
	"strings"
)

// Config конфигурация cut
type Config struct {
	Fields    []int
	Delimiter string
	Separated bool
}

// ProcessLine обработка строки
func ProcessLine(line string, config Config) (string, bool) {
	if config.Separated {
		if !strings.Contains(line, config.Delimiter) {
			return "", false
		}
	}

	parts := strings.Split(line, config.Delimiter)

	var result []string
	for _, fieldNum := range config.Fields {
		idx := fieldNum - 1
		if idx >= 0 && idx < len(parts) {
			result = append(result, parts[idx])
		}
	}

	if len(result) == 0 {
		return "", false
	}

	return strings.Join(result, config.Delimiter), true
}

// ProcessBatch обработка пачки
func ProcessBatch(lines []string, config Config) []string {
	var results []string
	for _, line := range lines {
		if processed, ok := ProcessLine(line, config); ok {
			results = append(results, processed)
		}
	}
	return results
}

// ParseFields записывает какие поля нужны
func ParseFields(fieldsStr string) []int {
	if fieldsStr == "" {
		return nil
	}

	var fields []int
	parts := strings.Split(fieldsStr, ",")

	for _, part := range parts {
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) == 2 {
				start, _ := strconv.Atoi(rangeParts[0])
				end, _ := strconv.Atoi(rangeParts[1])
				for i := start; i <= end; i++ {
					fields = append(fields, i)
				}
			}
		} else {
			num, _ := strconv.Atoi(part)
			fields = append(fields, num)
		}
	}

	return fields
}
