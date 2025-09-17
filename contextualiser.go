package main

import (
	"fmt"
	"sync"

	"github.com/braintrustdata/braintrust-go"
)

type samples []Sentence

type Contextualiser interface {
	contextualise(ContextCandidate) (Context, error)
	accumulate(Sentence, chan Sentence) error
	Ingest(chan Sentence) error
}

type SentenceContextualiser struct {
	sampleStore     *MemoryStore[samples]
	contextRegistry *MemoryStore[Context]
	maskRegistry    *MemoryStore[bool]
	wg              *sync.WaitGroup
	bClient         braintrust.Client
}

func NewSentenceContextualiser(contextRegistry *MemoryStore[Context], maskRegistry *MemoryStore[bool], wg *sync.WaitGroup) *SentenceContextualiser {
	client := braintrust.NewClient() // Defaults to os.LookUpEnv("BRAINTRUST_API_KEY")
	return &SentenceContextualiser{
		sampleStore: &MemoryStore[samples]{
			data: make(map[string]samples),
		},
		contextRegistry: contextRegistry,
		maskRegistry:    maskRegistry,
		wg:              wg,
		bClient:         client,
	}
}

func (sc *SentenceContextualiser) contextualise(input ContextCandidate) (Context, error) {
	// Responsible for preparing an api call to openai using the information in the input.
	sc.bClient.Functions.Invoke()
	return Context{}, nil
}

func (sc *SentenceContextualiser) accumulate(input Sentence, registeredChan chan Sentence) error {
	m := string(input.Mask)
	samples, err := sc.sampleStore.Get(m)
	if err != nil {
		value := []Sentence{input}
		sc.sampleStore.Put(m, value)
		return nil
	}

	// We want to keep accumulate all samples that have the same mask
	samples = append(samples, input)
	sc.sampleStore.Put(m, samples)

	if len(samples) == 3 {
		go func() {
			// Syncs with admin to close registered channel
			defer sc.wg.Done()

			var logLines []LogLine
			for _, s := range samples {
				logLines = append(logLines, s.Line)
			}

			candidate := ContextCandidate{
				Mask:    input.Mask,
				Samples: logLines,
			}

			context, err := sc.contextualise(candidate)
			if err != nil {
				fmt.Println("could not contextualise sentence")
				return
			}

			// Update context registry so that context can be fetched when labelling
			sc.contextRegistry.Put(m, context)

			// Update mask registry so that admin can direct all sentences with this mask signature to the correct context
			sc.maskRegistry.Put(m, true)

			// Release all samples into registered channel for processing
			// Refetch all samples as more could have been added
			samples, err := sc.sampleStore.Get(m)
			if err != nil {
				fmt.Println("could not contextualise sentence")
				return

			}

			for _, sample := range samples {
				registeredChan <- sample
			}

		}()
	}

	return nil
}

func (sc *SentenceContextualiser) Ingest(unRegistered chan Sentence, registered chan Sentence) error {
	for s := range unRegistered {
		sc.accumulate(s, registered)
	}

	return nil
}
