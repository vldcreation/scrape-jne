package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Location Model
type Location struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// Structs for API responses
type LocationResponse struct {
	Status bool       `json:"status"`
	Data   []Location `json:"data"`
}

func getLocationCode(locationType, query string) (string, error) {
	url := ""
	if locationType == "origin" {
		url = fmt.Sprintf("https://jne.co.id/api-origin?search=%s", query)
	} else {
		url = fmt.Sprintf("https://jne.co.id/api-destination?search=%s", query)
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var locResp LocationResponse
	if err := json.NewDecoder(resp.Body).Decode(&locResp); err != nil {
		return "", err
	}

	if !locResp.Status || len(locResp.Data) == 0 {
		return "", nil
	}

	return locResp.Data[0].Code, nil
}

func getLocations(locationType, query string) ([]Location, error) {
	url := ""
	switch locationType {
	case "origin":
		url = fmt.Sprintf("https://jne.co.id/api-origin?search=%s", query)
	case "destination":
		url = fmt.Sprintf("https://jne.co.id/api-destination?search=%s", query)
	default:
		return nil, fmt.Errorf("invalid location type")
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var locResp LocationResponse

	if err := json.NewDecoder(resp.Body).Decode(&locResp); err != nil {
		return nil, err
	}

	if !locResp.Status {
		return nil, nil
	}

	return locResp.Data, nil
}
