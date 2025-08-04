package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type User struct {
	Token        uuid.UUID `json:"token"`
	Login        string    `json:"login"`
	Password     string    `json:"pswd"`
	PasswordHash string    `json:"-"`
}
type Document struct {
	ID       uuid.UUID       `json:"id"`
	Name     string          `json:"name"`
	File     bool            `json:"file"`
	Public   bool            `json:"public"`
	Token    string          `json:"-"`
	Mime     string          `json:"mime"`
	Grant    []string        `json:"grant"`
	Created  time.Time       `json:"created"`
	Owner    string          `json:"-"`
	JsonData json.RawMessage `json:"json,omitempty"`
}
