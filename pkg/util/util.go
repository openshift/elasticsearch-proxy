package util

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
)

func GetCertPool(paths []string, system_roots bool) (*x509.CertPool, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("Invalid empty list of Root CAs file paths")
	}
	var pool *x509.CertPool
	if system_roots {
		// ignore errors
		pool, _ = x509.SystemCertPool()
		if pool == nil {
			log.Printf("No system certificates found")
			pool = x509.NewCertPool()
		}
	} else {
		pool = x509.NewCertPool()
	}
	for _, path := range paths {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("certificate authority file (%s) could not be read - %s", path, err)
		}
		if !pool.AppendCertsFromPEM(data) {
			return nil, fmt.Errorf("loading certificate authority (%s) failed", path)
		}
	}
	return pool, nil
}
