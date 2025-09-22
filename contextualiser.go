package main

import (
	"context"
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

type ContextualiseResponse struct {
	Labels []string `json:"labels"`
}

func (sc *SentenceContextualiser) contextualise(input ContextCandidate) (Context, error) {
	// Responsible for preparing an api call to openai using the information in the input.
	response, err := sc.bClient.Functions.Invoke(context.TODO(), "a26dfd04-0fd7-4a77-aa45-826560d785ab", braintrust.FunctionInvokeParams{
		Input: map[string]interface{}{
			"examples": "<examples><i>03-17 16:13:45.382  1702  3697 D PowerManagerService: acquire lock=189667585, flags=0x1, tag=\"*launch*\", name=android, ws=WorkSource{10113}, uid=1000, pid=1702</i></examples>",
			"template": "<template>Y-Y Y:Y:Y.Y  Y  Y Y Y: Y Y=Y, Y=Y, Y=\"X\", Y=Y, Y=Y{X}, Y=Y, Y=Y</template>",
		},
	})

	if err != nil {
		return Context{}, err
	}

	// The response is a pointer to any, so we need to dereference it first
	responseMap, ok := (*response).(map[string]interface{})
	if !ok {
		return Context{}, fmt.Errorf("failed to assert response as map[string]interface{}")
	}

	// Extract the labels field from the response
	labelsInterface, exists := responseMap["labels"]
	if !exists {
		return Context{}, fmt.Errorf("labels field not found in response")
	}

	// Type assert the labels to []interface{} and convert to []string
	labelsSlice, ok := labelsInterface.([]interface{})
	if !ok {
		return Context{}, fmt.Errorf("labels field is not an array")
	}

	var labels []string
	for _, label := range labelsSlice {
		labelStr, ok := label.(string)
		if !ok {
			return Context{}, fmt.Errorf("label is not a string: %v", label)
		}
		labels = append(labels, labelStr)
	}

	return Context{labels: labels}, nil
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
