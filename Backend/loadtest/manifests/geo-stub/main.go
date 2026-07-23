// Stub for BigDataCloud reverse-geocode-client used on loadtest stand.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
)

type response struct {
	CountryName          string `json:"countryName"`
	CountryCode          string `json:"countryCode"`
	PrincipalSubdivision string `json:"principalSubdivision"`
	City                 string `json:"city"`
	Locality             string `json:"locality"`
	Postcode             string `json:"postcode"`
	Continent            string `json:"continent"`
	ContinentCode        string `json:"continentCode"`
}

func handle(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	lat, _ := strconv.ParseFloat(q.Get("latitude"), 64)
	lon, _ := strconv.ParseFloat(q.Get("longitude"), 64)
	cell := fmt.Sprintf("%d_%d", int(math.Floor(lat)), int(math.Floor(lon)))
	resp := response{
		CountryName:          "TestCountry",
		CountryCode:          "TC",
		PrincipalSubdivision: "TestRegion",
		City:                 "TestCity_" + cell,
		Locality:             "TestLocality_" + cell,
		Continent:            "TestContinent",
		ContinentCode:        "TC",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func main() {
	http.HandleFunc("/", handle)
	log.Println("geo-stub listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
