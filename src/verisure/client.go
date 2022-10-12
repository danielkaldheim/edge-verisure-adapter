package verisure

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/verisure/model"
)

type Client struct {
	states *model.States
	giid   string
}

var (
	baseURLS = []string{"https://m-api01.verisure.com",
		"https://m-api02.verisure.com"}
	applicationID = "DK_FUTUREHOME"
)

func NewClient(states *model.States) (*Client, error) {
	c := Client{states: states}
	if states.GIID != "" {
		c.giid = states.GIID
	}

	return &c, nil
}

func (c *Client) request(method string, path string, requestBody []byte) ([]byte, error) {
	path = strings.TrimLeft(path, "/")

	var URLS []string

	URLS = append(URLS, baseURLS...)

	for _, baseURL := range URLS {
		url := fmt.Sprintf("%s/%s", baseURL, path)
		log.Debugf("%s - %s", method, url)

		req, err := http.NewRequest(method, url, bytes.NewReader(requestBody))
		if err != nil {
			return nil, err
		}

		if method == http.MethodPost {
			req.Header.Add("Content-Type", "application/json")
		}

		req.Header.Add("APPLICATION_ID", applicationID)
		for _, cookie := range c.states.Cookies {
			req.AddCookie(cookie)
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		if res.StatusCode < http.StatusOK && res.StatusCode >= http.StatusMultipleChoices {
			return body, fmt.Errorf("got %d from %s", res.StatusCode, url)
		}

		if res.StatusCode == http.StatusOK {
			if strings.Contains(string(body), "SYS_00004") {
				for i, j := 0, len(baseURLS)-1; i < j; i, j = i+1, j-1 {
					baseURLS[i], baseURLS[j] = baseURLS[j], baseURLS[i]
				}
				continue
			}

			cookieUpdated := false
			for i, cookie := range res.Cookies() {
				if cookie.Name == "vs-access" {
					for _, oldCookie := range c.states.Cookies {
						if oldCookie.Name == "vs-access" && oldCookie.RawExpires != cookie.RawExpires {
							c.states.Cookies[i] = cookie
							cookieUpdated = true
						}
					}
				}
			}

			if cookieUpdated {
				c.states.SaveToFile()
			}
		}

		return body, err
	}
	return nil, errors.New("failed to request")
}

func (c *Client) Login(username string, password string) error {
	accessCookie := c.states.GetCookieByName("vs-access")
	if accessCookie != nil {
		err := c.UpdateToken()
		if err == nil {
			return nil
		}
	}

	log.Debug("Do a sign in")

	url := fmt.Sprintf("%s/%s", baseURLS[0], "auth/login")

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("APPLICATION_ID", applicationID)

	b := fmt.Sprintf("%s:%s", username, password)
	se := base64.StdEncoding.EncodeToString([]byte(b))

	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", se))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode == http.StatusOK {
		c.states.Cookies = res.Cookies()
		c.states.Username = username
		c.states.SaveToFile()
		return nil
	}

	return errors.New("failed to login")
}

func (c *Client) UpdateToken() error {
	now := time.Now()

	accessCookie := c.states.GetCookieByName("vs-access")
	if accessCookie != nil && now.Before(accessCookie.Expires) {
		log.Debug("Access cookie is goood")
		return nil
	}

	refreshCookie := c.states.GetCookieByName("vs-refresh")
	if refreshCookie == nil {
		return errors.New("no refresh cookie found")
	}

	if now.After(refreshCookie.Expires) {
		return errors.New("refresh cookie expired")
	}

	log.Debug("Refresh please")
	_, err := c.request(http.MethodGet, "/auth/token", nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) FetchAllInstallations() ([]model.Installation, error) {

	if c.states.Username == "" {
		return nil, errors.New("must set installation to get installations")
	}

	q := GraphQLQuery{
		OperationName: "fetchAllInstallations",
		Variables:     map[string]interface{}{"email": c.states.Username},
		Query:         "query fetchAllInstallations($email: String!){\n  account(email: $email) {\n    installations {\n      giid\n      alias\n      customerType\n      dealerId\n      subsidiary\n      pinCodeLength\n      locale\n      address {\n        street\n        city\n        postalNumber\n        __typename\n      }\n      __typename\n    }\n    __typename\n  }\n}\n",
	}

	payload, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}

	body, err := c.request(http.MethodPost, "/graphql", payload)
	if err != nil {
		return nil, err
	}

	response := &GraphQLResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Data != nil && response.Data.Account.Installations != nil {
		return response.Data.Account.Installations, nil
	}

	return nil, errors.New("failed to fetch installations")
}

