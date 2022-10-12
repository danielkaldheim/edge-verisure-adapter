package model

import "time"

type ErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Errors struct {
	Message   string          `json:"message"`
	Locations []ErrorLocation `json:"locations"`
	Path      []string        `json:"path"`
	Data      ErrorData       `json:"data"`
}

type ErrorData struct {
	Status       int    `json:"status"`
	LogTraceID   string `json:"logTraceId"`
	ErrorGroup   string `json:"errorGroup"`
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

type Data struct {
	Account      *Account      `json:"account"`
	Installation *Installation `json:"installation"`
}

type Address struct {
	Street       string `json:"street"`
	City         string `json:"city"`
	PostalNumber string `json:"postalNumber"`
	Typename     string `json:"__typename"`
}

type Account struct {
	Installations []Installation `json:"installations"`
	Typename      string         `json:"__typename"`
}

type Gui struct {
	Label    string `json:"label"`
	Typename string `json:"__typename"`
}

type Device struct {
	DeviceLabel string `json:"deviceLabel"`
	Area        string `json:"area"`
	Gui         Gui    `json:"gui"`
	Typename    string `json:"__typename"`
}

type Threshold struct {
	AboveMaxAlert bool   `json:"aboveMaxAlert"`
	BelowMinAlert bool   `json:"belowMinAlert"`
	SensorType    string `json:"sensorType"`
	Typename      string `json:"__typename"`
}

type ClimateDevice struct {
	Device               Device      `json:"device"`
	HumidityEnabled      bool        `json:"humidityEnabled"`
	HumidityTimestamp    *time.Time  `json:"humidityTimestamp"`
	HumidityValue        *float64    `json:"humidityValue"`
	TemperatureTimestamp time.Time   `json:"temperatureTimestamp"`
	TemperatureValue     float64     `json:"temperatureValue"`
	Thresholds           []Threshold `json:"thresholds"`
	Typename             string      `json:"__typename"`
}

type DoorWindowDevice struct {
	Device     Device      `json:"device"`
	Type       interface{} `json:"type"`
	Area       string      `json:"area"`
	State      string      `json:"state"`
	Wired      bool        `json:"wired"`
	ReportTime time.Time   `json:"reportTime"`
	Typename   string      `json:"__typename"`
}

type User struct {
	Name     string `json:"name"`
	Typename string `json:"__typename"`
}

type SmartLockDevice struct {
	LockStatus   string    `json:"lockStatus"`
	DoorState    string    `json:"doorState"`
	LockMethod   string    `json:"lockMethod"`
	EventTime    time.Time `json:"eventTime"`
	DoorLockType string    `json:"doorLockType"`
	SecureMode   string    `json:"secureMode"`
	Device       Device    `json:"device"`
	User         User      `json:"user"`
	Typename     string    `json:"__typename"`
}

type ArmState struct {
	Type       interface{} `json:"type"`
	StatusType string      `json:"statusType"`
	Date       time.Time   `json:"date"`
	Name       string      `json:"name"`
	ChangedVia string      `json:"changedVia"`
	Typename   string      `json:"__typename"`
}

type UserTracking struct {
	IsCallingUser            bool      `json:"isCallingUser"`
	WebAccount               string    `json:"webAccount"`
	Status                   string    `json:"status"`
	XbnContactID             string    `json:"xbnContactId"`
	CurrentLocationName      string    `json:"currentLocationName"`
	DeviceID                 string    `json:"deviceId"`
	Name                     string    `json:"name"`
	Initials                 string    `json:"initials"`
	CurrentLocationTimestamp time.Time `json:"currentLocationTimestamp"`
	DeviceName               string    `json:"deviceName"`
	CurrentLocationID        string    `json:"currentLocationId"`
	Typename                 string    `json:"__typename"`
}

type Installation struct {
	Giid          string  `json:"giid"`
	Alias         string  `json:"alias"`
	CustomerType  string  `json:"customerType"`
	DealerID      string  `json:"dealerId"`
	PinCodeLength int     `json:"pinCodeLength"`
	Locale        string  `json:"locale"`
	Address       Address `json:"address"`

	Climates      []ClimateDevice    `json:"climates,omitempty"`
	DoorWindows   []DoorWindowDevice `json:"doorWindows,omitempty"`
	SmartLocks    []SmartLockDevice  `json:"smartLocks,omitempty"`
	UserTrackings []UserTracking     `json:"userTrackings,omitempty"`
	ArmState      *ArmState          `json:"armState,omitempty"`
	Typename      string             `json:"__typename"`
}

type LockState struct {
	IsSecured     *bool `json:"is_secured,omitempty"`
	DoorIsClosed  *bool `json:"door_is_closed,omitempty"`
	BoltIsLocked  *bool `json:"bolt_is_locked,omitempty"`
	LatchIsClosed *bool `json:"latch_is_closed,omitempty"`
}
