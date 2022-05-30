package panoptes

import (
	"errors"
	"fmt"
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

	fmt.Println(c.providers)
	fmt.Println(prov.Name)
	if _, ok := c.providers[prov.Name]; ok {
		return errors.New("A provider with the same name is already registered")
	} else {
		c.providers[prov.Name] = prov

	}

	return nil
}

// This will generate waitSync groups
func (c *Client) Start(cbk func(e *Event)) error {

	for _, r := range c.providers {
		if sess, err := etw.NewSession(r.winGuid); err != nil {
			c.currentSessions = append(c.currentSessions, sess)
			go func() {
				fmt.Printf("Adding session")
				c.wg.Add(1)
				if err := sess.Process(func(e *etw.Event) {
					cbk(&Event{EtwEvent: e})
				}); err != nil {
					fmt.Printf("Error? %s\n", err)
				}
				defer c.wg.Done()
			}()
		} else {
			return err
		}

	}
	return nil
}

func (c *Client) Stop() {
	for _, sess := range c.currentSessions {
		fmt.Println("Going to close session")
		sess.Close()
		fmt.Printf("Closing session")
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
