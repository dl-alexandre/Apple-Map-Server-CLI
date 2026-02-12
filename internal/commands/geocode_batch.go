package commands

import (
	"bufio"
	"io"
	"strings"
)

func readQueries(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	buffer := make([]byte, 0, 64*1024)
	scanner.Buffer(buffer, 1024*1024)

	var queries []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		queries = append(queries, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return queries, nil
}
