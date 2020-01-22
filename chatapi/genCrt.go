package chatapi

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
)

type PKCS8Key struct {
	Version             int
	PrivateKeyAlgorithm []asn1.ObjectIdentifier
	PrivateKey          []byte
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func MarshalPKCS8PrivateKey(key *rsa.PrivateKey) ([]byte, error) {
	var pkey PKCS8Key
	pkey.Version = 0
	pkey.PrivateKeyAlgorithm = make([]asn1.ObjectIdentifier, 1)
	pkey.PrivateKeyAlgorithm[0] = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
	pkey.PrivateKey = x509.MarshalPKCS1PrivateKey(key)
	return asn1.Marshal(pkey)
}

func GenCrt(host string) {
	flag.Parse()
	if len(host) == 0 {
		log.Fatalf("Missing required host parameter")
	}
	_, err := os.Stat("./burpServer.pem")
	if err == nil {
		fmt.Println("file", "burpServer.pem found no need to generate new key")
		return
	} else {
		fmt.Println("creating new certificates")
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
	}

	pkcs8Bytes, err := MarshalPKCS8PrivateKey(priv)
	if err != nil {
		log.Fatalf("failed to convert private key to PKCS8: %s", err)
	}

	notBefore, err := time.Parse("Mon Jan _2 15:04:05 2006", time.Now().Format("Mon Jan _2 15:04:05 2006"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse creation date: %s\n", err)
		os.Exit(1)
	}
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"BurpTeamServer"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	certOut, err := os.Create("burpServer.pem")
	if err != nil {
		log.Fatalf("failed to open burpServer.pem for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()
	log.Print("written burpServer.pem\n")

	keyOut, err := os.OpenFile("burpServer.key", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Print("failed to open burpServer.key for writing:", err)
		return
	}
	pem.Encode(keyOut,
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: pkcs8Bytes,
		},
	)
	keyOut.Close()
	log.Print("written burpServer.key\n")
}
