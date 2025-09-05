package main

import (
	"bufio"
	"errors"
	"os"
	"sync"
)

type Writer interface {
	Write(chan []rune) (error)
}

type FileWriter struct {
	filePath string
	bufPool *RunePool
	wg *sync.WaitGroup
}

func NewFileWriter(filePath string, bufPool *RunePool, wg *sync.WaitGroup) (*FileWriter) {
	return &FileWriter{
		filePath: filePath,
		bufPool: bufPool,
		wg: wg,
	}
}

func (fe *FileWriter) Write(in chan []rune) (error) {
	file, err := os.Create(fe.filePath);
	if err != nil {
		return errors.New("could not open output file")
	}

	go func() {
		defer file.Close()

		writer := bufio.NewWriter(file)
		defer writer.Flush()
		defer fe.wg.Done()

		for line := range in {
			for _ , r := range line {
				writer.WriteRune(r)
			}
	
			writer.WriteRune('\n')
	
			fe.bufPool.Put(line)
		}
	}()

	return nil
}