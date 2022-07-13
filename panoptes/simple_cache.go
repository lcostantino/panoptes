package panoptes

import "sync"

const MAX_ITEMS int = 2000

type CacheEvent struct {
	data  [][]byte
	mLock sync.Mutex
}

func (c *CacheEvent) AddEvent(jsonData []byte) {
	c.mLock.Lock()

	defer c.mLock.Unlock()
	if len(c.data) > MAX_ITEMS {
		return
	}
	c.data = append(c.data, jsonData)
}

func (c *CacheEvent) GetCopyAndClean() [][]byte {

	var dataCopy [][]byte
	c.mLock.Lock()
      if len(c.data) == 0 { 
          return [][]byte{}
      }
	dataCopy = append(dataCopy, c.data...)
	c.data = [][]byte{}
	c.mLock.Unlock()
	return dataCopy
}
