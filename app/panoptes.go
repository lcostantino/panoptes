package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/lcostantino/Panoptes/panoptes"
	"github.com/lcostantino/go-duktape"
	"github.com/logrusorgru/aurora"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
)

var version = "replace"

type PanoptesArgs struct {
	stdout         bool
	configFile     string
	logFile        string
	javascriptFile string
	consumers      int
	stopFile       string
	httpEndpoint   string
	disableColors  bool
}

var SessionId uuid.UUID

func init() {
	SessionId = uuid.New()
}

var au aurora.Aurora

//Very basic flag logic, use cobra, clapper , pflags , etc if really needed in the future
func parseCommandLineAndValidate() PanoptesArgs {

	args := PanoptesArgs{}

	flag.StringVar(&args.configFile, "config-file", "", "Config file for sensors")
	flag.BoolVar(&args.stdout, "stdout", true, "Print to stdout")
	flag.StringVar(&args.httpEndpoint, "http-endpoint", "", "If not empty will host an HTTP server to retrieve data. Ex: localhost:3999")
	flag.StringVar(&args.logFile, "log-file", "", "Log file")
	flag.StringVar(&args.stopFile, "stop-file", "", "If the file is NOT present, the application will stop")
	flag.StringVar(&args.javascriptFile, "js-file", "", "JS processor file")
	flag.BoolVar(&args.disableColors, "no-colors", false, "Disable color output")
	flag.IntVar(&args.consumers, "consumers", 1, "number of consumer routines")
	flag.Parse()
	if args.configFile == "" {
		fmt.Println(au.Red("Error: You need to provide a valid config file\n"))
		os.Exit(1)
	}
	if args.disableColors == true {
		au = aurora.NewAurora(false)
	}
	return args
}

