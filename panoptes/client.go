package panoptes

import (
	"errors"
	"sync"

	"github.com/bi-zone/etw"
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
		if sess, err := etw.NewSession(r.winGuid, etw.WithLevel(aLevel)); err == nil {
			c.currentSessions = append(c.currentSessions, sess)
			go func(guid, name string) {
				c.wg.Add(1)
				if err := sess.Process(func(e *etw.Event) {
					event := make(map[string]interface{})

					event["Header"] = e.Header
					if data, err := e.EventProperties(); err == nil {
						event["Props"] = data

					} else {
						errorChan <- err
					}

					eventChan <- Event{EventData: event, Name: name, Guid: guid}
				}); err != nil {
					errorChan <- err
				}
				defer c.wg.Done()
			}(r.Guid, r.Name)
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
