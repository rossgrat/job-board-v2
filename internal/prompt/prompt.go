package prompt

import (
	"errors"
	"fmt"
	"os"
)

var (
	ErrReadPromptFile  = errors.New("failed to read prompt file")
	ErrEmptyPromptFile = errors.New("prompt file is empty")
)

func Load(path string) (string, error) {
	const fn = "prompt::Load"

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s:%w:%w", fn, ErrReadPromptFile, err)
	}

	content := string(data)
	if content == "" {
		return "", fmt.Errorf("%s:%w", fn, ErrEmptyPromptFile)
	}

	return content, nil
}
