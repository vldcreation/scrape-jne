package main

import (
	"fmt"
	"net/http"
	"strconv"
)

func checkTariffHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	originQuery := query.Get("origin")
	destQuery := query.Get("destination")
	weightStr := query.Get("weight")

	if originQuery == "" || destQuery == "" || weightStr == "" {
		NewResponse(w, http.StatusUnprocessableEntity, "Missing parameters").
			JSON()
		return
	}

	weight, err := strconv.Atoi(weightStr)
	if err != nil || weight <= 0 {
		NewResponse(w, http.StatusUnprocessableEntity, "Invalid weight").
			JSON()
		return
	}

	// Get origin code
	originCode, err := getLocationCode("origin", originQuery)
	if err != nil {
		NewResponse(w, http.StatusUnprocessableEntity, "Invalid origin").
			JSON()
		return
	}
	if originCode == "" {
		NewResponse(w, http.StatusUnprocessableEntity, "There is no origin found").
			JSON()
		return
	}

	// Get destination code
	destCode, err := getLocationCode("destination", destQuery)
	if err != nil {
		NewResponse(w, http.StatusInternalServerError, "Error fetching destination code").
			JSON()
		return
	}
	if destCode == "" {
		NewResponse(w, http.StatusUnprocessableEntity, "There is no destination found").
			JSON()
		return
	}

	// Scrape shipping fee page
	url := fmt.Sprintf("https://jne.co.id/shipping-fee?origin=%s&destination=%s&weight=%d", originCode, destCode, weight)
	originLabel, destLabel, tariff, err := scrapeShippingFee(url)
	if err != nil {
		NewResponse(w, http.StatusInternalServerError, "Error scraping data").
			JSON()
		return
	}

	if originLabel == "" || destLabel == "" {
		NewResponse(w, http.StatusUnprocessableEntity, "Origin or destination not found on page").
			JSON()
		return
	}

	NewResponse(w, http.StatusOK, "OK").
		WithData(TariffResponse{
			Info: Info{
				Origin:      Location{Code: originCode, Label: originLabel},
				Destination: Location{Code: destCode, Label: destLabel},
				Weight:      weight,
			},
			Tariff: tariff,
		}).
		JSON()
	return
}
