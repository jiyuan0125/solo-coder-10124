package parsers

import (
	"fmt"
	"strings"

	"github.com/bjesus/pipet/common"
	"github.com/playwright-community/playwright-go"
)

type PlaywrightSession struct {
	PW      *playwright.Playwright
	Browser playwright.Browser
}

func InitPlaywrightSession() (*PlaywrightSession, error) {
	err := playwright.Install()
	if err != nil {
		return nil, fmt.Errorf("failed to install playwright: %w", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to start playwright: %w", err)
	}

	browser, err := pw.Chromium.Launch()
	if err != nil {
		pw.Stop()
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	return &PlaywrightSession{PW: pw, Browser: browser}, nil
}

func (s *PlaywrightSession) Close() {
	if s.Browser != nil {
		s.Browser.Close()
	}
	if s.PW != nil {
		s.PW.Stop()
	}
}

func ExecutePlaywrightBlock(block common.Block) (interface{}, error) {
	session, err := InitPlaywrightSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()
	return ExecutePlaywrightBlockWithSession(session, block)
}

func ExecutePlaywrightBlockWithSession(session *PlaywrightSession, block common.Block) (interface{}, error) {
	page, err := session.Browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create new page: %w", err)
	}
	defer page.Close()

	var url string

	switch cmd := block.Command.(type) {
	case string:
		url = strings.TrimPrefix(cmd, "playwright ")
	default:
	}

	_, err = page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	var result []interface{}

	for _, query := range block.Queries {
		parts := strings.Split(query, "|")
		jsQuery := strings.TrimSpace(parts[0])

		value, err := page.Evaluate(jsQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate JavaScript: %w", err)
		}

		if len(parts) > 1 {
			pipedValue, err := ExecutePipe(fmt.Sprintf("%v", value), strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("failed to execute pipe: %w", err)
			}
			result = append(result, pipedValue)
		} else {
			result = append(result, value)
		}
	}

	return result, nil
}
