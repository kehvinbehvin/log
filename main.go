// High-performance log masking and normalization tool.
// Processes large log files (500K+ lines) with minimal memory usage (~2MB) and sub-second performance.
// Uses a pipeline architecture: FileReader -> MaskConsumer -> FileWriter with buffer pooling for zero-allocation processing.
package main

import (
	//"flag"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	// "io"
	// "unicode/utf8"
)

// type LogLine []rune // 03-17 16:13:38.936  1702 14638 D PowerManagerService: release:lock=189667585, flg=0x0, tag="*launch*", name=android", ws=WorkSource{10113}, uid=1000, pid=1702

// type LogMask []rune // Y-Y Y:Y:Y.Y  Y Y Y Y: Y:Y=Y, Y=Y, Y="X", Y=Y", Y=Y{X}, Y=Y, Y=Y

// type Token []rune // Represents a single unit of value in a single line of log (A Slice of the original LogLine)

// type TokenLabel string

// type Context struct {
// 	SystemLabel string
// 	TokenLabels []TokenLabel
// }

// type ContextCandidate struct {
// 	Mask LogMask
// 	Samples []LogLine // 5-6 lines of logs with the same mask
// }

// type ContextManager interface {
// 	Evaluate(ContextCandidate) (Context, error)
// 	Verify(Context, Samples []LogLine) bool
// }

// type Sentence struct {
// 	Tokens []Token
// 	Mask LogMask
// }

// // Key Value Map of Labels to their underlying token
// type LabelledTokens struct {
// 	data map[TokenLabel]Token
// }

// type TokenLabeler interface {
// 	LabelTokens(Context, Sentence) LabelledTokens
// }

// type ContextMapper interface {
// 	ContextMap(Sentence) (Context, error)
// }

// type Tokenizer interface {
// 	Tokenize(LogLine) []Token
// }

// type Decomposer interface {
// 	Decompose(LogLine) Sentence
// }

// type LogNormaliser struct {
// 	runePool 	   *RunePool
// 	workers        int
// 	writers		   int
// 	memory         int
// 	inputFilePath  string
// 	outputFilePath string
// 	store 		   *MemoryStore[bool]
// }

// func NewLogNormaliser(workers int, writers int, memory int, input string, output string) *LogNormaliser {
// 	runePool := NewRunePool(10000, 24)
// 	store := NewMemoryStore()

// 	return &LogNormaliser{
// 		runePool: 		runePool,
// 		workers:        workers,
// 		writers: 		writers,
// 		memory:         memory,
// 		inputFilePath:  input,
// 		outputFilePath: output,
// 		store: 			store,
// 	}
// }

// func (ln *LogNormaliser) Start() error {
// 	fmt.Println("Start normalising")
// 	// Setup files
// 	file, err := os.Open(ln.inputFilePath)
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Println("Input file opened")

// 	defer file.Close()

// 	outputFile, err := os.Create(ln.outputFilePath)
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Println("Output file opened")

// 	defer outputFile.Close()

// 	// Setup channels
// 	lines := make(chan []rune, ln.memory)
// 	var results []chan []rune
// 	for i := 0; i < ln.writers; i++ {
// 		results = append(results, make(chan []rune, ln.memory))
// 	}

// 	fmt.Println("Channels setup")

// 	var ingestGrp sync.WaitGroup
// 	// Setup workers
// 	for i := 0; i < ln.workers; i++ {
// 		ingestGrp.Add(1)
// 		go ln.startWorker(i, lines, results[i % ln.writers], &ingestGrp)
// 	}

// 	fmt.Println("Workers started")

// 	var outputGrp sync.WaitGroup
// 	var tempFiles []*os.File

// 	for i := 0; i < ln.writers; i++ {
// 		tempFile, err := os.CreateTemp("", fmt.Sprintf("worker_%d_*.log", i))
// 		if err != nil {
// 			log.Fatal(err)
// 		}

