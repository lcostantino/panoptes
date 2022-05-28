package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/lcostantino/Panoptes/panoptes"
	"github.com/logrusorgru/aurora"
)

var version = "replace"

type PanoptesArgs struct {
	verbose    bool
	configFile string
}

func init() {
	NewLogger(false, true)
}

var au aurora.Aurora

//Very basic flag logic, use cobra, clapper , pflags , etc if really needed in the future
func parseCommandLineAndValidate() PanoptesArgs {

	args := PanoptesArgs{}

	flag.StringVar(&args.configFile, "config-file", "", "Config file for sensors")
	flag.BoolVar(&args.verbose, "verbose", false, "Verbose")
	flag.Parse()
	if args.configFile == "" {
		fmt.Println(au.Red("Error: You need to provide a valid config file\n"))
		os.Exit(1)
	}

	return args
}

func parseConfigFile(fName string) []panoptes.Provider {
	if data, err := os.ReadFile(fName); err != nil {
		GLogger.Error().Err(err).Msg("Failed to open config file")
		os.Exit(1)
	} else {
		mProviders := make([]panoptes.Provider, 10)
		if err := json.Unmarshal(data, &mProviders); err != nil {
			GLogger.Error().Err(err).Msg("Failed to parse config file")
			os.Exit(1)
		}
		return mProviders

	}
	return nil
	//register your own callback etc..
}

func main() {
	au = aurora.NewAurora(true)
	fmt.Println(au.Sprintf(au.Green("---- [ Panoptes Ver: %s ] ----\n"), au.BrightGreen(version)))
	args := parseCommandLineAndValidate()

	fmt.Println(args)
	client := panoptes.NewClient()

	providers := parseConfigFile(args.configFile)

	for _, r := range providers {
		if err := client.AddProvider(r); err != nil {
			GLogger.Error().Err(err).Msg("Failed to add provider")
		}
	}
}
