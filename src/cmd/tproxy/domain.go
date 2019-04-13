//
// Domain name validator
//

package main

import (
	"errors"
	"fmt"
	"strings"
)

//
// Validate domain name
//
// This function validates international domain name, entered by user.
// It may either accept domain as is, return an error or suggest a
// replacement (for example, URL is replaced with naked hostname)
//
// This function attempts to be as tolerant as possible to user input.
// In particular, it accepts URLs and domains with port. If function
// accepts user input, the returned replacement will be a pure host
// name, converted to lower case
//
func DomainValidate(domain string) (string, error) {
	// Check for URL
	domain = domainCheckURL(domain)

	// Strip port, if any
	if i := strings.IndexByte(domain, ':'); i >= 0 {
		domain = domain[:i]
	}

	// Decode IDN
	domain = IDNEncode(domain)

	// Check total length
	if len(domain) > 253 {
		return "", errors.New("Domain name too long")
	}

	// Check for invalid characters
	for _, c := range domain {
		switch {
		// Underscore characters are not allowed in domain names, but some
		// browsers allow them, so we do the same
		case '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' ||
			c == '-' || c == '_' || c == '.':
		default:
			return "", fmt.Errorf("Domain name cannot contain character '%c'", c)
		}
	}

	// Split into labels and check each of them
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		// Labels cannot start or end with hyphens, but we are permissive
		// here in case somebody may register such a domain and
		// browsers are tolerant
		switch {
		case len(label) < 1:
			return "", errors.New("Domain name label cannot be empty")
		case len(label) > 63:
			return "", errors.New("Domain name label cannot exceed 63 bytes")
		}
	}

	return IDNDecode(domain), nil
}

//
// If domain looks like URL, fetch and return the hostname
// Otherwise, return the domain verbatim
//
func domainCheckURL(domain string) string {
	// URL starts with scheme://host/...,
	// where scheme must be [a-zA-Z][a-zA-Z0-9+-.]*
	for i := 0; i < len(domain); i++ {
		c := domain[i]
		switch {
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
		case '0' <= c && c <= '9' || c == '+' || c == '-' || c == '.':
			if i == 0 {
				return domain
			}

		case c == ':':
			if i == 0 {
				return domain
			}

			if !strings.HasPrefix(domain[i:], "://") {
				return domain
			}

			// Strip schema
			host := domain[i+3:]

			// Strip path, if any
			if j := strings.IndexByte(host, '/'); j >= 0 {
				host = host[:j]
			}

			return host

		default:
			return domain
		}
	}
	return domain
}
