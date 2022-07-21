package panoptes

import (
	"github.com/google/uuid"

	"golang.org/x/sys/windows"
)

//This will convert exactly the same as bi-zone/etw lib but i wrap it just in case we change it later on..

type SessionOptions struct {
	Level int `json:"level"`

	// If MatchAnyKeyword is not set the session will receive ALL possible
	// events (which is equivalent setting all 64 bits to 1).
	// https://docs.microsoft.com/en-us/windows/win32/api/evntrace/nf-evntrace-enabletraceex2#remarks
	MatchAnyKeyword string `json:"matchAnyKeyword"`

	// This mask is not used if MatchAnyKeyword is zero.
	// all keywords must match ..
	MatchAllKeyword string `json:"matchAllKeyword"`

	FilterEventIds []uint16 `json:"eventIds"`
}

type ReportMode string

const (
	Json       ReportMode = "json"
	GoCallback            = "go"
)

type Provider struct {
	Guid     string         `json:"guid"`
	Name     string         `json:"name"`
	Options  SessionOptions `json:"options"`
	Report   ReportMode     `json:"report"`
	winGuid  windows.GUID
	Disabled bool `json:"disabled"`
}

type Event struct {
	EventData  map[string]interface{}
	Guid       string //to avoid header reconstruction on each event
	Name       string
	Marshalled []byte //json marshall result
	SessionId  uuid.UUID
}
