package model

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/futurehomeno/fimpgo/utils"
	log "github.com/sirupsen/logrus"
)

type States struct {
	path         string
	LogFile      string `json:"log_file"`
	LogLevel     string `json:"log_level"`
	LogFormat    string `json:"log_format"`
	WorkDir      string `json:"-"`
	ConfiguredAt string `json:"configuret_at"`
	ConfiguredBy string `json:"configures_by"`

	Cookies  []*http.Cookie `json:"cookies"`
	Username string         `json:"username"`
	GIID     string         `json:"giid"`

	Installations []Installation     `json:"installations"`
	Climates      []ClimateDevice    `json:"climates"`
	DoorWindows   []DoorWindowDevice `json:"doorWindows"`
	SmartLocks    []SmartLockDevice  `json:"smartLocks"`
}

func NewStates(workDir string) *States {
	state := &States{WorkDir: workDir}
	state.path = filepath.Join(workDir, "data", "state.json")
	if !utils.FileExists(state.path) {
		log.Info("State file doesn't exist.Loading default state")
		defaultStateFile := filepath.Join(workDir, "defaults", "state.json")
		err := utils.CopyFile(defaultStateFile, state.path)
		if err != nil {
			fmt.Print(err)
			panic("Can't copy state file.")
		}
	}
	return state
}

func (st *States) ClearState() error {
	st.Cookies = nil
	st.Username = ""
	st.GIID = ""

	st.Installations = nil
	st.Climates = nil
	st.DoorWindows = nil
	st.SmartLocks = nil

	return st.SaveToFile()
}

func (st *States) GetClimateByDeviceLabel(deviceLabel string) *ClimateDevice {
	deviceLabel = strings.ReplaceAll(deviceLabel, " ", "")
	for _, climate := range st.Climates {
		if deviceLabel == strings.ReplaceAll(climate.Device.DeviceLabel, " ", "") {
			return &climate
		}
	}
	return nil
}

func (st *States) GetDoorWindowByDeviceLabel(deviceLabel string) *DoorWindowDevice {
	deviceLabel = strings.ReplaceAll(deviceLabel, " ", "")
	for _, doorWindow := range st.DoorWindows {
		if deviceLabel == strings.ReplaceAll(doorWindow.Device.DeviceLabel, " ", "") {
			return &doorWindow
		}
	}
	return nil
}

func (st *States) GetSmartLockByDeviceLabel(deviceLabel string) *SmartLockDevice {
	deviceLabel = strings.ReplaceAll(deviceLabel, " ", "")
	for _, smartLock := range st.SmartLocks {
		if deviceLabel == strings.ReplaceAll(smartLock.Device.DeviceLabel, " ", "") {
			return &smartLock
		}
	}
	return nil
}

func (st *States) GetCookieByName(name string) *http.Cookie {
	for _, cookie := range st.Cookies {
		if name == cookie.Name {
			return cookie
		}
	}
	return nil
}

func (st *States) LoadFromFile() error {
	stateFileBody, err := ioutil.ReadFile(st.path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(stateFileBody, st)
	if err != nil {
		return err
	}
	return nil
}

func (st *States) SaveToFile() error {
	st.ConfiguredBy = "auto"
	st.ConfiguredAt = time.Now().Format(time.RFC3339)
	bpayload, err := json.Marshal(st)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(st.path, bpayload, 0664)
	if err != nil {
		return err
	}
	return err
}

func (st *States) GetDataDir() string {
	return filepath.Join(st.WorkDir, "data")
}

func (st *States) GetDefaultDir() string {
	return filepath.Join(st.WorkDir, "defaults")
}

func (st *States) LoadDefaults() error {
	stateFile := filepath.Join(st.WorkDir, "data", "state.json")
	os.Remove(stateFile)
	log.Info("State file doesn't exist.Loading default state")
	defaultStateFile := filepath.Join(st.WorkDir, "defaults", "state.json")
	return utils.CopyFile(defaultStateFile, stateFile)
}
