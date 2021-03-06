package storage

import (
	"errors"
	"github.com/KyberNetwork/reserve-data/common"
	"sync"
)

type RamRateStorage struct {
	mu      sync.RWMutex
	version int64
	data    map[int64]common.AllRateEntry
}

func NewRamRateStorage() *RamRateStorage {
	return &RamRateStorage{
		mu:      sync.RWMutex{},
		version: 0,
		data:    map[int64]common.AllRateEntry{},
	}
}

func (self *RamRateStorage) CurrentVersion(timepoint uint64) (int64, error) {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.version, nil
}

func (self *RamRateStorage) GetRates(version int64) (common.AllRateEntry, error) {
	self.mu.RLock()
	defer self.mu.RUnlock()
	all, exist := self.data[version]
	if !exist {
		return common.AllRateEntry{}, errors.New("Version doesn't exist")
	} else {
		return all, nil
	}
}

func (self *RamRateStorage) StoreNewData(data common.AllRateEntry, timepoint uint64) error {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.version = self.version + 1
	self.data[self.version] = data
	delete(self.data, self.version-1)
	return nil
}
