package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/check-tariff", checkTariffHandler)
	http.HandleFunc("/check-tariff/recursive", checkTariffRecursiveHandler)
	log.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
