package parsers

import (
	"fmt"
	"log"
	"strings"

	"github.com/bjesus/pipet/common"
	"github.com/playwright-community/playwright-go"
)

var sharedPlaywright *playwright.Playwright
var sharedBrowser playwright.Browser

func getSharedBrowser() (playwright.Browser, error) {
	if sharedBrowser != nil {
		return sharedBrowser, nil
	}

	err := playwright.Install()
	if err != nil {
		return nil, fmt.Errorf("failed to install playwright: %w", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to start playwright: %w", err)
	}
	sharedPlaywright = pw

	browser, err := pw.Chromium.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}
	sharedBrowser = browser

	log.Println("Playwright browser started and ready for reuse")
	return sharedBrowser, nil
}

func CloseSharedBrowser() {
	if sharedBrowser != nil {
		sharedBrowser.Close()
		sharedBrowser = nil
	}
	if sharedPlaywright != nil {
		sharedPlaywright.Stop()
		sharedPlaywright = nil
	}
}

func ExecutePlaywrightBlock(block common.Block) (interface{}, error) {
	browser, err := getSharedBrowser()
	if err != nil {
		return nil, err
	}

	page, err := browser.NewPage()
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
