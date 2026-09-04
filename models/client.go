package models

import "time"

type Client struct {
	ID        int64
	Name      string
	Protocol  string
	UUID      string
	Password  string
	CreatedAt time.Time
}
