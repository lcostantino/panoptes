package panoptes

import (
	"errors"
	"strconv"
	"sync"

	//	"github.com/bi-zone/etw"
	"github.com/lcostantino/etw"
	"golang.org/x/sys/windows"
)

type Client struct {
	providers       map[string]Provider
	currentSessions []*etw.Session
	running         bool
	wg              sync.WaitGroup
}

func (c *Client) AddProvider(prov Provider) error {

	var err error
	if prov.winGuid, err = windows.GUIDFromString(prov.Guid); err != nil {
		return err
	}
	if prov.Name == "" {
		return errors.New("Empty provider name")
	}
	if prov.Report != Json && prov.Report != GoCallback {
		return errors.New("Invalid report mode")
	}

	if _, ok := c.providers[prov.Name]; ok {
		return errors.New("A provider with the same name is already registered")
	} else {
		c.providers[prov.Name] = prov

	}

	return nil
}

func getLevelFor(val int) etw.TraceLevel {
	switch val {
	case 1:
		return etw.TRACE_LEVEL_VERBOSE
	case 2:
		return etw.TRACE_LEVEL_ERROR
	case 3:
		return etw.TRACE_LEVEL_WARNING
	case 4:
		return etw.TRACE_LEVEL_INFORMATION
	case 5:
		return etw.TRACE_LEVEL_VERBOSE
	}

	return etw.TRACE_LEVEL_VERBOSE
}

// This will generate waitSync groups
func (c *Client) Start(eventChan chan Event, errorChan chan error) error {
	if c.providers == nil || len(c.providers) == 0 {
		return errors.New("Empty providers list")
	}

	for _, r := range c.providers {
		aLevel := getLevelFor(r.Options.Level)
		var anyKeyword, allKeyword uint64
		var kerr error
		if anyKeyword, kerr = strconv.ParseUint(r.Options.MatchAnyKeyword, 16, 64); kerr != nil {
			return errors.New("Failed to parse any keywords: " + kerr.Error())
		}
		if allKeyword, kerr = strconv.ParseUint(r.Options.MatchAllKeyword, 16, 64); kerr != nil {
			return errors.New("Failed to parse all keywords: " + kerr.Error())
		}

		if sess, err := etw.NewSession(r.winGuid, etw.WithLevel(aLevel), etw.WithMatchKeywords(anyKeyword, allKeyword)); err == nil {
			c.currentSessions = append(c.currentSessions, sess)

			go func(guid, name string, filterIds []uint16, includeRaw bool) {
				c.wg.Add(1)
				if err := sess.Process(func(e *etw.Event) {
					if filterIds != nil && len(filterIds) > 0 {
						matched := false
						//Will be replaced by Event Filter ID at ETW lvl (win10sdk)
						for _, fId := range filterIds {
							if fId == uint16(e.Header.ID) {
								matched = true
								break
							}

						}
						if matched == false {
							return
						}
					}
					event := make(map[string]interface{})
					event["Header"] = e.Header
					//Ideally, this will encoded base64 the raw data so it can be parsed with JS :)
					event["ExtendedData"] = e.ExtendedInfo()
					if data, err := e.EventProperties(includeRaw); err == nil {
						event["Props"] = data
					} else {
						errorChan <- err
					}
					if includeRaw {
						event["RawData"] = e.RawEncodedEvent
					}

					eventChan <- Event{EventData: event, Name: name, Guid: guid}
				}); err != nil {
					errorChan <- err
				}
				defer c.wg.Done()
			}(r.Guid, r.Name, r.Options.FilterEventIds, r.IncludeRawEvent)
		} else {
			return err
		}

	}
	return nil
}

func (c *Client) Stop() {
	for _, sess := range c.currentSessions {
		sess.Close()
	}
	c.currentSessions = make([]*etw.Session, 0)
}

func (c *Client) Pull() { c.wg.Wait() }

func (c *Client) IsRunning() bool { return c.running }

func (c *Client) GetProviders() map[string]Provider { return c.providers }

func NewClient() *Client {
	cp := Client{running: false, providers: map[string]Provider{}}
	return &cp
}
