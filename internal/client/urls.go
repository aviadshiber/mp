// Package client provides an HTTP client for the Mixpanel API with region-aware
// URL resolution, authentication, and rate-limit retry logic.
package client

import "fmt"

// API family constants identify the four Mixpanel API endpoint families.
const (
	APIFamilyQuery     = "query"
	APIFamilyExport    = "export"
	APIFamilyApp       = "app"
	APIFamilyIngestion = "ingestion"
)

// Region constants for the three supported Mixpanel data residency regions.
const (
	RegionUS = "us"
	RegionEU = "eu"
	RegionIN = "in"
)

// baseURLs maps (family, region) to the base URL prefix.
var baseURLs = map[string]map[string]string{
	APIFamilyQuery: {
		RegionUS: "https://mixpanel.com/api/query",
		RegionEU: "https://eu.mixpanel.com/api/query",
		RegionIN: "https://in.mixpanel.com/api/query",
	},
	APIFamilyExport: {
		RegionUS: "https://data.mixpanel.com/api/2.0",
		RegionEU: "https://data-eu.mixpanel.com/api/2.0",
		RegionIN: "https://data-in.mixpanel.com/api/2.0",
	},
	APIFamilyApp: {
		RegionUS: "https://mixpanel.com/api/app",
		RegionEU: "https://eu.mixpanel.com/api/app",
		RegionIN: "https://in.mixpanel.com/api/app",
	},
	APIFamilyIngestion: {
		RegionUS: "https://api.mixpanel.com",
		RegionEU: "https://api-eu.mixpanel.com",
		RegionIN: "https://api-in.mixpanel.com",
	},
}

// ResolveURL returns the full base URL for the given API family and region.
// It returns an error if the family or region is unknown.
func ResolveURL(family, region string) (string, error) {
	regions, ok := baseURLs[family]
	if !ok {
		return "", fmt.Errorf("unknown API family %q; valid families: query, export, app, ingestion", family)
	}
	url, ok := regions[region]
	if !ok {
		return "", fmt.Errorf("unknown region %q; valid regions: us, eu, in", region)
	}
	return url, nil
}

// ValidRegion reports whether r is a recognized region string.
func ValidRegion(r string) bool {
	return r == RegionUS || r == RegionEU || r == RegionIN
}
