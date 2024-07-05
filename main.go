package main

import (
	"os"
	"fmt"
	"time"
	"bytes"
	"strings"
	"strconv"
    "net/http"
	"io/ioutil"
	"encoding/json"
)

type DNSRecord struct {
	Zone             string     `json:"zone"`
	Domain           string     `json:"domain"`
	Type             string     `json:"type"`
	UseClientSubnet  bool       `json:"use_client_subnet"`
	Answers          []Answer   `json:"answers"`
}
type Answer struct {
	Answer []string `json:"answer"`
}

type ZoneInfo struct {
	Name             string     `json:"name"`
	Records          []Record     `json:"records"`
}
type Record struct { // Don't really need any of this data
	
}

type APIResponse struct {
	PublicIP         string    `json:"public_ip"`
	ZoneName         string    `json:"zone_name"`
	RecordCount      int       `json:"record_count"`
}

const (
	recordType      = "A"
	baseURL         = "https://api.nsone.net/v1/zones/"
	defaultInterval = 10
)

func main() {
	domains := os.Getenv("NS1_DOMAINS")
	zone := os.Getenv("NS1_ZONE")
	key := os.Getenv("NS1_KEY")
	interval := os.Getenv("NS1_INTERVAL")
	disable_api := os.Getenv("NS1_API_DISABLE")

	if zone == "" || key == "" {
		fmt.Println("Missing required environment variables NS1_KEY and NS1_ZONE.")
		return
	}
	if domains == "" {
		fmt.Printf("Missing NS1_DOMAINS, setting it to NS1_ZONE (%s).\n", zone)
		domains = zone
	}
	intervalMinutes, err := strconv.Atoi(interval)
	if err != nil {
		fmt.Printf("Missing or invalid NS1_INTERVAL, setting it to %d (default).\n", defaultInterval)
		intervalMinutes = defaultInterval
	}
	
	client := &http.Client{}

	if disable_api != "true" { // TODO: add better api sometime
		go func() {
			http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					req, err := http.NewRequest("GET", baseURL + r.URL.Query().Get("zone"), nil)
					if err != nil {
						fmt.Printf("Error during api call : %v\n", err)
						return
					}
					req.Header.Set("X-NSONE-Key", key)
					req.Header.Set("Content-Type", "application/json")
			
					resp, err := client.Do(req)
					if err != nil {
						fmt.Printf("Error during api call : %v\n", err)
						return
					}
					defer resp.Body.Close()

					body, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						fmt.Printf("API: Error reading response body: %v\n", err)
						http.Error(w, "API: Failed to read response body", http.StatusInternalServerError)
						return
					}
				
					var zoneInfo ZoneInfo
					err = json.Unmarshal(body, &zoneInfo)
					if err != nil {
						fmt.Printf("API: Error unmarshaling response body: %v\n", err)
						http.Error(w, "API: Failed to parse JSON response", http.StatusInternalServerError)
						return
					}

					var response APIResponse
					response.ZoneName = zoneInfo.Name
					response.RecordCount = len(zoneInfo.Records)
					response.PublicIP = func() string {
						if value := os.Getenv("NS1_PUBIP"); value != "" {
							return value
						}
						return getPubIP()
					}()

					w.Header().Set("Content-Type", "application/json")
					err = json.NewEncoder(w).Encode(response)
					if err != nil {
						fmt.Printf("API: Error encoding response to JSON: %v\n", err)
						http.Error(w, "API: Failed to encode response to JSON", http.StatusInternalServerError)
						return
					}
				}
			})
			http.ListenAndServe(":80", nil)
		}()
	}

	UpdateDNS(domains, zone, key, client)
	ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			UpdateDNS(domains, zone, key, client)
		}
	}
}

func UpdateDNS(domains string, zone string, key string, client *http.Client) {
	pubIP := getPubIP()
	if pubIP == os.Getenv("NS1_PUBIP") {
		fmt.Printf("Records already up to date with IPv4 %s.\n", pubIP)
		return
	}
	fmt.Printf("Updating all records with IPv4 %s\n", pubIP)
	for _, domain := range strings.Split(domains, ",") {
		url := fmt.Sprintf("%s%s/%s/%s", baseURL, zone, domain, recordType)
		record := DNSRecord{
			Zone:            zone,
			Domain:          domain,
			Type:            recordType,
			UseClientSubnet: true,
			Answers: []Answer{
				{Answer: []string{pubIP}},
			},
		}

		recordJSON, err := json.Marshal(record)
		if err != nil {
			fmt.Printf("Error marshalling JSON for domain %s: %v\n", domain, err)
			continue
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(recordJSON))
		if err != nil {
			fmt.Printf("Error creating HTTP request for domain %s: %v\n", domain, err)
			continue
		}
		req.Header.Set("X-NSONE-Key", key)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error updating domain %s: %v\n", domain, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Printf("Record %s updated successfully.\n", domain)
		} else {
			fmt.Printf("Failed to update record %s. Status code: %d\n", domain, resp.StatusCode)
		}
	}
	os.Setenv("NS1_PUBIP", pubIP)
}

func getPubIP() string {
	resp, err := http.Get("https://ipinfo.io/ip")
	if err != nil {
		fmt.Printf("Failed to get public IPv4: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read the response body: %v\n", err)
		return ""
	}

	ip := string(body)
	return strings.TrimSpace(ip)
}