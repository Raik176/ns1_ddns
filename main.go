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
		fmt.Printf("Records already up to date with IPv4 %s.", pubIP)
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