func parseConfigFile(fName string) []panoptes.Provider {
	if data, err := os.ReadFile(fName); err != nil {
		GLogger.Error().Err(err).Msg("Failed to open config file")
		os.Exit(1)
	} else {
		var mProviders []panoptes.Provider
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
func consumer(eventChan chan panoptes.Event, errorChan chan error, ctx context.Context, jsChan chan panoptes.Event, tempCache *panoptes.CacheEvent) {

	for {
		select {
		case <-ctx.Done():
			GLogger.Info().Msg("Context done")
			return
		case e, ok := <-eventChan:
			if !ok {
				return
			}
			e.SessionId = SessionId
			e.EventData["Panoptes"] = map[string]interface{}{"SessionId": SessionId}
			e.Marshalled, _ = json.Marshal(e.EventData)
			//If JS enabled let it decide wether to output or not the data
			if jsChan != nil {
				jsChan <- e
			} else {
				if tempCache != nil {
					tempCache.AddEvent(e.Marshalled)
				}
				GLogger.Info().RawJSON("etwEvent", []byte(e.Marshalled)).Str("name", e.Name).Str("guid", e.Guid).Msg("Data")
			}
		case err := <-errorChan:
			GLogger.Error().Err(err).Msg("Error consuming event")

		}
	}

}

//We can create multiples runtimes or in this case, just ONE to avoid locks and shared state for the moment.
func jsProcessor(eventChan chan panoptes.Event, jsCtx *duktape.Context, ctx context.Context, tempCache *panoptes.CacheEvent) {

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
					if tempCache != nil {
						tempCache.AddEvent(e.Marshalled)
					}
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
	fmt.Printf("---- [ Panoptes Ver: %s ] ----\n", version)
	args := parseCommandLineAndValidate()
	NewLogger(args.logFile, args.stdout, args.disableColors)

	client := panoptes.NewClient()

	providers := parseConfigFile(args.configFile)

	for _, r := range providers {
		if r.Disabled == true {
			continue
		}
		if err := client.AddProvider(r); err != nil {
			GLogger.Error().Err(err).Str("guid", r.Guid).Str("name", r.Name).Msg("Failed to add provider")
		} else {
			GLogger.Info().Str("guid", r.Guid).Str("name", r.Name).Msg("Registered")
		}
	}

	var tempCache *panoptes.CacheEvent
	eventChan := make(chan panoptes.Event, 10)
	errorChan := make(chan error)
	var jsChan chan panoptes.Event

	//just in case
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer close(eventChan)
	defer close(errorChan)
	// Start HTTP Functionality
	if args.httpEndpoint != "" {

		handleRequest := func(w http.ResponseWriter, r *http.Request) {
			nData := tempCache.GetCopyAndClean()
			w.Write([]byte("["))
			lenData := len(nData)
			for x, d := range nData {
				w.Write(d)
				if x < lenData-1 {
					w.Write([]byte(","))
				}
			}
			w.Write([]byte("]"))
		}
		handleGetLogFile := func(w http.ResponseWriter, r *http.Request) {
			if args.logFile == "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if data, err := os.ReadFile(args.logFile); err == nil {
				if _, err = w.Write(data); err != nil {
					GLogger.Error().Err(err).Msg("Error sending logfile data")
				}

			}

		}
		http.HandleFunc("/getEvents", handleRequest)
		http.HandleFunc("/getLogFile", handleGetLogFile)
		tempCache = &panoptes.CacheEvent{}
		go func() {
			if err := http.ListenAndServe(args.httpEndpoint, nil); err != nil {
				GLogger.Error().Err(err).Msg("Cannot start HTTP server")
				stopApplication(cancel, &wg, client)
			} else {
				//note: we are not waiting for the server to start so logs may be printed before this one..
				GLogger.Info().Msg("HTTP Server Listening (/getEvents,/getLogFile)")
			}

		}()
		// End HTTP Functionality
	}
	// Start JS Functionality
	if args.javascriptFile != "" {
		jsChan = make(chan panoptes.Event, args.consumers)
		jsCtx := duktape.New()

		jsCtx.PushGlobalGoFunction("convertStringToUtf8", func(c *duktape.Context) int {
			if c.GetTop() == 4 && c.IsBuffer(-4) && c.IsString(-1) && c.IsNumber(-2) && c.IsNumber(-3) {
				var decoder *encoding.Decoder
				switch fromEncoding := c.SafeToString(-1); fromEncoding {
				case "utf-16le":
					decoder = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
				case "utf-16":
					decoder = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
				default:
					GLogger.Error().Msg(fmt.Sprintf("Invalid encoding %s valid are (utf-16,utf-16le)", fromEncoding))
					return -1
				}

				lastIndex := c.GetInt(-2)
				fromIndex := c.GetInt(-3)
				myBuffer, len := c.GetBuffer(-4)
				if myBuffer != nil {

					if outData, err := decoder.Bytes(unsafe.Slice((*uint8)(unsafe.Pointer(myBuffer)), len)[fromIndex:lastIndex]); err == nil {
						c.PushString(string(outData))
						return 1
					} else {
						GLogger.Error().Err(err).Msg("Invalid convertString decoder.Bytes")
						return -1
					}
				}
			}
			GLogger.Error().Err(errors.New("Invalid argument count")).Msg("Invalid convertString call")
			return -1
		})
		jsCtx.PushGlobalGoFunction("pLog", func(c *duktape.Context) int {
			fName := c.SafeToString(-2)
			if fName != "" {
				if ds, err := os.OpenFile(fName, os.O_APPEND|os.O_CREATE, 0777); err == nil {
					ds.WriteString(c.SafeToString(-1))
					ds.Close()
					return 0
				} else {
					GLogger.Error().Err(err).Msg("Error writing to log from JS script")

				}
			}
			return -1

		})
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
			jsProcessor(jsChan, jsCtx, ctx, tempCache)
			defer wg.Done()
		}()
		wg.Add(1)
		// End JS Functionality
	}

	go func() {
		consumer(eventChan, errorChan, ctx, jsChan, tempCache)
		defer wg.Done()
	}()
	wg.Add(1)

	if err := client.Start(eventChan, errorChan); err != nil {
		GLogger.Error().Err(err).Msg("Failed to start")
		stopApplication(cancel, &wg, client)

	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	//Add Watchdog - usefully when running as PPL
	if args.stopFile != "" {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		go func() {
			for {
				select {
				case _ = <-ticker.C:
					if _, err := os.Stat(args.stopFile); err != nil {
						sigCh <- os.Interrupt
					}
				}
			}
		}()
	}
	for range sigCh {
		GLogger.Info().Msg("Shutting the session down")
		stopApplication(cancel, &wg, client)

	}
	client.Pull()

}
