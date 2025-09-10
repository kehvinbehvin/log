package main

// TODO: Refactor with better abstraction and rename the structs!!
import (
	"bufio"
	"errors"
	"os"
	"strings"
	"sync"
)

type Writer[T any] interface {
	Write(chan T) error
}

type FileBufferWriter struct {
	filePath string
	wg       *sync.WaitGroup
}

func NewFileBufferWriter(filePath string, wg *sync.WaitGroup) *FileBufferWriter {
	return &FileBufferWriter{
		filePath: filePath,
		wg:       wg,
	}
}

func (fe *FileBufferWriter) Write(in chan []rune) error {
	file, err := os.Create(fe.filePath)
	if err != nil {
		return errors.New("could not open output file")
	}

	go func() {
		defer file.Close()

		writer := bufio.NewWriter(file)
		defer writer.Flush()
		defer fe.wg.Done()

		for line := range in {
			for _, r := range line {
				writer.WriteRune(r)
			}

			writer.WriteRune('\n')
		}
	}()

	return nil
}

type FileIntWriter struct {
	filePath string
	wg       *sync.WaitGroup
}

func NewFileIntWriter(filePath string, wg *sync.WaitGroup) *FileIntWriter {
	return &FileIntWriter{
		filePath: filePath,
		wg:       wg,
	}
}

func (fw *FileIntWriter) Write(in chan [][]rune) error {
	file, err := os.Create(fw.filePath)
	if err != nil {
		return errors.New("could not open output file")
	}

	go func() {
		defer file.Close()

		writer := bufio.NewWriter(file)
		defer writer.Flush()
		defer fw.wg.Done()

		for line := range in {
			var sb strings.Builder
			for _, runeSlice := range line {
				sb.WriteString(string(runeSlice))
				sb.WriteString(",")
			}

			s := sb.String()
			writer.WriteString(s)
			writer.WriteRune('\n')
		}
	}()

	return nil
}
