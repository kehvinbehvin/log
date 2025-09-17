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
)

type LogLine []rune // 03-17 16:13:38.936  1702 14638 D PowerManagerService: release:lock=189667585, flg=0x0, tag="*launch*", name=android", ws=WorkSource{10113}, uid=1000, pid=1702

type LogMask []rune // Y-Y Y:Y:Y.Y  Y Y Y Y: Y:Y=Y, Y=Y, Y="X", Y=Y", Y=Y{X}, Y=Y, Y=Y

type Token []rune // Represents a single unit of value in a single line of log (A Slice of the original LogLine)

type TokenLabel string

type ContextCandidate struct {
	Mask    LogMask
	Samples []LogLine // 5-6 lines of logs with the same mask
}

type Sentence struct {
	Tokens []Token
	Mask   LogMask
	Line   LogLine
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
	fmt.Println("Start")

	var wg sync.WaitGroup
	wg.Add(2)

	fileReader := NewFileReader("./data/raw/mini.log")
	maskConsumer := NewMaskConsumer()
	fileWriter := NewFileBufferWriter("./data/results/data.log", &wg)
	fileIntWriter := NewFileIntWriter("./data/results/data_int.log", &wg)
	maskRegistry := NewMemoryStore()
	contextRegistry := NewContextStore()
	admin := NewAdmin(maskRegistry, contextRegistry, &wg)
	contextualiser := NewSentenceContextualiser(contextRegistry, maskRegistry, &wg)
	labeller := NewTokenLabeller(contextRegistry)

	readOut, err := fileReader.Read()
	if err != nil {
		fmt.Println("error when reading from file")
		return
	}

	sentenceOut, err := maskConsumer.Consume(readOut)
	if err != nil {
		fmt.Println("error when masking")
		return
	}

	unRegistered, registered, err := admin.Administrate(sentenceOut)
	if err != nil {
		fmt.Println("error when administrating")
		return
	}

	err := contextualiser.Ingest(unRegistered, registered)
	if err != nil {
		fmt.Println("error when contextualising")
		return
	}

	labelledTokensChan, err := labeller.Ingest(registered)
	if err != nil {
		fmt.Println("error when labelling")
		return
	}

	go func() {
		// Synced between admin and contextualiser
		// as both are channel writers to registered chan
		wg.Wait()
		close(registered)
	}()

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
