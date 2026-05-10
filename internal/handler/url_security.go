package handler

import (
	"errors"
	"fmt"
	"net"
	"net/url"
)

var errInvalidPublicHTTPSURL = errors.New("URL must be https and resolve to a public address")

func validatePublicHTTPSURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return "", errInvalidPublicHTTPSURL
	}

	host := parsed.Hostname()
	if host == "" {
		return "", errInvalidPublicHTTPSURL
	}

	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return "", errInvalidPublicHTTPSURL
		}
	} else {
		ips, err := net.LookupIP(host)
		if err != nil {
			return "", fmt.Errorf("failed to resolve host: %w", err)
		}
		if len(ips) == 0 {
			return "", errInvalidPublicHTTPSURL
		}
		for _, ip := range ips {
			if !isPublicIP(ip) {
				return "", errInvalidPublicHTTPSURL
			}
		}
	}

	parsed.Path = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func isPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return !ip.IsLoopback() &&
		!ip.IsPrivate() &&
		!ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() &&
		!ip.IsUnspecified() &&
		!ip.IsMulticast()
}
