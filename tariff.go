package main

type Info struct {
	Origin      Location `json:"origin"`
	Destination Location `json:"destination"`
	Weight      int      `json:"weight"`
}

type TariffEntry struct {
	ServiceName  string `json:"service_name"`
	ShipmentType string `json:"shipment_type"`
	Fee          string `json:"fee"`
	ETD          string `json:"etd"`
}

type TariffResponse struct {
	Info   Info          `json:"info"`
	Tariff []TariffEntry `json:"tariff"`
}
