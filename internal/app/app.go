package app

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/bjesus/pipet/common"
	"github.com/bjesus/pipet/parsers"
	"github.com/google/shlex"
	"github.com/tidwall/match"
)

func ParseSpecFile(e *common.PipetApp, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentBlock *common.Block
	pendingName := ""

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			if currentBlock != nil {
				e.Blocks = append(e.Blocks, *currentBlock)
				currentBlock = nil
			}
			pendingName = ""
			continue
		}

		if strings.HasPrefix(line, "//") {
			commentText := strings.TrimSpace(strings.TrimPrefix(line, "//"))
			if strings.HasPrefix(commentText, "name:") {
				pendingName = strings.TrimSpace(strings.TrimPrefix(commentText, "name:"))
			}
			continue
		}
		if currentBlock == nil {
			blockName := pendingName
			if strings.HasPrefix(line, "curl ") {
				currentBlock = &common.Block{Name: blockName, Type: "curl", Command: line}
			} else if strings.HasPrefix(line, "playwright ") {
				currentBlock = &common.Block{Name: blockName, Type: "playwright", Command: line}
			} else {
				return fmt.Errorf("invalid block start: %s", line)
			}
			pendingName = ""
		} else {
			if strings.HasPrefix(line, "> ") {

				currentBlock.NextPage = strings.TrimPrefix(line, ">")
			} else {
				currentBlock.Queries = append(currentBlock.Queries, line)
			}
		}
	}

	log.Println("Found block", currentBlock)
	if currentBlock != nil {
		e.Blocks = append(e.Blocks, *currentBlock)
	}

	return scanner.Err()
}

func FilterBlocks(e *common.PipetApp, patterns string) {
	if patterns == "" {
		return
	}

	rawPatterns := strings.Split(patterns, ",")
	var patternList []string
	for _, p := range rawPatterns {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			patternList = append(patternList, trimmed)
		}
	}

	if len(patternList) == 0 {
		return
	}

	var filtered []common.Block
	for _, block := range e.Blocks {
		matched := false
		for _, pattern := range patternList {
			if match.Match(block.Name, pattern) {
				matched = true
				break
			}
		}
		if matched {
			filtered = append(filtered, block)
		}
	}
	e.Blocks = filtered
}

func ExecuteBlocks(e *common.PipetApp, browserCtx *parsers.BrowserContext) error {
	for _, block := range e.Blocks {
		var data interface{}
		var err error
		var nextPageURL string

		for page := 0; page < e.MaxPages; page++ {
			if block.Type == "curl" {
				data, nextPageURL, err = parsers.ExecuteCurlBlock(block)
			} else if block.Type == "playwright" {
				data, err = parsers.ExecutePlaywrightBlock(block, browserCtx)
			}

			if err != nil {
				return err
			}

			e.Data = append(e.Data, data)
			e.BlockNames = append(e.BlockNames, block.Name)

			if nextPageURL == "" {
				break
			}
			var parts []string
			switch cmd := block.Command.(type) {
			case string:
				parts, _ = shlex.Split(cmd)
			case []string:
				parts = cmd
			default:
			}

			for i, u := range parts {
				if len(u) >= 4 && u[:4] == "http" {
					parts[i] = concatenateURLs(parts[i], nextPageURL)
					break
				}
			}

			block.Command = parts
		}
	}

	return nil
}

func concatenateURLs(base, ref string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		panic(err)
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		panic(err)
	}

	// Resolve reference URL relative to the base URL
	fullURL := baseURL.ResolveReference(refURL)

	return fullURL.String()
}
