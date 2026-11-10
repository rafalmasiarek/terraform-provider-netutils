package iplookup

// Response is the full decoded API response. Not every optional section is
// guaranteed present - a successful response does not imply asn, location,
// abuse, or rbl are populated.
type Response struct {
	Status  string         `json:"status"`
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    Data           `json:"data"`
	Errors  []string       `json:"errors"`
	Meta    map[string]any `json:"meta"`
}

// Data is the response's "data" section.
type Data struct {
	IP       string    `json:"ip"`
	Hostname string    `json:"hostname"`
	Network  string    `json:"network"`
	ASN      *ASN      `json:"asn,omitempty"`
	Location *Location `json:"location,omitempty"`
	Abuse    *Abuse    `json:"abuse,omitempty"`
	RBL      *RBL      `json:"rbl,omitempty"`
}

// ASN is the response's "data.asn" section.
type ASN struct {
	ASN  string `json:"asn"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Location is the response's "data.location" section. Region and City are
// pointers because the API documents them as possibly null even when the
// section itself is present.
type Location struct {
	Country     string  `json:"country"`
	CountryName string  `json:"country_name"`
	Region      *string `json:"region"`
	City        *string `json:"city"`
	IsEU        bool    `json:"is_eu"`
}

// Abuse is the response's "data.abuse" section. The upstream API's exact
// field set for this section is not documented in this provider's own
// specification - callers needing structured abuse-contact fields should
// use RawResponse (see the netutils_ip_info data source) until that schema
// is confirmed against the real API.
type Abuse map[string]any

// RBL is the response's "data.rbl" section - same caveat as Abuse.
type RBL map[string]any
