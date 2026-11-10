package iplookup

import (
	"net/url"
	"strconv"
	"strings"
)

// LookupOptions are the parameters of a single lookup call. An empty IP
// looks up the public IP address observed by the API itself.
type LookupOptions struct {
	IP       string
	ASN      bool
	Location bool
	Abuse    bool
	RBL      bool
}

func buildURL(baseURL string, opts LookupOptions) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	if opts.IP != "" {
		u.Path = strings.TrimSuffix(u.Path, "/") + "/" + opts.IP
	}

	q := u.Query()
	q.Set("asn", strconv.FormatBool(opts.ASN))
	q.Set("location", strconv.FormatBool(opts.Location))
	q.Set("abuse", strconv.FormatBool(opts.Abuse))
	q.Set("rbl", strconv.FormatBool(opts.RBL))
	u.RawQuery = q.Encode()

	return u.String(), nil
}