func (c *Client) FetchInstallationState() (*model.Installation, error) {
	if c.giid == "" {
		return nil, errors.New("must set installation to get climate")
	}

	q := GraphQLQuery{
		OperationName: "GetState",
		Variables:     map[string]interface{}{"giid": c.giid},
		Query:         "query GetState($giid: String!) {\n  installation(giid: $giid) {\n    doorWindows {\n      device {\n        deviceLabel\n      }\n      state\n      reportTime\n    }\n    climates {\n      device {\n        deviceLabel\n      }\n      humidityEnabled\n      humidityTimestamp\n      humidityValue\n      temperatureTimestamp\n      temperatureValue\n    }\n    smartLocks {\n      device {\n        deviceLabel\n      }\n      lockStatus\n      doorState\n      lockMethod\n      eventTime\n      doorLockType\n      secureMode\n      user {\n        name\n      }\n    }\n    armState {\n      type\n      statusType\n      date\n      name\n      changedVia\n    }\n    smartplugs {\n      device {\n        deviceLabel\n      }\n      currentState\n      icon\n      isHazardous\n    }\n  }\n}\n",
	}

	payload, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}

	body, err := c.request(http.MethodPost, "/graphql", payload)
	if err != nil {
		return nil, err
	}

	response := &GraphQLResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Data != nil && response.Data.Installation != nil {
		return response.Data.Installation, nil
	}

	return nil, errors.New("failed to fetch climates")
}

func (c *Client) FetchClimate() ([]model.ClimateDevice, error) {
	if c.giid == "" {
		return nil, errors.New("must set installation to get climate")
	}

	q := GraphQLQuery{
		OperationName: "Climate",
		Variables:     map[string]interface{}{"giid": c.giid},
		Query:         "query Climate($giid: String!) {\n  installation(giid: $giid) {\n    climates {\n      device {\n        deviceLabel\n        area\n        gui {\n          label\n          __typename\n        }\n        __typename\n      }\n      humidityEnabled\n      humidityTimestamp\n      humidityValue\n      temperatureTimestamp\n      temperatureValue\n      thresholds {\n        aboveMaxAlert\n        belowMinAlert\n        sensorType\n        __typename\n      }\n      __typename\n    }\n    __typename\n  }\n}\n",
	}

	payload, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}

	body, err := c.request(http.MethodPost, "/graphql", payload)
	if err != nil {
		return nil, err
	}

	response := &GraphQLResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Data != nil && response.Data.Installation.Climates != nil {
		return response.Data.Installation.Climates, nil
	}

	return nil, errors.New("failed to fetch climates")
}

func (c *Client) FetchDoorWindow() ([]model.DoorWindowDevice, error) {
	if c.giid == "" {
		return nil, errors.New("must set installation to get door and windows")
	}

	q := GraphQLQuery{
		OperationName: "DoorWindow",
		Variables:     map[string]interface{}{"giid": c.giid},
		Query:         "query DoorWindow($giid: String!) {\n  installation(giid: $giid) {\n    doorWindows {\n      device {\n        deviceLabel\n        area\n        gui {\n          label\n          __typename\n        }\n        __typename\n      }\n      type\n      area\n      state\n      wired\n      reportTime\n      __typename\n    }\n    __typename\n  }\n}\n",
	}

	payload, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}

	body, err := c.request(http.MethodPost, "/graphql", payload)
	if err != nil {
		return nil, err
	}

	response := &GraphQLResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Data != nil && response.Data.Installation.DoorWindows != nil {
		return response.Data.Installation.DoorWindows, nil
	}

	return nil, errors.New("failed to fetch door and windows")
}

func (c *Client) LockSmartLock(deviceLabel string, code string) error {
	if c.giid == "" {
		return errors.New("must set installation to lock smart locks")
	}

	q := GraphQLQuery{
		OperationName: "DoorLock",
		Variables: map[string]interface{}{
			"giid":        c.giid,
			"deviceLabel": deviceLabel,
			"input": map[string]interface{}{
				"code": code,
			},
		},
		Query: "mutation DoorLock(\n  $giid: String!\n  $deviceLabel: String!\n  $input: LockDoorInput!\n) {\n  DoorLock(giid: $giid, deviceLabel: $deviceLabel, input: $input)\n}\n",
	}

	payload, err := json.Marshal(q)
	if err != nil {
		return err
	}

	body, err := c.request(http.MethodPost, "/graphql", payload)
	if err != nil {
		return err
	}

	response := &GraphQLResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	if response.Errors != nil {
		return errors.New(response.Errors[0].Message)
	}

	return nil
}

