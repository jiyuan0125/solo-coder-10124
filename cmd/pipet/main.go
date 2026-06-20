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
	"github.com/bjesus/pipet/utils"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)
	cli.VersionFlag = &cli.BoolFlag{
		Name:  "version",
		Usage: "print the pipet version",
	}
	application := &cli.App{
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
			&cli.StringFlag{
				Name:    "template",
				Aliases: []string{"t"},
				Usage:   "path to file for template output",
			},
			&cli.BoolFlag{
				Name:    "csv",
				Aliases: []string{"C"},
				Usage:   "output as CSV, one record per line",
			},
			&cli.StringSliceFlag{
				Name:    "csv-header",
				Aliases: []string{"H"},
				Usage:   "column name for CSV output, can be used multiple times",
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
			&cli.StringFlag{
				Name:    "block",
				Aliases: []string{"b"},
				Usage:   "only run blocks with names matching this pattern (supports exact match and wildcards)",
			},
			&cli.BoolFlag{
				Name:    "stable-fingerprint",
				Aliases: []string{"f"},
				Usage:   "use stable fingerprint comparison for on-change (field order, whitespace, quote case insensitive)",
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

	if err := application.Run(os.Args); err != nil {
		log.Println("pipet version", application.Version)
		log.Fatal(err)
	}
}

func runPipet(c *cli.Context, specFile string) error {
	jsonOutput := c.Bool("json")
	csvOutput := c.Bool("csv")
	csvHeader := c.StringSlice("csv-header")
	separators := c.StringSlice("separator")
	templateFile := c.String("template")
	onChange := c.String("on-change")
	maxPages := c.Int("max-pages")
	interval := c.Int("interval")
	verbose := c.Bool("verbose")
	blockFilter := c.String("block")
	stableFingerprint := c.Bool("stable-fingerprint")

	if !verbose {
		log.SetOutput(io.Discard)
	}

	automaticTemplateFile := strings.TrimSuffix(specFile, filepath.Ext(specFile)) + ".tpl"

	if !jsonOutput && !csvOutput && templateFile == "" && utils.FileExists(automaticTemplateFile) {
		log.Println("Detected template file at", specFile)
		templateFile = automaticTemplateFile
	}

	pipet := &common.PipetApp{
		MaxPages:          maxPages,
		Separator:         separators,
		CSVHeader:         csvHeader,
		StableFingerprint: stableFingerprint,
	}

	log.Println("Parsing pipet file:", specFile)
	err := app.ParseSpecFile(pipet, specFile)
	if err != nil {
		return fmt.Errorf("error parsing spec file: %w", err)
	}

	defer app.CleanupPlaywrightSession()

	iterate := true
	previousValue := ""
	isFirstRun := true

	for iterate {
		newValue := ""
		log.Println("Executing blocks")
		err = app.ExecuteBlocks(pipet, blockFilter)
		if err != nil {
			return fmt.Errorf("error executing blocks: %w", err)
		}

		log.Println("Generating output")

		if jsonOutput {
			newValue = outputs.OutputJSON(pipet)
		} else if csvOutput {
			csvStr, csvErr := outputs.OutputCSV(pipet)
			if csvErr != nil {
				return csvErr
			}
			newValue = csvStr
		} else if templateFile != "" {
			newValue = outputs.OutputTemplate(pipet, templateFile)
		} else {
			newValue = outputs.OutputText(pipet)
		}

		fmt.Print(newValue)

		if interval > 0 {
			if onChange != "" && !isFirstRun {
				changed := false
				if stableFingerprint {
					changed = utils.StableFingerprint(previousValue) != utils.StableFingerprint(newValue)
				} else {
					changed = previousValue != newValue
				}
				if changed {
					command := strings.ReplaceAll(onChange, "{}", utils.BashQuote(newValue))
					log.Println("Executing on change command: " + command)
					cmd := exec.Command("bash", "-c", command)
					cmd.Output()
				}
			}
			previousValue = newValue
			isFirstRun = false
			pipet.Data = []interface{}{}
			time.Sleep(time.Duration(interval) * time.Second)
		} else {
			iterate = false
		}
	}
	return nil
}
