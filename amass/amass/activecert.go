// Copyright 2017 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package amass

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTLSConnectTimeout = 1 * time.Second
	defaultHandshakeDeadline = 3 * time.Second
)

// pullCertificate - Attempts to pull a cert from several ports on an IP
func PullCertificate(addr string, config *AmassConfig, add bool) {
	var requests []*AmassRequest

	// Check hosts for certificates that contain subdomain names
	for _, port := range config.Ports {
		cfg := &tls.Config{InsecureSkipVerify: true}
		// Set the maximum time allowed for making the connection
		ctx, cancel := context.WithTimeout(context.Background(), defaultTLSConnectTimeout)
		defer cancel()
		// Obtain the connection
		conn, err := DialContext(ctx, "tcp", addr+":"+strconv.Itoa(port))
		if err != nil {
			continue
		}
		defer conn.Close()
		c := tls.Client(conn, cfg)
		// Attempt to acquire the certificate chain
		errChan := make(chan error, 2)
		// This goroutine will break us out of the handshake
		time.AfterFunc(defaultHandshakeDeadline, func() {
			errChan <- errors.New("Handshake timeout")
		})
		// Be sure we do not wait too long in this attempt
		c.SetDeadline(time.Now().Add(defaultHandshakeDeadline))
		// The handshake is performed in the goroutine
		go func() {
			errChan <- c.Handshake()
		}()
		// The error channel returns handshake or timeout error
		if err = <-errChan; err != nil {
			continue
		}
		// Get the correct certificate in the chain
		certChain := c.ConnectionState().PeerCertificates
		cert := certChain[0]
		// Create the new requests from names found within the cert
		requests = append(requests, reqFromNames(namesFromCert(cert), config)...)
	}
	// Get all unique root domain names from the generated requests
	if add {
		var domains []string
		for _, r := range requests {
			domains = UniqueAppend(domains, r.Domain)
		}
		config.AddDomains(domains)
	}

	for _, req := range requests {
		for _, domain := range config.Domains() {
			if req.Domain == domain {
				config.dns.SendRequest(req)
				break
			}
		}
	}
}

func namesFromCert(cert *x509.Certificate) []string {
	var cn string

	for _, name := range cert.Subject.Names {
		oid := name.Type
		if len(oid) == 4 && oid[0] == 2 && oid[1] == 5 && oid[2] == 4 {
			if oid[3] == 3 {
				cn = fmt.Sprintf("%s", name.Value)
				break
			}
		}
	}

	var subdomains []string
	// Add the subject common name to the list of subdomain names
	commonName := removeAsteriskLabel(cn)
	if commonName != "" {
		subdomains = append(subdomains, commonName)
	}
	// Add the cert DNS names to the list of subdomain names
	for _, name := range cert.DNSNames {
		n := removeAsteriskLabel(name)
		if n != "" {
			subdomains = UniqueAppend(subdomains, n)
		}
	}
	return subdomains
}

func removeAsteriskLabel(s string) string {
	var index int

	labels := strings.Split(s, ".")
	for i := len(labels) - 1; i >= 0; i-- {
		if strings.TrimSpace(labels[i]) == "*" {
			break
		}
		index = i
	}
	if index == len(labels)-1 {
		return ""
	}
	return strings.Join(labels[index:], ".")
}

func reqFromNames(subdomains []string, config *AmassConfig) []*AmassRequest {
	var requests []*AmassRequest

	// For each subdomain name, attempt to make a new AmassRequest
	for _, name := range subdomains {
		requests = append(requests, &AmassRequest{
			Name:   name,
			Domain: config.dns.SubdomainToDomain(name),
			Tag:    "cert",
			Source: "Active Cert",
		})
	}
	return requests
}
