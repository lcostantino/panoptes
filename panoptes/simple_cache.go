package panoptes

import "sync"

const MAX_ITEMS int = 2000

type CacheEvent struct {
	data  []string
	mLock sync.Mutex
}

func (c *CacheEvent) AddEvent(jsonData string) {
	c.mLock.Lock()

	defer c.mLock.Unlock()
	if len(c.data) > MAX_ITEMS {
		return
	}
	c.data = append(c.data, jsonData)
}

func (c *CacheEvent) GetCopyAndClean() []string {

	var dataCopy []string
	c.mLock.Lock()

	dataCopy = append(dataCopy, c.data...)
	c.data = []string{}
	c.mLock.Unlock()
	return dataCopy
}
