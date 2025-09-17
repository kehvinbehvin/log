package main

import (
	"sync"
)

type UnRegisteredChan chan Sentence
type RegisteredChan chan Sentence

type Administrator interface {
	Administrate(chan Sentence) (UnRegisteredChan, RegisteredChan, error)
}

type Admin struct {
	maskStore    *MemoryStore[bool]
	contextStore *MemoryStore[Context]
	wg           *sync.WaitGroup
}

func NewAdmin(maskStore *MemoryStore[bool], contextStore *MemoryStore[Context], wg *sync.WaitGroup) *Admin {
	return &Admin{
		maskStore:    maskStore,
		contextStore: contextStore,
		wg:           wg,
	}
}

func (a *Admin) Administrate(input chan Sentence) (UnRegisteredChan, RegisteredChan, error) {
	unRegisteredChan := make(UnRegisteredChan, 100)
	registeredChan := make(RegisteredChan, 100)

	go func() {
		// Syncs with 2nd writer of registered chan in contextualiser
		defer a.wg.Done()

		// Safe to close unregistredChan since this is the only writer
		defer close(unRegisteredChan)

		// Decided not to close all channels here as we want the caller to handle the closing of the channels
		for s := range input {
			key := string(s.Mask)
			status, _ := a.maskStore.Get(key)
			if status {
				registeredChan <- s
			} else {
				a.maskStore.Put(key, false)
				unRegisteredChan <- s
			}

		}
	}()
	return unRegisteredChan, registeredChan, nil
}
