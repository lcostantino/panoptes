package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"

	"github.com/lcostantino/Panoptes/panoptes"
	"github.com/logrusorgru/aurora"
)

var version = "replace"

type PanoptesArgs struct {
	verbose    bool
	configFile string
	logFile    string
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

//Similar to https://github.com/bi-zone/etw/blob/master/examples/tracer/main.go
func consumer(eventChan chan panoptes.Event, errorChan chan error, ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			GLogger.Info().Msg("Context done")
			return
		case e, ok := <-eventChan:
			if !ok {
				return
			}

			jdata, _ := json.Marshal(e.EventData)
			GLogger.Info().RawJSON("etwEvent", jdata).Str("name", e.Name).Str("guid", e.Guid).Msg("Data")

		case err := <-errorChan:
			GLogger.Error().Err(err).Msg("Error consuming event")
			return

		}
	}

}

//This can be done with channels too
func errorCbk(err error) {
	GLogger.Error().Err(err).Msg("Failed to process event")

}

func stopApplication(cancelFnc context.CancelFunc, wg *sync.WaitGroup, c *panoptes.Client) {
	c.Stop()
	cancelFnc()
	wg.Wait()
	os.Exit(0)
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
			GLogger.Error().Err(err).Str("guid", r.Guid).Str("name", r.Name).Msg("Failed to add provider")
		} else {
			GLogger.Info().Str("guid", r.Guid).Str("name", r.Name).Msg("Registered")
		}
	}

	eventChan := make(chan panoptes.Event, 10)
	errorChan := make(chan error)

	//just in case
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer close(eventChan)
	defer close(errorChan)

	go func() {
		consumer(eventChan, errorChan, ctx)
		wg.Done()
	}()
	wg.Add(1)

	if err := client.Start(eventChan, errorChan); err != nil {
		GLogger.Error().Err(err).Msg("Failed to start")
		stopApplication(cancel, &wg, client)

	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	for range sigCh {
		GLogger.Info().Msg("Shutting the session down")
		stopApplication(cancel, &wg, client)

	}
	client.Pull()

}
