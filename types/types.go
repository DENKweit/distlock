package types

type AcquireReturn struct {
	SessionID string `json:"sessionId"`
	Success   bool   `json:"success"`
}

type ReleaseReturn struct {
	Success bool `json:"success"`
}

type SetReturn struct {
	Success bool `json:"success"`
}

type GetReturn struct {
	Success bool   `json:"success"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

type GetMRequest struct {
	Keys []string `json:"keys"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type KeyValueSuccess struct {
	KeyValue
	Success bool `json:"success"`
}

type GetMReturn struct {
	Entries []KeyValueSuccess `json:"entries"`
}

type SetMRequest struct {
	Entries []KeyValue `json:"entries"`
}

type SetMReturn struct {
	Success bool `json:"success"`
}

type StatusReturn struct {
	Running bool `json:"running"`
}

type IntReturn struct {
	Value   int64  `json:"value"`
	Success bool   `json:"success"`
	Op      string `json:"op"`
}

type IntOpType string

const (
	IntOpTypeInc IntOpType = "inc"
	IntOpTypeDec IntOpType = "dec"
	IntOpTypeSet IntOpType = "set"
	IntOpTypeGet IntOpType = "get"
)

type MutexReturn struct {
	Success bool `json:"success"`
}
