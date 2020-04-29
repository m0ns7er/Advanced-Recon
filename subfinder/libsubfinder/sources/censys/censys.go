//
// Written By : @ice3man (Nizamul Rana)
//
// Distributed Under MIT License
// Copyrights (C) 2018 Ice3man
//

// A golang client for Censys Subdomain Discovery
package censys

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/Ice3man543/subfinder/libsubfinder/helper"
)

type resultsq struct {
	Data  []string `json:"parsed.extensions.subject_alt_name.dns_names"`
	Data1 []string `json:"parsed.names"`
}

type response struct {
	Results  []resultsq `json:"results"`
	Metadata struct {
		Pages int `json:"pages"`
	} `json:"metadata"`
}

// all subdomains found
var subdomains []string

// Query function returns all subdomains found using the service.
func Query(domain string, state *helper.State, ch chan helper.Result) {

	var uniqueSubdomains []string
	var initialSubdomains []string
	var hostResponse response
	var result helper.Result

	// Default Censys Pages to process. I think 10 is a good value
	//DefaultCensysPages := 10

	// We have recieved an API Key
	if state.ConfigState.CensysUsername != "" && state.ConfigState.CensysSecret != "" {

		// Get credentials for performing HTTP Basic Auth
		username := state.ConfigState.CensysUsername
		key := state.ConfigState.CensysSecret

		if state.CurrentSettings.CensysPages != "all" {

			CensysPages, _ := strconv.Atoi(state.CurrentSettings.CensysPages)

			for i := 1; i <= CensysPages; i++ {
				// Create JSON Get body
				var request = []byte(`{"query":"` + domain + `", "page":` + strconv.Itoa(i) + `, "fields":["parsed.names","parsed.extensions.subject_alt_name.dns_names"], "flatten":true}`)

				client := &http.Client{}
				req, err := http.NewRequest("POST", "https://www.censys.io/api/v1/search/certificates", bytes.NewBuffer(request))
				req.SetBasicAuth(username, key)

				// Set content type as application/json
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Accept", "application/json")

				resp, err := client.Do(req)
				if err != nil {
					result.Subdomains = subdomains
					result.Error = err
					ch <- result
					return
				}

				// Get the response body
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					result.Subdomains = subdomains
					result.Error = err
					ch <- result
					return
				}

				err = json.Unmarshal([]byte(body), &hostResponse)
				if err != nil {
					result.Subdomains = subdomains
					result.Error = err
					ch <- result
					return
				}

				// Add all items found
				for _, res := range hostResponse.Results {
					for _, host := range res.Data {
						initialSubdomains = append(initialSubdomains, host)
					}
					for _, host := range res.Data1 {
						initialSubdomains = append(initialSubdomains, host)
					}
				}

				validSubdomains := helper.Validate(domain, initialSubdomains)
				uniqueSubdomains = helper.Unique(validSubdomains)
			}

			// Append each subdomain found to subdomains array
			for _, subdomain := range uniqueSubdomains {

				if strings.Contains(subdomain, "*.") {
					subdomain = strings.Split(subdomain, "*.")[1]
				}

				if state.Verbose == true {
					if state.Color == true {
						fmt.Printf("\n[%sCENSYS%s] %s", helper.Red, helper.Reset, subdomain)
					} else {
						fmt.Printf("\n[CENSYS] %s", subdomain)
					}
				}

				subdomains = append(subdomains, subdomain)
			}
		} else if state.CurrentSettings.CensysPages == "all" {

			// Create JSON Get body
			var request = []byte(`{"query":"` + domain + `", "page":1, "fields":["parsed.names","parsed.extensions.subject_alt_name.dns_names"], "flatten":true}`)

			client := &http.Client{}
			req, err := http.NewRequest("POST", "https://www.censys.io/api/v1/search/certificates", bytes.NewBuffer(request))
			req.SetBasicAuth(username, key)

			// Set content type as application/json
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				result.Subdomains = subdomains
				result.Error = err
				ch <- result
				return
			}

			// Get the response body
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				result.Subdomains = subdomains
				result.Error = err
				ch <- result
				return
			}

			err = json.Unmarshal([]byte(body), &hostResponse)
			if err != nil {
				result.Subdomains = subdomains
				result.Error = err
				ch <- result
				return
			}

			// Add all items found
			for _, res := range hostResponse.Results {
				for _, host := range res.Data {
					initialSubdomains = append(initialSubdomains, host)
				}
				for _, host := range res.Data1 {
					initialSubdomains = append(initialSubdomains, host)
				}
			}

			validSubdomains := helper.Validate(domain, initialSubdomains)
			uniqueSubdomains = helper.Unique(validSubdomains)

			// Append each subdomain found to subdomains array
			for _, subdomain := range uniqueSubdomains {

				if strings.Contains(subdomain, "*.") {
					subdomain = strings.Split(subdomain, "*.")[1]
				}

				if state.Verbose == true {
					if state.Color == true {
						fmt.Printf("\n[%sCENSYS%s] %s", helper.Red, helper.Reset, subdomain)
					} else {
						fmt.Printf("\n[CENSYS] %s", subdomain)
					}
				}

				subdomains = append(subdomains, subdomain)
			}

			for i := 2; i <= hostResponse.Metadata.Pages; i++ {
				// Create JSON Get body
				var request = []byte(`{"query":"` + domain + `", "page":` + strconv.Itoa(i) + `, "fields":["parsed.names","parsed.extensions.subject_alt_name.dns_names"], "flatten":true}`)

				client := &http.Client{}
				req, err := http.NewRequest("POST", "https://www.censys.io/api/v1/search/certificates", bytes.NewBuffer(request))
				req.SetBasicAuth(username, key)

				// Set content type as application/json
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Accept", "application/json")

				resp, err := client.Do(req)
				if err != nil {
					result.Subdomains = subdomains
					result.Error = err
					ch <- result
					return
				}

				// Get the response body
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					result.Subdomains = subdomains
					result.Error = err
					ch <- result
					return
				}

				err = json.Unmarshal([]byte(body), &hostResponse)
				if err != nil {
					result.Subdomains = subdomains
					result.Error = err
					ch <- result
					return
				}

				// Add all items found
				for _, res := range hostResponse.Results {
					for _, host := range res.Data {
						initialSubdomains = append(initialSubdomains, host)
					}
					for _, host := range res.Data1 {
						initialSubdomains = append(initialSubdomains, host)
					}
				}

				validSubdomains := helper.Validate(domain, initialSubdomains)
				uniqueSubdomains = helper.Unique(validSubdomains)

				// Append each subdomain found to subdomains array
				for _, subdomain := range uniqueSubdomains {

					if strings.Contains(subdomain, "*.") {
						subdomain = strings.Split(subdomain, "*.")[1]
					}

					if state.Verbose == true {
						if state.Color == true {
							fmt.Printf("\n[%sCENSYS%s] %s", helper.Red, helper.Reset, subdomain)
						} else {
							fmt.Printf("\n[CENSYS] %s", subdomain)
						}
					}

					subdomains = append(subdomains, subdomain)
				}
			}
		}

		result.Subdomains = subdomains
		result.Error = nil
		ch <- result
		return

	} else {
		result.Subdomains = subdomains
		result.Error = nil
		ch <- result
		return
	}
}
