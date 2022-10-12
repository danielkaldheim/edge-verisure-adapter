package model

import (
	"github.com/futurehomeno/fimpgo/discovery"
)

func GetDiscoveryResource() discovery.Resource {
	return discovery.Resource{
		ResourceName:           ServiceName,
		ResourceFullName:       "Verisure",
		ResourceType:           discovery.ResourceTypeAd,
		Author:                 "daniel@kaldheim.org",
		Description:            "Connect your Verisure system to Future Home",
		IsInstanceConfigurable: false,
		InstanceId:             "1",
		Version:                "1.0.0",
		AdapterInfo: discovery.AdapterInfo{
			Technology:            "verisure",
			FwVersion:             "all",
			NetworkManagementType: "inclusion_exclusion",
		},
	}

}
