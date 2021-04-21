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

type StatusReturn struct {
	Running bool `json:"running"`
}