// 		tempFiles = append(tempFiles, tempFile)

// 		outputGrp.Add(1)
// 		go ln.BatchOutput(results[i], &outputGrp, tempFile)
// 	}

// 	fmt.Println("Batched output started")

// 	// Ingest logs
// 	fmt.Println("Start processing")
// 	scanner := bufio.NewScanner(file)

// 	var scanBuf []byte
// 	for scanner.Scan() {
// 		// Blocks Scanning if channel is full
// 		// lines <- []rune(scanner.Text()) Process over 100K Lines faster, but program memory scales linearly with input size

// 		// We trade performance because of manual decoding for constant memory size.
// 		// Get a rune buffer from runePool
// 		// rune buffer is only released after its processed contents has been passed to the output file writer
// 		output := ln.runePool.Get()
// 		scanBuf = scanner.Bytes()
// 		for len(scanBuf) > 0 {
// 			r, size := utf8.DecodeRune(scanBuf)
// 			output = append(output, r)
// 			scanBuf = scanBuf[size:]
// 		}

// 		lines <- output
// 	}

// 	// All lines in file has been read in channel
// 	close(lines)
// 	ingestGrp.Wait()

// 	// We can close all the channels because all the workers have completed their tasks
// 	for _, result := range results {
// 		close(result)
// 	}

// 	// Now we need to wait for all the writers to complete writing to their files
// 	outputGrp.Wait()

// 	// Combine the temp file into a single file
// 	// Close and Remove the temp files
// 	for _, tempFile := range tempFiles {
// 		tempFile.Seek(0, 0)

// 		io.Copy(outputFile, tempFile)
// 		tempFile.Close()
// 		os.Remove(tempFile.Name())
// 	}

// 	return nil
// }

// func (ln *LogNormaliser) startWorker(id int, lines <-chan []rune, result chan<- []rune, wg *sync.WaitGroup) error {
// 	defer wg.Done()
// 	// fmt.Println("Worker started: " + strconv.Itoa(id))

// 	for line := range lines {
// 		logLine, err := Mask(line)
// 		if err != nil {
// 			return err
// 		}

// 		// Blocks if channel is full
// 		// Allocates a new string here.
// 		result <- logLine
// 	}

// 	return nil
// }

// func (ln *LogNormaliser) BatchOutput(results <-chan []rune, wg *sync.WaitGroup, outputFile *os.File) {
// 	defer wg.Done()
// 	// var sb strings.Builder

// 	// Defaults 4KB per batch
// 	writer := bufio.NewWriter(outputFile)
// 	defer writer.Flush()

// 	// Allocating new string here
// 	for result := range results {
// 		value := string(result)
// 		if exists, _ := ln.store.Get(value); !exists {
// 			ln.store.Put(value, true)
// 		}

// 		for _ , r := range result {
// 			writer.WriteRune(r)
// 		}

// 		writer.WriteRune('\n')

// 		// Return rune buffer to pool
// 		ln.runePool.Put(result)
// 	}
// }

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
	fmt.Println("Start")

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Add(1)

	fileReader := NewFileReader("./data/raw/mini.log")
	maskConsumer := NewMaskConsumer()
	fileWriter := NewFileBufferWriter("./data/results/data.log", &wg)
	fileIntWriter := NewFileIntWriter("./data/results/data_int.log", &wg)

	readOut, err := fileReader.Read()
	if err != nil {
		fmt.Println("error when reading from file")
		return
	}

	maskOut, tokenOut, err := maskConsumer.Consume(readOut)
	if err != nil {
		fmt.Println("error when masking")
		return
	}

	err = fileWriter.Write(maskOut)
	if err != nil {
		fmt.Println("error when writing")
		return
	}

	err = fileIntWriter.Write(tokenOut)
	if err != nil {
		fmt.Println("error when writing tokens")
		return
	}

	wg.Wait()

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
}
