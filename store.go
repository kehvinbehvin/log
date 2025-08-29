package main;

import (
	"errors"
	"fmt"
	"os"
	"bufio"
)

type Store[T any] interface {
	Get(key string) (T, error)
	Put(key string, value T) (error)
	Report(fileName string) (error)
}

type MemoryStore[T any] struct {
	data map[string]T
}

func NewMemoryStore() (*MemoryStore[bool]) {
	return &MemoryStore[bool]{
		data: make(map[string]bool),
	}
}

func (m *MemoryStore[T]) Get(key string) (T, error) {
	value, exists := m.data[key]
	if !exists {
		var zero T
		return zero, errors.New("key not found")
	}

	return value, nil
}

func (m *MemoryStore[T]) Put(key string, value T) (error) {
	m.data[key] = value // Hardcoded for dev, remove in prod
	return nil
}

func (m *MemoryStore[T]) Report(filename string) (error) {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()
	
	for k := range m.data {
		_, err := fmt.Fprint(writer, k + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}