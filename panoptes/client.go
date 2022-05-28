package panoptes

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

type Client struct {
	providers map[string]Provider
	running   bool
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

func (c *Client) Start() error {
	return nil
}

func (c *Client) Stop() error {
	return nil
}

func (c *Client) Pull() {}

func (c *Client) IsRunning() bool { return c.running }

func (c *Client) GetProviders() map[string]Provider { return c.providers }

func NewClient() *Client {
	cp := Client{running: false, providers: map[string]Provider{}}
	return &cp
}
