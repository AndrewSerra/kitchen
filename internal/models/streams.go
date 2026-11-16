package models

import "time"

type Stream struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	UserID    int       `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	Streams []*Stream `json:"streams"`
	Success bool      `json:"success"`
	Message string    `json:"message"`
}

type DeleteStreamResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
