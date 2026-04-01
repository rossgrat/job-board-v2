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
	// json.Marshal escapes <, >, & to unicode sequences; undo that so the
	// HTML tag regex can match them.
	s := strings.ReplaceAll(string(rawData), `\u003c`, "<")
	s = strings.ReplaceAll(s, `\u003e`, ">")
	s = strings.ReplaceAll(s, `\u0026`, "&")
	plain := htmlTagRegex.ReplaceAllString(s, " ")
	plain = htmlEntityRegex.ReplaceAllString(plain, " ")
	plain = whitespaceRegex.ReplaceAllString(plain, " ")
	plain = strings.TrimSpace(plain)
	return plain
}
