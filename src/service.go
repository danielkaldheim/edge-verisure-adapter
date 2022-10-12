package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/discovery"
	"github.com/futurehomeno/fimpgo/edgeapp"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/verisure/model"
	"github.com/thingsplex/verisure/router"
	"github.com/thingsplex/verisure/verisure"
)

func main() {
	var workDir string
	flag.StringVar(&workDir, "c", "", "Work dir")
	flag.Parse()
	if workDir == "" {
		workDir = "./"
	} else {
		fmt.Println("Work dir ", workDir)
	}
	appLifecycle := edgeapp.NewAppLifecycle()
	configs := model.NewConfigs(workDir)
	err := configs.LoadFromFile()
	if err != nil {
		fmt.Print(err)
		panic("Can't load config file.")
	}

	states := model.NewStates(workDir)
	err = states.LoadFromFile()
	if err != nil {
		fmt.Print(err)
		panic("Can't load state file.")
	}

	edgeapp.SetupLog(configs.LogFile, configs.LogLevel, configs.LogFormat)
	log.Info("--------------Starting verisure----------------")
	log.Info("Work directory : ", configs.WorkDir)

	appLifecycle.SetAppState(edgeapp.AppStateNotConfigured, nil)

	mqtt := fimpgo.NewMqttTransport(configs.MqttServerURI, configs.MqttClientIdPrefix, configs.MqttUsername, configs.MqttPassword, true, 1, 1)
	err = mqtt.Start()
	responder := discovery.NewServiceDiscoveryResponder(mqtt)
	responder.RegisterResource(model.GetDiscoveryResource())
	responder.Start()

	vsureService, _ := verisure.NewClient(states)

	fimpRouter := router.NewFromFimpRouter(mqtt, appLifecycle, configs, vsureService, states)
	fimpRouter.Start()
	//------------------ Remote API check -- !!!!IMPORTANT!!!!-------------
	// The app MUST perform remote API availability check.
	// During gateway boot process the app might be started before network is initialized or another local app booted.
	// Remove that codee if the app is not dependent from local network internet availability.
	//------------------ Sample code --------------------------------------
	sys := edgeapp.NewSystemCheck()
	sys.WaitForInternet(time.Second * 10)
	//---------------------------------------------------------------------
	if err != nil {
		log.Error("Can't connect to broker. Error:", err.Error())
	} else {
		log.Info("Connected")
	}
	appLifecycle.SetAppState(edgeapp.AppStateRunning, nil)

	// PollString := configs.PollTimeMin
	PollString := "1"
	PollTime, err := strconv.Atoi(PollString)
	if err != nil {
		PollTime = 5
	}
	for {
		appLifecycle.WaitForState("main", edgeapp.SystemEventTypeState, edgeapp.AppStateRunning)
		log.Info("Starting ticker")
		ticker := time.NewTicker(time.Duration(PollTime) * time.Minute)
		for ; true; <-ticker.C {
			if configs.Installation == "" {
				log.Debug("No installation is setup")
				continue
			}

			vsureService.SetGIID(configs.Installation)

			if err := vsureService.UpdateToken(); err != nil {
				log.Error(err)
				appLifecycle.SetConnectionState(edgeapp.ConnStateDisconnected)
				continue
			}

			appLifecycle.SetConnectionState(edgeapp.ConnStateConnected)

			installationState, err := vsureService.FetchInstallationState()
			if err != nil {
				log.Error(err)
				continue
			}

			if installationState != nil {

				for _, climate := range installationState.Climates {
					deviceId := strings.ReplaceAll(climate.Device.DeviceLabel, " ", "")

					bk := states.GetClimateByDeviceLabel(climate.Device.DeviceLabel)
					if bk != nil && climate.TemperatureTimestamp == bk.TemperatureTimestamp {
						continue
					}
					tempVal := climate.TemperatureValue
					props := fimpgo.Props{}
					props["unit"] = "C"

					adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "sensor_temp", ServiceAddress: deviceId}
					msg := fimpgo.NewMessage("evt.sensor.report", "sensor_temp", fimpgo.VTypeFloat, tempVal, props, nil, nil)
					mqtt.Publish(adr, msg)

					if climate.HumidityEnabled {
						humidityVal := climate.HumidityValue
						props := fimpgo.Props{}
						props["unit"] = "%"

						adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "sensor_humid", ServiceAddress: deviceId}
						msg := fimpgo.NewMessage("evt.sensor.report", "sensor_humid", fimpgo.VTypeFloat, humidityVal, props, nil, nil)
						mqtt.Publish(adr, msg)
					}
				}
				states.Climates = installationState.Climates

				for _, daw := range installationState.DoorWindows {
					deviceId := strings.ReplaceAll(daw.Device.DeviceLabel, " ", "")
					bk := states.GetDoorWindowByDeviceLabel(daw.Device.DeviceLabel)
					if bk != nil && daw.ReportTime == bk.ReportTime {
						continue
					}

					stateVal := false
					if daw.State == "OPEN" {
						stateVal = true
					}

					adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "sensor_contact", ServiceAddress: deviceId}
					msg := fimpgo.NewMessage("evt.open.report", "sensor_contact", fimpgo.VTypeBool, stateVal, nil, nil, nil)
					mqtt.Publish(adr, msg)
				}

				states.DoorWindows = installationState.DoorWindows

				for _, smartLock := range installationState.SmartLocks {
					deviceId := strings.ReplaceAll(smartLock.Device.DeviceLabel, " ", "")
					bk := states.GetSmartLockByDeviceLabel(smartLock.Device.DeviceLabel)
					if bk != nil && smartLock.EventTime == bk.EventTime {
						continue
					}

					stateVal := &model.LockState{}

					trueVal := true
					falseVal := false
					if smartLock.LockStatus == "LOCKED" {
						stateVal.IsSecured = &trueVal
					} else {
						stateVal.IsSecured = &falseVal
					}

					props := fimpgo.Props{}
					if smartLock.LockMethod == "CODE" {
						props["lock_type"] = "PIN"
					} else {
						props["lock_type"] = "KEY"
					}

					adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "door_lock", ServiceAddress: deviceId}
					msg := fimpgo.NewMessage("evt.lock.report", "door_lock", fimpgo.VTypeBoolMap, stateVal, props, nil, nil)
					mqtt.Publish(adr, msg)
				}

				states.SmartLocks = installationState.SmartLocks

				states.SaveToFile()
			}
		}
		appLifecycle.WaitForState("main", edgeapp.SystemEventTypeState, edgeapp.AppStateNotConfigured)
	}

	mqtt.Stop()
	time.Sleep(5 * time.Second)
}
