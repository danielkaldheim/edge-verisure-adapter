package model

import (
	"fmt"
	"strings"

	"github.com/futurehomeno/fimpgo/fimptype"
)

type NetworkService struct {
}

func (ns *NetworkService) SendClimateInclusionReport(device ClimateDevice) fimptype.ThingInclusionReport {

	var name, manufacturer string
	var deviceAddr string
	services := []fimptype.Service{}

	sensorInterfaces := []fimptype.Interface{{
		Type:      "in",
		MsgType:   "cmd.sensor.get_report",
		ValueType: "null",
		Version:   "1",
	}, {
		Type:      "out",
		MsgType:   "evt.sensor.report",
		ValueType: "float",
		Version:   "1",
	}}

	tempSensorService := fimptype.Service{
		Name:    "sensor_temp",
		Alias:   "Temperature sensor",
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:1/sv:sensor_temp/ad:", ServiceName),
		Enabled: true,
		Groups:  []string{"ch_0"},
		Props: map[string]interface{}{
			"sup_units": []string{"C"},
		},
		Tags:             nil,
		PropSetReference: "",
		Interfaces:       sensorInterfaces,
	}

	humSensorService := fimptype.Service{
		Name:    "sensor_humid",
		Alias:   "Relative humidity sensor",
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:1/sv:sensor_humid/ad:", ServiceName),
		Enabled: true,
		Groups:  []string{"ch_0"},
		Props: map[string]interface{}{
			"sup_units": []string{"%"},
		},
		Tags:             nil,
		PropSetReference: "",
		Interfaces:       sensorInterfaces,
	}

	deviceId := strings.ReplaceAll(device.Device.DeviceLabel, " ", "")
	manufacturer = "verisure"
	name = fmt.Sprintf("%s %s", device.Device.Gui.Label, device.Device.Area)
	serviceAddress := deviceId
	tempSensorService.Address = tempSensorService.Address + serviceAddress
	services = append(services, tempSensorService)

	if device.HumidityEnabled {
		humSensorService.Address = humSensorService.Address + serviceAddress
		services = append(services, humSensorService)
	}

	deviceAddr = deviceId
	powerSource := "battery"

	inclReport := fimptype.ThingInclusionReport{
		IntegrationId:     "",
		Address:           deviceAddr,
		Type:              "",
		ProductHash:       fmt.Sprintf("%s %s", device.Device.DeviceLabel, device.Device.Gui.Label),
		Alias:             fmt.Sprintf("%s %s", manufacturer, device.Device.Gui.Label),
		CommTechnology:    "",
		ProductName:       name,
		ManufacturerId:    manufacturer,
		DeviceId:          deviceId,
		HwVersion:         "1",
		SwVersion:         "1",
		PowerSource:       powerSource,
		WakeUpInterval:    "-1",
		Security:          "",
		Tags:              nil,
		Groups:            []string{"ch_0"},
		PropSets:          nil,
		TechSpecificProps: nil,
		Services:          services,
	}

	return inclReport
}

func (ns *NetworkService) SendSmartLockInclusionReport(device SmartLockDevice) fimptype.ThingInclusionReport {

	var name, manufacturer string
	var deviceAddr string
	services := []fimptype.Service{}

	sensorInterfaces := []fimptype.Interface{
		{
			Type:      "in",
			MsgType:   "cmd.lock.get_report",
			ValueType: "null",
			Version:   "1",
		},
		{
			Type:      "in",
			MsgType:   "cmd.lock.set",
			ValueType: "bool",
			Version:   "1",
		},
		{
			Type:      "out",
			MsgType:   "evt.lock.report",
			ValueType: "bool_map",
			Version:   "1",
		},
	}

	doorLockService := fimptype.Service{
		Name:    "door_lock",
		Alias:   "Door lock",
		Address: fmt.Sprintf("/rt:dev/rn:%s/ad:1/sv:door_lock/ad:", ServiceName),
		Enabled: true,
		Groups:  []string{"ch_0"},
		Props: map[string]interface{}{
			"sup_components": []string{"is_secured"},
		},
		Tags:             nil,
		PropSetReference: "",
		Interfaces:       sensorInterfaces,
	}

	deviceId := strings.ReplaceAll(device.Device.DeviceLabel, " ", "")
	manufacturer = "verisure"
	name = fmt.Sprintf("%s %s", device.Device.Gui.Label, device.Device.Area)
	serviceAddress := deviceId
	doorLockService.Address = doorLockService.Address + serviceAddress
	services = append(services, doorLockService)

	deviceAddr = deviceId
	powerSource := "battery"

	inclReport := fimptype.ThingInclusionReport{
		IntegrationId:     "",
		Address:           deviceAddr,
		Type:              "",
		ProductHash:       fmt.Sprintf("%s %s", device.Device.DeviceLabel, device.Device.Gui.Label),
		Alias:             fmt.Sprintf("%s %s", manufacturer, device.Device.Gui.Label),
		CommTechnology:    "",
		ProductName:       name,
		ManufacturerId:    manufacturer,
		DeviceId:          deviceId,
		HwVersion:         "1",
		SwVersion:         "1",
		PowerSource:       powerSource,
		WakeUpInterval:    "-1",
		Security:          "",
		Tags:              nil,
		Groups:            []string{"ch_0"},
		PropSets:          nil,
		TechSpecificProps: nil,
		Services:          services,
	}

	return inclReport
}

func (ns *NetworkService) SendDoorWindowInclusionReport(device DoorWindowDevice) fimptype.ThingInclusionReport {

	var name, manufacturer string
	var deviceAddr string
	services := []fimptype.Service{}

	sensorInterfaces := []fimptype.Interface{{
		Type:      "in",
		MsgType:   "cmd.open.get_report",
		ValueType: "null",
		Version:   "1",
	}, {
		Type:      "out",
		MsgType:   "evt.open.report",
		ValueType: "bool",
		Version:   "1",
	}}

	tempSensorService := fimptype.Service{
		Name:             "sensor_contact",
		Alias:            "Contact sensor",
		Address:          fmt.Sprintf("/rt:dev/rn:%s/ad:1/sv:sensor_contact/ad:", ServiceName),
		Enabled:          true,
		Groups:           []string{"ch_0"},
		Tags:             nil,
		PropSetReference: "",
		Interfaces:       sensorInterfaces,
	}

	deviceId := strings.ReplaceAll(device.Device.DeviceLabel, " ", "")
	manufacturer = "verisure"
	name = fmt.Sprintf("%s %s", device.Device.Gui.Label, device.Device.Area)
	serviceAddress := deviceId
	tempSensorService.Address = tempSensorService.Address + serviceAddress
	services = append(services, tempSensorService)

	deviceAddr = deviceId
	powerSource := "battery"

	inclReport := fimptype.ThingInclusionReport{
		IntegrationId:     "",
		Address:           deviceAddr,
		Type:              "",
		ProductHash:       fmt.Sprintf("%s %s", device.Device.DeviceLabel, device.Device.Gui.Label),
		Alias:             fmt.Sprintf("%s %s", manufacturer, device.Device.Gui.Label),
		CommTechnology:    "",
		ProductName:       name,
		ManufacturerId:    manufacturer,
		DeviceId:          deviceId,
		HwVersion:         "1",
		SwVersion:         "1",
		PowerSource:       powerSource,
		WakeUpInterval:    "-1",
		Security:          "",
		Tags:              nil,
		Groups:            []string{"ch_0"},
		PropSets:          nil,
		TechSpecificProps: nil,
		Services:          services,
	}

	return inclReport
}
