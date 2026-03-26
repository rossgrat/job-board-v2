package model

import "github.com/google/uuid"

type RawJob struct {
	CompanyID   uuid.UUID
	SourceJobID string
	URL         string
	RawData     []byte
}
