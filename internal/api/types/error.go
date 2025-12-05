package model

type Error struct {
	Code    *int    `json:"code,omitempty"`
	Error   *string `json:"error,omitempty"`
	Message *string `json:"message,omitempty"`
}
