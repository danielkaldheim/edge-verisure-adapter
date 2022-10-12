package router

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/edgeapp"
	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/verisure/model"
	"github.com/thingsplex/verisure/verisure"
)

type FromFimpRouter struct {
	inboundMsgCh fimpgo.MessageCh
	mqt          *fimpgo.MqttTransport
	instanceId   string
	appLifecycle *edgeapp.Lifecycle
	configs      *model.Configs
	client       *verisure.Client
	states       *model.States
}

func NewFromFimpRouter(mqt *fimpgo.MqttTransport, appLifecycle *edgeapp.Lifecycle, configs *model.Configs, client *verisure.Client, states *model.States) *FromFimpRouter {
	fc := FromFimpRouter{inboundMsgCh: make(fimpgo.MessageCh, 5), mqt: mqt, appLifecycle: appLifecycle, configs: configs, client: client, states: states}
	fc.mqt.RegisterChannel("ch1", fc.inboundMsgCh)
	return &fc
}

func (fc *FromFimpRouter) Start() {

	// TODO: Choose either adapter or app topic

	// ------ Adapter topics ---------------------------------------------
	fc.mqt.Subscribe(fmt.Sprintf("pt:j1/+/rt:dev/rn:%s/ad:1/#", model.ServiceName))
	fc.mqt.Subscribe(fmt.Sprintf("pt:j1/+/rt:ad/rn:%s/ad:1", model.ServiceName))

	// ------ Application topic -------------------------------------------
	//fc.mqt.Subscribe(fmt.Sprintf("pt:j1/+/rt:app/rn:%s/ad:1",model.ServiceName))

	go func(msgChan fimpgo.MessageCh) {
		for {
			select {
			case newMsg := <-msgChan:
				fc.routeFimpMessage(newMsg)
			}
		}
	}(fc.inboundMsgCh)
}

