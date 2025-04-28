package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
)

// Add new structs for recursive response
type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
	TotalItems int `json:"total_items"`
}

type RecursiveResponse struct {
	Message    string      `json:"message"`
	Data       []DataEntry `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

type DataEntry struct {
	Origin      OriginDestination `json:"origin"`
	Destination OriginDestination `json:"destination"`
	Weight      int               `json:"weight"`
	Tariff      []TariffEntry     `json:"tariff"`
}

// Structs for API responses
type LocationResponse struct {
	Status bool `json:"status"`
	Data   []struct {
		Code  string `json:"code"`
		Label string `json:"label"`
	} `json:"data"`
}

// Structs for the final response
type OriginDestination struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type Info struct {
	Origin      OriginDestination `json:"origin"`
	Destination OriginDestination `json:"destination"`
	Weight      int               `json:"weight"`
}

type TariffEntry struct {
	ServiceName  string `json:"service_name"`
	ShipmentType string `json:"shipment_type"`
	Fee          string `json:"fee"`
	ETD          string `json:"etd"`
}

type Response struct {
	Message string        `json:"message"`
	Info    Info          `json:"info"`
	Tariff  []TariffEntry `json:"tariff"`
}

// Modified location struct
type Location struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

func checkTariffHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	originQuery := query.Get("origin")
	destQuery := query.Get("destination")
	weightStr := query.Get("weight")

	if originQuery == "" || destQuery == "" || weightStr == "" {
		respondWithError(w, "Missing parameters", http.StatusUnprocessableEntity)
		return
	}

	weight, err := strconv.Atoi(weightStr)
	if err != nil || weight <= 0 {
		respondWithError(w, "Invalid weight", http.StatusUnprocessableEntity)
		return
	}

	// Get origin code
	originCode, err := getLocationCode("origin", originQuery)
	if err != nil {
		respondWithError(w, "Error fetching origin code", http.StatusInternalServerError)
		return
	}
	if originCode == "" {
		respondWithError(w, "There is no origin found", http.StatusUnprocessableEntity)
		return
	}

	// Get destination code
	destCode, err := getLocationCode("destination", destQuery)
	if err != nil {
		respondWithError(w, "Error fetching destination code", http.StatusInternalServerError)
		return
	}
	if destCode == "" {
		respondWithError(w, "There is no destination found", http.StatusUnprocessableEntity)
		return
	}

	// Scrape shipping fee page
	url := fmt.Sprintf("https://jne.co.id/shipping-fee?origin=%s&destination=%s&weight=%d", originCode, destCode, weight)
	originLabel, destLabel, tariff, err := scrapeShippingFee(url)
	if err != nil {
		respondWithError(w, "Error scraping data", http.StatusInternalServerError)
		return
	}

	if originLabel == "" || destLabel == "" {
		respondWithError(w, "Origin or destination not found on page", http.StatusUnprocessableEntity)
		return
	}

	response := Response{
		Message: "ok",
		Info: Info{
			Origin:      OriginDestination{Code: originCode, Label: originLabel},
			Destination: OriginDestination{Code: destCode, Label: destLabel},
			Weight:      weight,
		},
		Tariff: tariff,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func checkTariffRecursiveHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	originQuery := query.Get("origin")
	destQuery := query.Get("destination")
	weightStr := query.Get("weight")
	pageStr := query.Get("page")
	perPageStr := query.Get("per_page")

	// Validate parameters
	if originQuery == "" || destQuery == "" || weightStr == "" {
		respondWithError(w, "Missing parameters", http.StatusUnprocessableEntity)
		return
	}

	weight, err := strconv.Atoi(weightStr)
	if err != nil || weight <= 0 {
		respondWithError(w, "Invalid weight", http.StatusUnprocessableEntity)
		return
	}

	// Get all origins and destinations
	origins, err := getLocations("origin", originQuery)
	if err != nil || len(origins) == 0 {
		respondWithError(w, "There is no origin found", http.StatusUnprocessableEntity)
		return
	}

	dests, err := getLocations("destination", destQuery)
	if err != nil || len(dests) == 0 {
		respondWithError(w, "There is no destination found", http.StatusUnprocessableEntity)
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
				Origin:      OriginDestination{Code: c.Origin.Code, Label: originLabel},
				Destination: OriginDestination{Code: c.Destination.Code, Label: destLabel},
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

	response := RecursiveResponse{
		Message: "ok",
		Data:    data,
		Pagination: Pagination{
			Page:       page,
			PerPage:    perPage,
			TotalPages: totalPages,
			TotalItems: totalItems,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

	var locResp struct {
		Status bool       `json:"status"`
		Data   []Location `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&locResp); err != nil {
		return nil, err
	}

	if !locResp.Status {
		return nil, nil
	}

	return locResp.Data, nil
}

func scrapeShippingFee(url string) (originLabel, destLabel string, tariff []TariffEntry, err error) {
	c := colly.NewCollector()

	c.OnHTML("div.box-tujuan", func(e *colly.HTMLElement) {
		e.ForEach("div.box-tujuan__list", func(_ int, el *colly.HTMLElement) {
			h6 := strings.TrimSpace(el.ChildText("h6"))
			p := strings.TrimSpace(el.ChildText("p"))
			switch h6 {
			case "Dari":
				originLabel = p
			case "Tujuan":
				destLabel = p
			}
		})
	})

	c.OnHTML("div.wrap-table table tbody tr", func(e *colly.HTMLElement) {
		service := strings.TrimSpace(e.ChildText("td:nth-child(1)"))
		shipmentType := strings.TrimSpace(e.ChildText("td:nth-child(2)"))
		fee := strings.TrimSpace(e.ChildText("td:nth-child(3)"))
		etd := strings.TrimSpace(e.ChildText("td:nth-child(4)"))

		tariff = append(tariff, TariffEntry{
			ServiceName:  service,
			ShipmentType: shipmentType,
			Fee:          fee,
			ETD:          etd,
		})
	})

	c.OnError(func(r *colly.Response, e error) {
		err = e
	})

	err = c.Visit(url)
	return
}

func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}

func main() {
	http.HandleFunc("/check-tariff", checkTariffHandler)
	http.HandleFunc("/check-tariff/recursive", checkTariffRecursiveHandler)
	log.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
