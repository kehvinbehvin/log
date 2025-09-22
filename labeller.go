package main

import "fmt"

type Context struct {
	labels []string
}

// Key Value Map of Labels to their underlying token
type LabelledTokens struct {
	data map[TokenLabel][]Token
}

type Labeler interface {
	LabelTokens(Context, Sentence) (LabelledTokens, error)
}

type TokenLabeller struct {
	contextRegistry *MemoryStore[Context]
}

func NewTokenLabeller(c *MemoryStore[Context]) *TokenLabeller {
	return &TokenLabeller{
		contextRegistry: c,
	}
}

func (te *TokenLabeller) LabelTokens(context Context, sentence Sentence) (LabelledTokens, error) {
	results := LabelledTokens{
		data: make(map[TokenLabel][]Token),
	}

	// Reject any context that do not match up 100% with tokens
	// Need to add retry handling here (braintrust client in contextualiser should handle the retry for us)
	if len(context.labels) != len(sentence.Tokens) {
		return results, fmt.Errorf(
			"token/label count mismatch: %d tokens, %d labels",
			len(sentence.Tokens),
			len(context.labels),
		)
	}

	// The order of labels and tokens should be the same.
	for i, label := range context.labels {
		tokenLabel := TokenLabel(label)
		results.data[tokenLabel] = append(results.data[tokenLabel], sentence.Tokens[i])
	}

	return results, nil
}

func (te *TokenLabeller) Ingest(input chan Sentence) (chan LabelledTokens, error) {
	output := make(chan LabelledTokens, 100)

	go func() {
		// We want to close the output channel only when the input channel is closed and empty
		defer close(output)

		for sentence := range input {
			m := string(sentence.Mask)
			c, err := te.contextRegistry.Get(m)
			// Ephemeral error should not stop processing other logs
			if err != nil {
				fmt.Println("error fetching context for mask") // Format mask variable into log
				continue
			}

			data, err := te.LabelTokens(c, sentence)
			// Ephemeral error should not stop processing other logs
			if err != nil {
				fmt.Println("error labelling tokens using context") // Format mask variable into log
				continue
			}

			output <- data
		}
	}()

	return output, nil
}
