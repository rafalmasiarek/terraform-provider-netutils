package netutil

import (
	"regexp"
	"strings"
)

var dnsLabelPattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

// DNSLabelValidate reports whether label is a valid single DNS label per
// RFC 1123: 1-63 characters, alphanumeric and hyphen only, must start and
// end with an alphanumeric character.
func DNSLabelValidate(label string) bool {
	return dnsLabelPattern.MatchString(label)
}

// HostnameValidate reports whether hostname is valid: every dot-separated
// label passes DNSLabelValidate, and the total length (including dots,
// excluding an optional trailing dot) is at most 253 characters.
func HostnameValidate(hostname string) bool {
	hostname = strings.TrimSuffix(hostname, ".")
	if hostname == "" || len(hostname) > 253 {
		return false
	}
	for _, label := range strings.Split(hostname, ".") {
		if !DNSLabelValidate(label) {
			return false
		}
	}
	return true
}

// HostnameNormalize lowercases hostname and sanitizes each dot-separated
// label independently: invalid characters become hyphens, runs of
// hyphens collapse to one, and leading/trailing hyphens are stripped.
// Empty labels produced by sanitization (or already present in the
// input) are dropped.
func HostnameNormalize(hostname string) string {
	labels := strings.Split(strings.ToLower(hostname), ".")
	var out []string
	for _, label := range labels {
		sanitized := sanitizeLabel(label)
		if sanitized != "" {
			out = append(out, sanitized)
		}
	}
	return strings.Join(out, ".")
}

var invalidLabelChar = regexp.MustCompile(`[^a-z0-9-]`)
var repeatedHyphen = regexp.MustCompile(`-+`)

func sanitizeLabel(label string) string {
	label = invalidLabelChar.ReplaceAllString(label, "-")
	label = repeatedHyphen.ReplaceAllString(label, "-")
	return strings.Trim(label, "-")
}

// FQDNJoin joins hostname and domain into a canonical, fully-qualified
// domain name with a trailing dot. A trailing dot already present on
// either argument is stripped first, so joining is never doubled.
func FQDNJoin(hostname, domain string) string {
	hostname = strings.TrimSuffix(hostname, ".")
	domain = strings.Trim(domain, ".")
	if domain == "" {
		return hostname + "."
	}
	return hostname + "." + domain + "."
}

// FQDNValidate reports whether fqdn is a valid fully-qualified domain
// name: every dot-separated label passes DNSLabelValidate, and the total
// length (excluding an optional trailing dot) is at most 253 characters.
// A trailing dot is accepted but not required.
func FQDNValidate(fqdn string) bool {
	return HostnameValidate(fqdn)
}
