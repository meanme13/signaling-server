package types

import (
	"github.com/go-playground/validator/v10"
)

var Validate = validator.New()

type SignalMessage struct {
	Type         string `json:"type" validate:"required,oneof=offer answer candidate phrase status warning msg update_limit signal call_initiate call_accept call_end"`
	Msg          string `json:"msg,omitempty"`
	KeyPhrase    string `json:"keyPhrase,omitempty" validate:"omitempty,min=1"`
	Name         string `json:"name,omitempty"`
	From         string `json:"from,omitempty"`
	To           string `json:"to,omitempty"`
	SDP          string `json:"sdp,omitempty"`
	ICE          string `json:"ice,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	RoomID       string `json:"roomID,omitempty"`
	CallType     string `json:"call_type,omitempty" validate:"omitempty,oneof=audio video"`
	CallDuration int    `json:"call_duration,omitempty"`
	CallId       string `json:"callId,omitempty" validate:"omitempty,uuid4"`
}

type InitMessage struct {
	KeyPhrase    string `json:"keyPhrase" validate:"required,min=1"`
	Name         string `json:"name,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	AESKey       string `json:"aesKey" validate:"required"`
	ClientPubKey string `json:"clientPubKey,omitempty" validate:"required"`
}
