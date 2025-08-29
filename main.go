package main

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"flag"
	"log"
	"runtime/pprof"
	"runtime"
	"io"
	"unicode/utf8"
)

const (
	nestedContent               = 'X'
	topLevelAlphaNumericContent = 'Y'
)

var enclosingSymbols = map[rune]rune{
	'[':  ']',
	'{':  '}',
	'<':  '>',
	'(':  ')',
	'"':  '"',
	'\'': '\'',
}

func RemoveAlphanumeric(input []rune) []rune {
	for i, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			input[i] = topLevelAlphaNumericContent
		}
	}

	return input
}

func Compress(input []rune) ([]rune, error) {
	var counter int

	for i, current := range input {
		// Append all symbols
		if current != topLevelAlphaNumericContent {
			input[counter] = current
			counter++
			continue
		}

		// Only append the last Y of connected Ys
		// Only append the last Y of the input
		if (i+1) == len(input) || input[i+1] != topLevelAlphaNumericContent {
			input[counter] = current
			counter++
		}
	}

	return input[:counter], nil
}

func Mask(input []rune, closingSym rune) ([]rune, int, error) {
	var content []rune

	for i := 0; i < len(input); i++ {
		content = append(content, input[i])
		close, open := enclosingSymbols[input[i]]
		
		// Check for closing syms first because some closing symbols can be the same as their opening
		if input[i] == closingSym {
			// Closing Sym found, all nested content in this stack should be masked
			return []rune{'X', closingSym}, i, nil 
		}
		
		if open {
			// State A: Closing sym found -> Mask returned
			// State B: Closing sym found but no content wanted -> empty rune slice returned
			innerContent, depth, err := Mask(input[i + 1:], close)
			if err != nil {
				return []rune{}, 0, err
			}

			// Fast forward + offset
			i = i + depth + 1
			// Append whatever Mask returns
			content = append(content, innerContent...)

			if i >= len(input) {
				break
			}

			continue
		}
	}

	// No closing symbols found, return whatever we have processed.
	// Amount processed is not len(content)
	return content, len(input), nil
}

func ProcessLogLine(input []rune) ([]rune, error) {
	symbolsOnly := RemoveAlphanumeric(input)	
	maskedSymbols, _, err := Mask(symbolsOnly, 0) // Returns new slice headers?
	if err != nil {
		return []rune{}, err
	}

	compressed, err := Compress(maskedSymbols)
	if err != nil {
		return []rune{}, err
	}

	return compressed, nil
}

func (ln *LogNormaliser) BatchOutput(results <-chan []rune, wg *sync.WaitGroup, outputFile *os.File) {
	defer wg.Done()
	// var sb strings.Builder

	// Defaults 4KB per batch
	writer := bufio.NewWriter(outputFile)
	defer writer.Flush()

	// Allocating new string here
	for result := range results {
		value := string(result)
		if exists, _ := ln.store.Get(value); !exists {
			ln.store.Put(value, true)
		}

		for _ , r := range result {
			writer.WriteRune(r)
		}

		writer.WriteRune('\n')
		
		// Return rune buffer to pool
		ln.runePool.Put(result)
	}
}

type LogNormaliser struct {
	runePool 	   *RunePool
	workers        int
	writers		   int
	memory         int
	inputFilePath  string
	outputFilePath string
	store 		   Store[bool]
}

func NewLogNormaliser(workers int, writers int, memory int, input string, output string) *LogNormaliser {
	runePool := NewRunePool(10000, 24)
	store := NewMemoryStore()

	return &LogNormaliser{
		runePool: 		runePool,
		workers:        workers,
		writers: 		writers,
		memory:         memory,
		inputFilePath:  input,
		outputFilePath: output,
		store: 			store,
	}
}

func (ln *LogNormaliser) Start() error {
	fmt.Println("Start normalising")
	// Setup files
	file, err := os.Open(ln.inputFilePath)
	if err != nil {
		return err
	}
	fmt.Println("Input file opened")

	defer file.Close()

	outputFile, err := os.Create(ln.outputFilePath)
	if err != nil {
		return err
	}
	fmt.Println("Output file opened")

	defer outputFile.Close()

	// Setup channels
	lines := make(chan []rune, ln.memory)
	var results []chan []rune
	for i := 0; i < ln.writers; i++ {
		results = append(results, make(chan []rune, ln.memory))
	}

	fmt.Println("Channels setup")

	var ingestGrp sync.WaitGroup
	// Setup workers
	for i := 0; i < ln.workers; i++ {
		ingestGrp.Add(1)
		go ln.startWorker(i, lines, results[i % ln.writers], &ingestGrp)
	}

	fmt.Println("Workers started")

	var outputGrp sync.WaitGroup
	var tempFiles []*os.File

	for i := 0; i < ln.writers; i++ {
		tempFile, err := os.CreateTemp("", fmt.Sprintf("worker_%d_*.log", i))
		if err != nil {
			log.Fatal(err)
		}

		tempFiles = append(tempFiles, tempFile)

		outputGrp.Add(1)
		go ln.BatchOutput(results[i], &outputGrp, tempFile)
	}

	fmt.Println("Batched output started")


	// Ingest logs
	fmt.Println("Start processing")
	scanner := bufio.NewScanner(file)

	var scanBuf []byte
	for scanner.Scan() {
		// Blocks Scanning if channel is full
		// lines <- []rune(scanner.Text()) Process over 100K Lines faster, but program memory scales linearly with input size

		// We trade performance because of manual decoding for constant memory size.
		// Get a rune buffer from runePool
		// rune buffer is only released after its processed contents has been passed to the output file writer
		output := ln.runePool.Get()
		scanBuf = scanner.Bytes()
		for len(scanBuf) > 0 {
			r, size := utf8.DecodeRune(scanBuf)
			output = append(output, r)
			scanBuf = scanBuf[size:]
		}

		lines <- output
	}

	// All lines in file has been read in channel
	close(lines)
	ingestGrp.Wait()

	// We can close all the channels because all the workers have completed their tasks
	for _, result := range results {
		close(result)
	}

	// Now we need to wait for all the writers to complete writing to their files
	outputGrp.Wait()

	// Combine the temp file into a single file
	// Close and Remove the temp files
	for _, tempFile := range tempFiles {
		tempFile.Seek(0, 0)

		io.Copy(outputFile, tempFile)
		tempFile.Close()
		os.Remove(tempFile.Name())
	}

	return nil
}

func (ln *LogNormaliser) startWorker(id int, lines <-chan []rune, result chan<- []rune, wg *sync.WaitGroup) error {
	defer wg.Done()
	// fmt.Println("Worker started: " + strconv.Itoa(id))

	for line := range lines {
		logLine, err := ProcessLogLine(line)
		if err != nil {
			return err
		}

		// Blocks if channel is full
		// Allocates a new string here.
		result <- logLine 
	}

	return nil
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("Could not create cpu profile")
		}

		defer f.Close()

		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("Could not start cpu profile")
		}

		defer pprof.StopCPUProfile()
	}

	normaliser := NewLogNormaliser(1, 1, 100, "./data/raw/mixed.log", "./data/results/data.log")
	normaliser.Start()

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("Could not create memory profile")
		}

		defer f.Close()
		runtime.GC()

		if err := pprof.Lookup("allocs").WriteTo(f, 0); err != nil {
            log.Fatal("could not write memory profile: ", err)
        }
	}

	normaliser.store.Report("./data/results/store.log")
}
