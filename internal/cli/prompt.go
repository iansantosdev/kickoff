package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"syscall"
)

// ErrPromptCanceled reports that the user intentionally aborted an interactive prompt.
var ErrPromptCanceled = errors.New("prompt canceled")

// ErrPromptClosed reports that the interactive input stream was closed.
var ErrPromptClosed = errors.New("prompt input closed")

func isPromptCancel(input string) bool {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "q", "quit", "exit", "sair":
		return true
	default:
		return false
	}
}

type promptReadResult struct {
	input string
	err   error
}

const (
	ansiCursorUpLine = "\033[1F"
	ansiClearLine    = "\033[2K"
)

func writeChoicePrompt(stdout io.Writer, prompt string, first bool) {
	if first {
		fmt.Fprintf(stdout, "\n%s: ", prompt)
		return
	}
	fmt.Fprintf(stdout, "%s: ", prompt)
}

func rewriteChoicePrompt(stdout io.Writer, prompt, invalid string, invalidShown bool) {
	clearPreviousChoiceLine(stdout)
	if !invalidShown {
		fmt.Fprintf(stdout, "%s\n", invalid)
	}
	fmt.Fprintf(stdout, "%s: ", prompt)
}

func clearPreviousChoiceLine(stdout io.Writer) {
	fmt.Fprint(stdout, ansiCursorUpLine, ansiClearLine)
}

func clearChoiceBlock(stdout io.Writer, optionCount int, invalidShown bool) {
	lines := optionCount + 3
	if invalidShown {
		lines++
	}
	for range lines {
		clearPreviousChoiceLine(stdout)
	}
}

func readPromptInput(ctx context.Context, reader *bufio.Reader) (string, error) {
	if ctx.Err() != nil {
		return "", ErrPromptCanceled
	}

	resultCh := make(chan promptReadResult, 1)
	go func() {
		input, err := reader.ReadString('\n')
		resultCh <- promptReadResult{input: input, err: err}
	}()

	select {
	case <-ctx.Done():
		return "", ErrPromptCanceled
	case result := <-resultCh:
		if result.err != nil {
			if ctx.Err() != nil || isPromptInterruptError(result.err) {
				return "", ErrPromptCanceled
			}
			if errors.Is(result.err, io.EOF) {
				return "", ErrPromptClosed
			}
			return "", fmt.Errorf("error reading input: %w", result.err)
		}
		return strings.TrimSpace(result.input), nil
	}
}

func isPromptInterruptError(err error) bool {
	return errors.Is(err, context.Canceled) ||
		errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, syscall.EINTR)
}
