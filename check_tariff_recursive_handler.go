package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
)

func checkTariffRecursiveHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	originQuery := query.Get("origin")
	destQuery := query.Get("destination")
	weightStr := query.Get("weight")
	pageStr := query.Get("page")
	perPageStr := query.Get("per_page")

	// Validate parameters
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

	// Get all origins and destinations
	origins, err := getLocations("origin", originQuery)
	if err != nil || len(origins) == 0 {
		NewResponse(w, http.StatusUnprocessableEntity, "There is no origin found").
			JSON()
		return
	}

	dests, err := getLocations("destination", destQuery)
	if err != nil || len(dests) == 0 {
		NewResponse(w, http.StatusUnprocessableEntity, "There is no destination found").
			JSON()
		return
	}

	// Generate all combinations
	var combinations []struct{ Origin, Destination Location }
	for _, o := range origins {
		for _, d := range dests {
			combinations = append(combinations, struct{ Origin, Destination Location }{o, d})
		}
	}

	// Pagination logic
	totalItems := len(combinations)
	page := 1
	if pageStr != "" {
		page, _ = strconv.Atoi(pageStr)
		if page < 1 {
			page = 1
		}
	}

	perPage := 10
	if perPageStr != "" {
		perPage, _ = strconv.Atoi(perPageStr)
		if perPage < 0 {
			perPage = 10
		}
	}

	if perPage == 0 {
		perPage = totalItems
	}

	totalPages := totalItems / perPage
	if totalItems%perPage != 0 {
		totalPages++
	}

	start := (page - 1) * perPage
	if start >= totalItems {
		start = totalItems
	}
	end := start + perPage
	if end > totalItems {
		end = totalItems
	}

	currentCombos := combinations[start:end]

	// Collect results with concurrency
	type result struct {
		data DataEntry
		err  error
	}
	results := make(chan result, len(currentCombos))

	for _, combo := range currentCombos {
		go func(c struct{ Origin, Destination Location }) {
			url := fmt.Sprintf(
				"https://jne.co.id/shipping-fee?origin=%s&destination=%s&weight=%d",
				c.Origin.Code, c.Destination.Code, weight,
			)

			originLabel, destLabel, tariff, err := scrapeShippingFee(url)
			if err != nil {
				results <- result{err: err}
				return
			}

			results <- result{data: DataEntry{
				Origin:      Location{Code: c.Origin.Code, Label: originLabel},
				Destination: Location{Code: c.Destination.Code, Label: destLabel},
				Weight:      weight,
				Tariff:      tariff,
			}}
		}(combo)
	}

	// Collect results
	var data []DataEntry
	for i := 0; i < len(currentCombos); i++ {
		res := <-results
		if res.err != nil {
			log.Printf("Scraping error: %v", res.err)
			continue
		}
		data = append(data, res.data)
	}

	NewResponse(w, http.StatusOK, "Success").
		WithData(data).WithMeta(Pagination{
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		TotalItems: totalItems,
	}).JSON()
	return
}
