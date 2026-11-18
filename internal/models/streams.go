package models

import "time"

type Stream struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	UserID    int       `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Streams = []*Stream

type StreamListFilter struct {
	Name   string
	ID     int
	Limit  int
	Offset int
	SortBy string
	SortDir string
}

type CreateStreamRequest struct {
	Name string `json:"name" required:"true"`
}

type CreateStreamResponse struct {
	Stream  *Stream `json:"stream"`
	Success bool    `json:"success"`
	Message string  `json:"message"`
}

type GetStreamResponse struct {
	Stream  *Stream `json:"stream"`
	Success bool    `json:"success"`
	Message string  `json:"message"`
}

type GetAllStreamsResponse struct {
	Streams Streams `json:"streams"`
	Success bool    `json:"success"`
	Message string  `json:"message"`
}

type DeleteStreamResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
