package main

import (
	"bufio"
	"errors"

	// "fmt"
	"os"
	"unicode/utf8"
)

// The responsibility of a ingester is to ingest logs from a source
type Reader interface {
	Read() (chan []rune, error)
}

type FileReader struct {
	filePath string
}

func NewFileReader(filePath string) *FileReader {
	return &FileReader{
		filePath: filePath,
	}
}

func (f *FileReader) Read() (chan []rune, error) {
	file, err := os.Open(f.filePath)
	if err != nil {
		return nil, errors.New("could not open file")
	}

	out := make(chan []rune, 100)

	go func() {
		scanner := bufio.NewScanner(file)
		var scanBuf []byte

		defer file.Close()
		defer close(out)

		for scanner.Scan() {

			output := make([]rune, 0)
			scanBuf = scanner.Bytes()

			for len(scanBuf) > 0 {
				r, size := utf8.DecodeRune(scanBuf)
				output = append(output, r)
				scanBuf = scanBuf[size:]
			}

			out <- output
		}
	}()

	return out, nil
}