func (fc *FromFimpRouter) routeFimpMessage(newMsg *fimpgo.Message) {
	log.Debugf("New fimp msg . cmd = %s, %s", newMsg.Payload.Type, newMsg.Payload.Service)

	addr := strings.Replace(newMsg.Addr.ServiceAddress, "_0", "", 1)
	if fc.configs.Installation != "" {
		fc.client.SetGIID(fc.configs.Installation)
	}
	ns := model.NetworkService{}
	switch newMsg.Payload.Service {
	case "door_lock":
		addr = strings.Replace(addr, "l", "", 1)
		switch newMsg.Payload.Type {
		case "cmd.lock.set":
			lockPin := fmt.Sprintf("%d", fc.configs.LockPin)
			if lockPin == "" || lockPin == "0" {
				log.Error("missing lock pin")
				return
			}

			isLocking, err := newMsg.Payload.GetBoolValue()
			if err != nil {
				log.Error(err)
			}

			smartLock := fc.states.GetSmartLockByDeviceLabel(addr)
			if smartLock != nil {
				deviceId := strings.ReplaceAll(smartLock.Device.DeviceLabel, " ", "")
				stateVal := &model.LockState{}
				trueVal := true
				falseVal := false
				if isLocking {
					log.Debug("Locking")
					err := fc.client.LockSmartLock(smartLock.Device.DeviceLabel, lockPin)
					if err != nil {
						log.Error(err)
						// TODO: Handle error response
						return
					}
					stateVal.IsSecured = &trueVal
				} else {
					log.Debug("Unlocking")
					err := fc.client.UnlockSmartLock(smartLock.Device.DeviceLabel, lockPin)
					if err != nil {
						log.Error(err)
						// TODO: Handle error response
						return
					}
					stateVal.IsSecured = &falseVal
				}

				adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "door_lock", ServiceAddress: deviceId}
				msg := fimpgo.NewMessage("evt.lock.report", "door_lock", fimpgo.VTypeBoolMap, stateVal, nil, nil, nil)
				fc.mqt.Publish(adr, msg)
			}
		case "cmd.lock.get_report":
			smartLock := fc.states.GetSmartLockByDeviceLabel(addr)
			locks, err := fc.client.FetchSmartLock()
			if err != nil {
				log.Error(err)
			}
			if len(locks) > 0 {
				for _, l := range locks {
					if l.Device.DeviceLabel == smartLock.Device.DeviceLabel {
						if smartLock.EventTime == l.EventTime {
							break
						}
						deviceId := strings.ReplaceAll(l.Device.DeviceLabel, " ", "")
						stateVal := &model.LockState{}

						trueVal := true
						falseVal := false
						if l.LockStatus == "LOCKED" {
							stateVal.IsSecured = &trueVal
						} else {
							stateVal.IsSecured = &falseVal
						}

						props := fimpgo.Props{}
						if l.LockMethod == "CODE" {
							props["lock_type"] = "PIN"
						} else {
							props["lock_type"] = "KEY"
						}

						adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: model.ServiceName, ResourceAddress: "1", ServiceName: "door_lock", ServiceAddress: deviceId}
						msg := fimpgo.NewMessage("evt.lock.report", "door_lock", fimpgo.VTypeBoolMap, stateVal, props, nil, nil)
						fc.mqt.Publish(adr, msg)
					}
				}
				fc.states.SmartLocks = locks
				fc.states.SaveToFile()
			}
		}

	case model.ServiceName:
		adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: model.ServiceName, ResourceAddress: "1"}
		switch newMsg.Payload.Type {
		case "cmd.auth.login":
			authReq := model.Login{}
			err := newMsg.Payload.GetObjectValue(&authReq)
			if err != nil {
				log.Error("Incorrect login message ")
				return
			}
			status := model.AuthStatus{
				Status:    edgeapp.AuthStateAuthenticated,
				ErrorText: "",
				ErrorCode: "",
			}
			if authReq.Username != "" && authReq.Password != "" {
				fc.appLifecycle.SetAuthState(edgeapp.AuthStateInProgress)

				fc.states.ClearState()

				err = fc.client.Login(authReq.Username, authReq.Password)
				if err != nil {
					log.Error(err)
					status.Status = "ERROR"
					status.ErrorText = "Invalid username or password"
					fc.appLifecycle.SetAuthState(edgeapp.AuthStateNotAuthenticated)
				} else {
					fc.appLifecycle.SetAuthState(edgeapp.AuthStateAuthenticated)
				}
			} else {
				status.Status = "ERROR"
				status.ErrorText = "Empty username or password"
				fc.appLifecycle.SetAuthState(edgeapp.AuthStateNotAuthenticated)
			}

			msg := fimpgo.NewMessage("evt.auth.status_report", model.ServiceName, fimpgo.VTypeObject, status, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.auth.set_tokens":
			authReq := model.SetTokens{}
			err := newMsg.Payload.GetObjectValue(&authReq)
			if err != nil {
				log.Error("Incorrect login message ")
				return
			}
			status := model.AuthStatus{
				Status:    edgeapp.AuthStateAuthenticated,
				ErrorText: "",
				ErrorCode: "",
			}
			if authReq.AccessToken != "" && authReq.RefreshToken != "" {
				// TODO: This is an example . Add your logic here or remove
			} else {
				status.Status = "ERROR"
				status.ErrorText = "Empty username or password"
			}
			fc.appLifecycle.SetAuthState(edgeapp.AuthStateAuthenticated)
			msg := fimpgo.NewMessage("evt.auth.status_report", model.ServiceName, fimpgo.VTypeObject, status, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.app.get_manifest":
			mode, err := newMsg.Payload.GetStringValue()
			if err != nil {
				log.Error("Incorrect request format ")
				return
			}
			manifest := edgeapp.NewManifest()
			err = manifest.LoadFromFile(filepath.Join(fc.configs.GetDefaultDir(), "app-manifest.json"))
			if err != nil {
				log.Error("Failed to load manifest file .Error :", err.Error())
				return
			}
			if mode == "manifest_state" {
				manifest.AppState = *fc.appLifecycle.GetAllStates()
				manifest.ConfigState = fc.configs
			}
			accessCookie := fc.states.GetCookieByName("vs-access")
			if accessCookie != nil && accessCookie.Expires.After(time.Now()) {
				fc.appLifecycle.SetAuthState(edgeapp.AuthStateAuthenticated)
				fc.appLifecycle.SetConnectionState(edgeapp.ConnStateConnected)

				installations, err := fc.client.FetchAllInstallations()
				if err != nil {
					log.Error(err)
				}

				if installations != nil {
					fc.states.Installations = installations
					fc.states.SaveToFile()

					var installationSelect []interface{}
					manifest.Configs[0].ValT = "string"
					manifest.Configs[0].UI.Type = "select_horizontal"
					for i := 0; i < len(installations); i++ {
						installationSelect = append(installationSelect, map[string]interface{}{"val": fc.states.Installations[i].Giid, "label": map[string]interface{}{"en": fc.states.Installations[i].Alias}})
					}
					manifest.Configs[0].UI.Select = installationSelect
				} else {
					manifest.Configs[0].ValT = "string"
					manifest.Configs[0].UI.Type = "input_readonly"
					var val edgeapp.Value
					val.Default = "Failed to fetch installations"
					manifest.Configs[0].Val = val
				}
			} else {
				manifest.Configs[0].ValT = "string"
				manifest.Configs[0].UI.Type = "input_readonly"
				var val edgeapp.Value
				val.Default = "You need to login first"
				manifest.Configs[0].Val = val
			}

			msg := fimpgo.NewMessage("evt.app.manifest_report", model.ServiceName, fimpgo.VTypeObject, manifest, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				if err := fc.mqt.Publish(adr, msg); err != nil {
					log.Error(err)
				}
			}

		case "cmd.app.get_state":
			msg := fimpgo.NewMessage("evt.app.manifest_report", model.ServiceName, fimpgo.VTypeObject, fc.appLifecycle.GetAllStates(), nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.config.get_extended_report":

			msg := fimpgo.NewMessage("evt.config.extended_report", model.ServiceName, fimpgo.VTypeObject, fc.configs, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.config.extended_set":
			conf := model.Configs{}
			err := newMsg.Payload.GetObjectValue(&conf)
			if err != nil {
				// TODO: This is an example . Add your logic here or remove
				log.Error("Can't parse configuration object")
				return
			}
			fc.configs.Installation = conf.Installation

			fc.client.SetGIID(conf.Installation)

			fc.configs.LockPin = conf.LockPin
			fc.configs.SaveToFile()
			log.Debugf("App reconfigured . New parameters : %v", fc.configs)
			// TODO: This is an example . Add your logic here or remove

			if conf.Installation != "" {
				fc.client.UpdateToken()
				climates, err := fc.client.FetchClimate()
				if err != nil {
					log.Error(err)
				}
				for _, climate := range climates {
					inclReport := ns.SendClimateInclusionReport(climate)
					if err != nil {
						log.Error(err)
					}

					msg2 := fimpgo.NewMessage("evt.thing.inclusion_report", model.ServiceName, fimpgo.VTypeObject, inclReport, nil, nil, nil)
					adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: model.ServiceName, ResourceAddress: "1"}
					fc.mqt.Publish(&adr, msg2)
				}

				doorsAndWindows, err := fc.client.FetchDoorWindow()
				if err != nil {
					log.Error(err)
				}

				for _, doorsAndWindow := range doorsAndWindows {
					inclReport := ns.SendDoorWindowInclusionReport(doorsAndWindow)
					if err != nil {
						log.Error(err)
					}

					msg2 := fimpgo.NewMessage("evt.thing.inclusion_report", model.ServiceName, fimpgo.VTypeObject, inclReport, nil, nil, nil)
					adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: model.ServiceName, ResourceAddress: "1"}
					fc.mqt.Publish(&adr, msg2)
				}

				smartLocks, err := fc.client.FetchSmartLock()
				if err != nil {
					log.Error(err)
				}

				for _, smartLock := range smartLocks {
					inclReport := ns.SendSmartLockInclusionReport(smartLock)
					if err != nil {
						log.Error(err)
					}

					msg2 := fimpgo.NewMessage("evt.thing.inclusion_report", model.ServiceName, fimpgo.VTypeObject, inclReport, nil, nil, nil)
					adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: model.ServiceName, ResourceAddress: "1"}
					fc.mqt.Publish(&adr, msg2)
				}

				fc.appLifecycle.SetAppState(edgeapp.AppStateRunning, nil)
				fc.appLifecycle.SetConfigState(edgeapp.ConfigStateConfigured)
			}

			configReport := model.ConfigReport{
				OpStatus: "ok",
				AppState: *fc.appLifecycle.GetAllStates(),
			}
			msg := fimpgo.NewMessage("evt.app.config_report", model.ServiceName, fimpgo.VTypeObject, configReport, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				log.Error(err)
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.log.set_level":
			// Configure log level
			level, err := newMsg.Payload.GetStringValue()
			if err != nil {
				return
			}
			logLevel, err := log.ParseLevel(level)
			if err == nil {
				log.SetLevel(logLevel)
				fc.configs.LogLevel = level
				fc.configs.SaveToFile()
			}
			log.Info("Log level updated to = ", logLevel)

		case "cmd.system.reconnect":
			// This is optional operation.
			//val := map[string]string{"status":status,"error":errStr}
			val := edgeapp.ButtonActionResponse{
				Operation:       "cmd.system.reconnect",
				OperationStatus: "ok",
				Next:            "config",
				ErrorCode:       "",
				ErrorText:       "",
			}
			msg := fimpgo.NewMessage("evt.app.config_action_report", model.ServiceName, fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.app.factory_reset":
			fc.states.ClearState()
			val := edgeapp.ButtonActionResponse{
				Operation:       "cmd.app.factory_reset",
				OperationStatus: "ok",
				Next:            "config",
				ErrorCode:       "",
				ErrorText:       "",
			}
			fc.appLifecycle.SetConfigState(edgeapp.ConfigStateNotConfigured)
			fc.appLifecycle.SetAppState(edgeapp.AppStateNotConfigured, nil)
			fc.appLifecycle.SetAuthState(edgeapp.AuthStateNotAuthenticated)
			msg := fimpgo.NewMessage("evt.app.config_action_report", model.ServiceName, fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}
		case "cmd.app.uninstall":
			// TODO: The message is sent to the app from fhbutler before performing package uninstall operation

		case "cmd.network.get_all_nodes":
			// TODO: This is an example . Add your logic here or remove
		case "cmd.thing.get_inclusion_report":
			fc.client.UpdateToken()
			climates, err := fc.client.FetchClimate()
			if err != nil {
				log.Error(err)
			}
			for _, climate := range climates {
				inclReport := ns.SendClimateInclusionReport(climate)
				if err != nil {
					log.Error(err)
				}

				msg2 := fimpgo.NewMessage("evt.thing.inclusion_report", model.ServiceName, fimpgo.VTypeObject, inclReport, nil, nil, nil)
				adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: model.ServiceName, ResourceAddress: "1"}
				fc.mqt.Publish(&adr, msg2)
			}

			doorsAndWindows, err := fc.client.FetchDoorWindow()
			if err != nil {
				log.Error(err)
			}

			for _, doorsAndWindow := range doorsAndWindows {
				inclReport := ns.SendDoorWindowInclusionReport(doorsAndWindow)
				if err != nil {
					log.Error(err)
				}

				msg2 := fimpgo.NewMessage("evt.thing.inclusion_report", model.ServiceName, fimpgo.VTypeObject, inclReport, nil, nil, nil)
				adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: model.ServiceName, ResourceAddress: "1"}
				fc.mqt.Publish(&adr, msg2)
			}

			smartLocks, err := fc.client.FetchSmartLock()
			if err != nil {
				log.Error(err)
			}

			for _, smartLock := range smartLocks {
				inclReport := ns.SendSmartLockInclusionReport(smartLock)
				if err != nil {
					log.Error(err)
				}

				msg2 := fimpgo.NewMessage("evt.thing.inclusion_report", model.ServiceName, fimpgo.VTypeObject, inclReport, nil, nil, nil)
				adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: model.ServiceName, ResourceAddress: "1"}
				fc.mqt.Publish(&adr, msg2)
			}

		case "cmd.thing.inclusion":
			//flag , _ := newMsg.Payload.GetBoolValue()
			// TODO: This is an example . Add your logic here or remove
		case "cmd.thing.delete":
			// remove device from network
			val, err := newMsg.Payload.GetStrMapValue()
			if err != nil {
				log.Error("Wrong msg format")
				return
			}
			deviceID, ok := val["address"]
			if ok {
				val := map[string]interface{}{
					"address": deviceID,
				}
				adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: model.ServiceName, ResourceAddress: "1"}
				msg := fimpgo.NewMessage("evt.thing.exclusion_report", model.ServiceName, fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
				fc.mqt.Publish(adr, msg)
				log.Info("Device with deviceID: ", deviceID, " has been removed from network.")
			} else {
				log.Error("Incorrect address")

			}
		}

	}

}
