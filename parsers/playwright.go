package parsers

import (
	"fmt"
	"strings"

	"github.com/bjesus/pipet/common"
	"github.com/playwright-community/playwright-go"
)

type BrowserContext struct {
	Pw      *playwright.Playwright
	Browser playwright.Browser
}

func InitBrowser() (*BrowserContext, error) {
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

	return &BrowserContext{Pw: pw, Browser: browser}, nil
}

func (bc *BrowserContext) Close() {
	if bc == nil {
		return
	}
	if bc.Browser != nil {
		bc.Browser.Close()
		bc.Browser = nil
	}
	if bc.Pw != nil {
		bc.Pw.Stop()
		bc.Pw = nil
	}
}

func ExecutePlaywrightBlock(block common.Block, browserCtx *BrowserContext) (interface{}, error) {
	var pw *playwright.Playwright
	var browser playwright.Browser
	var page playwright.Page
	var err error
	usingShared := false

	if browserCtx != nil && browserCtx.Pw != nil && browserCtx.Browser != nil {
		if !browserCtx.Browser.IsConnected() {
			browserCtx.Pw = nil
			browserCtx.Browser = nil
		} else {
			pw = browserCtx.Pw
			browser = browserCtx.Browser
			usingShared = true
		}
	}
	if !usingShared {
		err = playwright.Install()
		if err != nil {
			return nil, fmt.Errorf("failed to install playwright: %w", err)
		}

		pw, err = playwright.Run()
		if err != nil {
			return nil, fmt.Errorf("failed to start playwright: %w", err)
		}
		defer pw.Stop()

		browser, err = pw.Chromium.Launch()
		if err != nil {
			return nil, fmt.Errorf("failed to launch browser: %w", err)
		}
		defer browser.Close()
	}

	page, err = browser.NewPage()
	if err != nil {
		if usingShared {
			browserCtx.Pw = nil
			browserCtx.Browser = nil
		}
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
		if usingShared {
			browserCtx.Pw = nil
			browserCtx.Browser = nil
		}
		return nil, fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	var result []interface{}

	for _, query := range block.Queries {
		parts := strings.Split(query, "|")
		jsQuery := strings.TrimSpace(parts[0])

		value, err := page.Evaluate(jsQuery)
		if err != nil {
			if usingShared {
				browserCtx.Pw = nil
				browserCtx.Browser = nil
			}
			return nil, fmt.Errorf("failed to evaluate JavaScript: %w", err)
		}

		if len(parts) > 1 {
			pipedValue, err := ExecutePipe(fmt.Sprintf("%v", value), strings.TrimSpace(parts[1]))
			if err != nil {
				if usingShared {
					browserCtx.Pw = nil
					browserCtx.Browser = nil
				}
				return nil, fmt.Errorf("failed to execute pipe: %w", err)
			}
			result = append(result, pipedValue)
		} else {
			result = append(result, value)
		}
	}

	return result, nil
}
