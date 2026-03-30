package model

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	htmlTagRegex    = regexp.MustCompile(`<[^>]*>`)
	htmlEntityRegex = regexp.MustCompile(`&[a-zA-Z0-9#]+;`)
	whitespaceRegex = regexp.MustCompile(`\s{2,}`)
)

type RawJob struct {
	CompanyID   uuid.UUID
	SourceJobID string
	URL         string
	RawData     []byte
	CleanData   string
}

func CleanContent(rawData []byte) string {
	plain := htmlTagRegex.ReplaceAllString(string(rawData), " ")
	plain = htmlEntityRegex.ReplaceAllString(plain, " ")
	plain = whitespaceRegex.ReplaceAllString(plain, " ")
	plain = strings.TrimSpace(plain)
	return plain
}
