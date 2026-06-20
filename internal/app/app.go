package app

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/bjesus/pipet/common"
	"github.com/bjesus/pipet/parsers"
	"github.com/bjesus/pipet/utils"
	"github.com/google/shlex"
)

var (
	pwOnce     sync.Once
	pwInstance *parsers.PlaywrightSession
	pwErr      error
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
			trimmed := strings.TrimSpace(strings.TrimPrefix(line, "//"))
			if trimmed != "" && currentBlock == nil {
				pendingName = trimmed
			}
			continue
		}
		if currentBlock == nil {
			if strings.HasPrefix(line, "curl ") {
				currentBlock = &common.Block{Type: "curl", Command: line, Name: pendingName}
			} else if strings.HasPrefix(line, "playwright ") {
				currentBlock = &common.Block{Type: "playwright", Command: line, Name: pendingName}
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

func filterBlocks(blocks []common.Block, pattern string) []common.Block {
	if pattern == "" {
		return blocks
	}
	var filtered []common.Block
	for _, block := range blocks {
		if utils.MatchWildcard(pattern, block.Name) {
			filtered = append(filtered, block)
		}
	}
	return filtered
}

func ExecuteBlocks(e *common.PipetApp, blockFilter string) error {
	blocks := filterBlocks(e.Blocks, blockFilter)
	for _, block := range blocks {
		var data interface{}
		var err error
		var nextPageURL string

		for page := 0; page < e.MaxPages; page++ {
			if block.Type == "curl" {
				data, nextPageURL, err = parsers.ExecuteCurlBlock(block)
			} else if block.Type == "playwright" {
				session, sErr := getPlaywrightSession()
				if sErr != nil {
					return sErr
				}
				data, err = parsers.ExecutePlaywrightBlockWithSession(session, block)
			}

			if err != nil {
				return err
			}

			e.Data = append(e.Data, data)

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

func getPlaywrightSession() (*parsers.PlaywrightSession, error) {
	pwOnce.Do(func() {
		pwInstance, pwErr = parsers.InitPlaywrightSession()
	})
	return pwInstance, pwErr
}

func CleanupPlaywrightSession() {
	if pwInstance != nil {
		pwInstance.Close()
	}
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

	fullURL := baseURL.ResolveReference(refURL)

	return fullURL.String()
}
