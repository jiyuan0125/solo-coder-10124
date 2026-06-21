package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/bjesus/pipet/common"
	"github.com/bjesus/pipet/internal/app"
	"github.com/bjesus/pipet/outputs"
	"github.com/bjesus/pipet/parsers"
	"github.com/bjesus/pipet/utils"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)
	cli.VersionFlag = &cli.BoolFlag{
		Name:  "version",
		Usage: "print the pipet version",
	}
	app := &cli.App{
		Name:                   "pipet",
		Usage:                  "swiss-army tool for web scraping, made for hackers",
		HideHelpCommand:        true,
		UseShortOptionHandling: true,
		EnableBashCompletion:   true,
		Version:                "0.3.0",
		ArgsUsage:              "<pipet_file>",

		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "output as JSON",
			},
			&cli.BoolFlag{
				Name:    "lines",
				Aliases: []string{"l"},
				Usage:   "output as CSV lines, one record per line",
			},
			&cli.StringSliceFlag{
				Name:    "header",
				Aliases: []string{"d"},
				Usage:   "column header for lines output (can be used multiple times)",
			},
			&cli.StringFlag{
				Name:    "block",
				Aliases: []string{"b"},
				Usage:   "only run blocks with names matching this pattern (supports wildcards)",
			},
			&cli.BoolFlag{
				Name:    "stable-diff",
				Aliases: []string{"f"},
				Usage:   "use stable fingerprint for on-change comparison (ignores whitespace, key order, etc.)",
			},
			&cli.StringFlag{
				Name:    "template",
				Aliases: []string{"t"},
				Usage:   "path to file for template output",
			},
			&cli.StringSliceFlag{
				Name:    "separator",
				Aliases: []string{"s"},
				Usage:   "set a separator for text output (can be used multiple times)",
			},
			&cli.IntFlag{
				Name:    "max-pages",
				Value:   3,
				Aliases: []string{"p"},
				Usage:   "maximum number of pages to scrape",
			},
			&cli.IntFlag{
				Name:    "interval",
				Value:   0,
				Aliases: []string{"i"},
				Usage:   "rerun pipet after X seconds, 0 to disable",
			},
			&cli.StringFlag{
				Name:    "on-change",
				Aliases: []string{"c"},
				Usage:   "a command to run when the pipet result is new",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "enable verbose logging",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("pipet file argument is required")
			}
			spec := c.Args().Get(0)
			return runPipet(c, spec)
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "pipet version", app.Version)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runPipet(c *cli.Context, specFile string) error {
	jsonOutput := c.Bool("json")
	linesOutput := c.Bool("lines")
	headers := c.StringSlice("header")
	blockPattern := c.String("block")
	stableDiff := c.Bool("stable-diff")
	separators := c.StringSlice("separator")
	templateFile := c.String("template")
	onChange := c.String("on-change")
	maxPages := c.Int("max-pages")
	interval := c.Int("interval")
	verbose := c.Bool("verbose")

	if !verbose {
		log.SetOutput(io.Discard)
	}

	automaticTemplateFile := strings.TrimSuffix(specFile, filepath.Ext(specFile)) + ".tpl"

	if !jsonOutput && !linesOutput && templateFile == "" && utils.FileExists(automaticTemplateFile) {
		log.Println("Detected template file at", specFile)
		templateFile = automaticTemplateFile
	}

	pipet := &common.PipetApp{
		MaxPages:  maxPages,
		Separator: separators,
	}

	log.Println("Parsing pipet file:", specFile)
	err := app.ParseSpecFile(pipet, specFile)
	if err != nil {
		return fmt.Errorf("error parsing spec file: %w", err)
	}

	if blockPattern != "" {
		log.Println("Filtering blocks by pattern:", blockPattern)
		app.FilterBlocks(pipet, blockPattern)
	}

	hasPlaywright := false
	for _, block := range pipet.Blocks {
		if block.Type == "playwright" {
			hasPlaywright = true
			break
		}
	}

	var browserCtx *parsers.BrowserContext
	if interval > 0 && hasPlaywright {
		log.Println("Initializing persistent browser for polling")
		browserCtx, err = parsers.InitBrowser()
		if err != nil {
			return fmt.Errorf("error initializing browser: %w", err)
		}
		defer browserCtx.Close()
	}

	iterate := true
	previousValue := ""
	previousFingerprint := ""
	isFirstIteration := true

	for iterate {
		newValue := ""
		log.Println("Executing blocks")
		err = app.ExecuteBlocks(pipet, browserCtx)
		if err != nil {
			return fmt.Errorf("error executing blocks: %w", err)
		}

		log.Println("Generating output")

		if jsonOutput {
			newValue = outputs.OutputJSON(pipet)
		} else if linesOutput {
			var linesErr error
			newValue, linesErr = outputs.OutputLines(pipet, headers)
			if linesErr != nil {
				return fmt.Errorf("error generating lines output: %w", linesErr)
			}
		} else if templateFile != "" {
			newValue = outputs.OutputTemplate(pipet, templateFile)
		} else {
			newValue = outputs.OutputText(pipet)
		}

		fmt.Print(newValue)

		if interval > 0 {
			if onChange != "" && !isFirstIteration {
				changed := false
				if stableDiff {
					currentFingerprint := utils.StableFingerprint(pipet.Data)
					if previousFingerprint != "" && currentFingerprint != previousFingerprint {
						changed = true
					}
					previousFingerprint = currentFingerprint
				} else {
					if previousValue != "" && previousValue != newValue {
						changed = true
					}
					previousValue = newValue
				}

				if changed {
					command := strings.ReplaceAll(onChange, "{}", utils.BashEscape(newValue))
					log.Println("Executing on change command: " + command)
					cmd := exec.Command("bash", "-c", command)
					cmd.Output()
				}
			} else if isFirstIteration {
				if stableDiff {
					previousFingerprint = utils.StableFingerprint(pipet.Data)
				} else {
					previousValue = newValue
				}
			}
			isFirstIteration = false
			pipet.Data = []interface{}{}
			time.Sleep(time.Duration(interval) * time.Second)
		} else {
			iterate = false
		}
	}
	return nil
}