func (c *Client) UnlockSmartLock(deviceLabel string, code string) error {
	if c.giid == "" {
		return errors.New("must set installation to lock smart locks")
	}

	q := GraphQLQuery{
		OperationName: "DoorUnlock",
		Variables: map[string]interface{}{
			"giid":        c.giid,
			"deviceLabel": deviceLabel,
			"input": map[string]interface{}{
				"code": code,
			},
		},
		Query: "mutation DoorUnlock(\n  $giid: String!\n  $deviceLabel: String!\n  $input: LockDoorInput!\n) {\n  DoorUnlock(giid: $giid, deviceLabel: $deviceLabel, input: $input)\n}\n",
	}

	payload, err := json.Marshal(q)
	if err != nil {
		return err
	}

	body, err := c.request(http.MethodPost, "/graphql", payload)
	if err != nil {
		return err
	}

	response := &GraphQLResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	if response.Errors != nil {
		return errors.New(response.Errors[0].Message)
	}

	return nil
}

func (c *Client) FetchSmartLock() ([]model.SmartLockDevice, error) {
	if c.giid == "" {
		return nil, errors.New("must set installation to get smart locks")
	}

	q := GraphQLQuery{
		OperationName: "SmartLock",
		Variables:     map[string]interface{}{"giid": c.giid},
		Query:         "query SmartLock($giid: String!) {\n  installation(giid: $giid) {\n    smartLocks {\n      device {\n        deviceLabel\n        area\n        gui {\n          label\n          __typename\n        }\n        __typename\n      }\n lockStatus\n      doorState\n      lockMethod\n      eventTime\n      doorLockType\n      secureMode\n      user {\n        name\n        __typename\n      }\n      __typename\n    }\n    __typename\n  }\n}\n",
	}

	payload, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}

	body, err := c.request(http.MethodPost, "/graphql", payload)
	if err != nil {
		return nil, err
	}

	response := &GraphQLResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Data != nil && response.Data.Installation.SmartLocks != nil {
		return response.Data.Installation.SmartLocks, nil
	}

	return nil, errors.New("failed to fetch smart locks")
}

func (c *Client) FetchUserTracking() ([]model.UserTracking, error) {
	if c.giid == "" {
		return nil, errors.New("must set installation to get user tracking")
	}

	q := GraphQLQuery{
		OperationName: "userTrackings",
		Variables:     map[string]interface{}{"giid": c.giid},
		Query:         "query userTrackings($giid: String!) {\n  installation(giid: $giid) {\n    userTrackings {\n      isCallingUser\n      webAccount\n      status\n      xbnContactId\n      currentLocationName\n      deviceId\n      name\n      initials\n      currentLocationTimestamp\n      deviceName\n      currentLocationId\n      __typename\n    }\n    __typename\n  }\n}\n",
	}

	payload, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}

	body, err := c.request(http.MethodPost, "/graphql", payload)
	if err != nil {
		return nil, err
	}

	response := &GraphQLResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Data != nil && response.Data.Installation.UserTrackings != nil {
		return response.Data.Installation.UserTrackings, nil
	}

	return nil, errors.New("failed to fetch user tracking")
}

func (c *Client) FetchArmState() (*model.ArmState, error) {
	if c.giid == "" {
		return nil, errors.New("must set installation to get arm state")
	}

	q := GraphQLQuery{
		OperationName: "ArmState",
		Variables:     map[string]interface{}{"giid": c.giid},
		Query:         "query ArmState($giid: String!) {\n  installation(giid: $giid) {\n    armState {\n      type\n      statusType\n      date\n      name\n      changedVia\n      __typename\n    }\n    __typename\n  }\n}\n",
	}

	payload, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}

	body, err := c.request(http.MethodPost, "/graphql", payload)
	if err != nil {
		return nil, err
	}

	response := &GraphQLResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Data != nil && response.Data.Installation.ArmState != nil {
		return response.Data.Installation.ArmState, nil
	}

	return nil, errors.New("failed to fetch arm state")
}

func (c *Client) SetGIID(giid string) error {
	c.giid = giid
	c.states.GIID = giid

	c.states.SaveToFile()

	return nil
}
