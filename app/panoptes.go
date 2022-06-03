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
	"github.com/lcostantino/go-duktape"
	"github.com/logrusorgru/aurora"
)

var version = "replace"

type PanoptesArgs struct {
	stdout         bool
	configFile     string
	logFile        string
	javascriptFile string
	consumers      int
}

func init() {

}

var au aurora.Aurora

//Very basic flag logic, use cobra, clapper , pflags , etc if really needed in the future
func parseCommandLineAndValidate() PanoptesArgs {

	args := PanoptesArgs{}

	flag.StringVar(&args.configFile, "config-file", "", "Config file for sensors")
	flag.BoolVar(&args.stdout, "stdout", true, "Print to stdout")
	flag.StringVar(&args.logFile, "log-file", "", "Log file")
	flag.StringVar(&args.javascriptFile, "js-file", "", "JS processor file")
	flag.IntVar(&args.consumers, "consumers", 1, "number of consumer routines")
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
func consumer(eventChan chan panoptes.Event, errorChan chan error, ctx context.Context, jsChan chan panoptes.Event) {

	for {
		select {
		case <-ctx.Done():
			GLogger.Info().Msg("Context done")
			return
		case e, ok := <-eventChan:
			if !ok {
				return
			}
			e.Marshalled, _ = json.Marshal(e.EventData)
			//If JS enabled let it decide wether to output or not the data
			if jsChan != nil {
				jsChan <- e
			} else {
				GLogger.Info().RawJSON("etwEvent", []byte(e.Marshalled)).Str("name", e.Name).Str("guid", e.Guid).Msg("Data")
			}
		case err := <-errorChan:
			GLogger.Error().Err(err).Msg("Error consuming event")
			return

		}
	}

}

//We can create multiples runtimes or in this case, just ONE to avoid locks and shared state for the moment.
func jsProcessor(eventChan chan panoptes.Event, jsCtx *duktape.Context, ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			return
		case e := <-eventChan:
			jsCtx.PushGlobalObject()
			jsCtx.GetPropString(-1, "panoptesProcess")
			jsCtx.PushString(string(e.Marshalled))
			if jsCtx.Pcall(1) == 0 {
				if str := jsCtx.SafeToString(-1); str != "" {
					GLogger.Info().RawJSON("etwEvent", []byte(str)).Str("name", e.Name).Str("guid", e.Guid).Msg("Data")
				}
			}
			jsCtx.Pop3()
		}
	}
}

func stopApplication(cancelFnc context.CancelFunc, wg *sync.WaitGroup, c *panoptes.Client) {
	cancelFnc()
	c.Stop()
	wg.Wait()
	os.Exit(0)
}

func main() {
	au = aurora.NewAurora(true)
	fmt.Println(au.Sprintf(au.Green("---- [ Panoptes Ver: %s ] ----\n"), au.BrightGreen(version)))
	args := parseCommandLineAndValidate()

	NewLogger(args.logFile, args.stdout)

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
	var jsChan chan panoptes.Event

	//just in case
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer close(eventChan)
	defer close(errorChan)

	if args.javascriptFile != "" {
		jsChan = make(chan panoptes.Event, args.consumers)
		jsCtx := duktape.New()
		jsCtx.PushTimers()
		if data, err := os.ReadFile(args.javascriptFile); err != nil {
			GLogger.Error().Err(err).Msg("Error reading JS file")
			stopApplication(cancel, &wg, client)
		} else {
			pStrData := string(data)
			jsCtx.PushLstring(pStrData, len(pStrData))
			if err := jsCtx.Peval(); err != nil {
				GLogger.Error().Err(err).Msg("Error parsing JS file")
				stopApplication(cancel, &wg, client)

			}
			jsCtx.Pop()
			//Test the method exists
			jsCtx.PushGlobalObject()
			jsCtx.GetPropString(-1, "panoptesProcess")
			jsCtx.PushString("{}")
			if jsCtx.Pcall(1) != 0 {
				str := jsCtx.SafeToString(-1)
				GLogger.Error().Str("error", str).Msg("Missing required function panoptesProcess(jsonData) {} ")
				stopApplication(cancel, &wg, client)
			}
			jsCtx.Pop3()
			defer jsCtx.DestroyHeap()
		}
		go func() {
			jsProcessor(jsChan, jsCtx, ctx)
			defer wg.Done()
		}()
		wg.Add(1)

	}

	go func() {
		consumer(eventChan, errorChan, ctx, jsChan)
		defer wg.Done()
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
