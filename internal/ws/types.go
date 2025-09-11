package ws

import (
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

type SignalMessage struct {
	Type      string `json:"type" validate:"required,oneof=offer answer candidate phrase status warning msg update_limit signal"`
	Msg       string `json:"msg,omitempty"`
	KeyPhrase string `json:"keyPhrase,omitempty" validate:"omitempty,min=1"`
	Name      string `json:"name,omitempty"`
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	SDP       string `json:"sdp,omitempty"`
	ICE       string `json:"ice,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	RoomID    string `json:"roomID,omitempty"`
}

type InitMessage struct {
	KeyPhrase string `json:"keyPhrase" validate:"required,min=1"`
	Name      string `json:"name,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}
