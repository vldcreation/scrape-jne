package main

type DataEntry struct {
	Origin      Location      `json:"origin"`
	Destination Location      `json:"destination"`
	Weight      int           `json:"weight"`
	Tariff      []TariffEntry `json:"tariff"`
}

type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
	TotalItems int `json:"total_items"`
}
