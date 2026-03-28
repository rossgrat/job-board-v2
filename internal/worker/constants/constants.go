package constants

// Pipeline task names
const (
	PipelineTriage        = "triage"
	PipelineNormalize     = "normalize"
	PipelineHardFilter    = "hard_filter"
	PipelineClassify      = "classify"
	PipelineLivenessCheck = "liveness_check"
)

// Classified job statuses
const (
	StatusPending           = "pending"
	StatusNonTechnical      = "non_technical"
	StatusFilteredLocation  = "filtered_location"
	StatusFilteredLevel     = "filtered_level"
	StatusFilteredRelevance = "filtered_relevance"
	StatusDead              = "dead"
	StatusAccepted          = "accepted"
)

// Outbox task statuses
const (
	TaskWaiting    = "waiting"
	TaskProcessing = "processing"
	TaskDone       = "done"
	TaskFailed     = "failed"
)
