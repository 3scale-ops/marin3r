package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/3scale-ops/marin3r/pkg/util/pki"
	"github.com/spf13/cobra"
)

var (
	key            string
	signerCertPath string
	signerKeyPath  string
	commonName     string
	host           string
	notAfter       string
	notBefore      string
	isServer       bool
	isCA           bool
	keySize        int
	outFileName    string
)

var cmd = &cobra.Command{
	Use:   "gen-cert",
	Short: "Small script to generate certificates for testing purposes",
	Run:   run,
}

func init() {
	cmd.Flags().StringVar(&signerCertPath, "signer-cert", "", "Filesystem location of the signing certificate in PEM format (self-signed if unset)")
	cmd.Flags().StringVar(&signerKeyPath, "signer-key", "", "Filesystem location of the signing certificate key in PEM format (self-signed if unset)")
	cmd.Flags().StringVar(&commonName, "common-name", "localhost", "Common name for the certificate (default is 'localhost')")
	cmd.Flags().StringVar(&host, "host", "localhost", "Alternate server name for the certificate (default is 'localhost')")
	// time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
	cmd.Flags().StringVar(&notBefore, "not-before", "", "Start of the certificate's validity period, in RFC3339 format as in '2006-01-02T15:04:05Z'")
	cmd.Flags().StringVar(&notAfter, "not-after", "", "End of the the certificate's validity period, in RFC3339 format as in '2006-01-02T15:04:05Z'")
	cmd.Flags().BoolVar(&isServer, "is-server-certificate", false, "Set true if the certificate is issued for server purposes (defaults to false)")
	cmd.Flags().BoolVar(&isCA, "is-ca-certificate", false, "Set true if the certificate is a certification authority (defaults to false)")
	cmd.Flags().IntVar(&keySize, "key-size", 2048, "Size of the RSA key (default is 2048)")
	cmd.Flags().StringVar(&outFileName, "out", "", "Name of the output file. The extension '.crt' will be appended to the certificate file name and the "+
		"extension '.key' will be appended to the key file name. Stdout output if unset.")

	cmd.MarkFlagRequired("not-before")
	cmd.MarkFlagRequired("not-after")

}

func main() {
	cmd.Execute()
}

func run(cmd *cobra.Command, args []string) {

	priv, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		panic(err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		panic(err)
	}

	before, err := time.Parse(time.RFC3339, notBefore)
	if err != nil {
		panic(err)
	}

	after, err := time.Parse(time.RFC3339, notAfter)
	if err != nil {
		panic(err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"marin3r.test"},
			CommonName:   commonName,
		},
		NotBefore: before,
		NotAfter:  after,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	if isServer {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}
	if isCA {
		template.IsCA = true
		template.KeyUsage = x509.KeyUsageCertSign
	}

	var derBytes []byte
	if signerCertPath == "" && signerKeyPath == "" {
		// Self-signed
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
		if err != nil {
			panic(err)
		}

	} else {
		// CA signed
		scb, err := ioutil.ReadFile(signerCertPath)
		if err != nil {
			panic(err)
		}
		signerCert, err := pki.LoadX509Certificate(scb)
		if err != nil {
			panic(err)
		}

		skb, err := ioutil.ReadFile(signerKeyPath)
		if err != nil {
			panic(err)
		}
		signerKey, err := pki.DecodePrivateKeyBytes(skb)
		if err != nil {
			panic(err)
		}

		derBytes, err = x509.CreateCertificate(rand.Reader, &template, signerCert, &priv.PublicKey, signerKey)
		if err != nil {
			panic(err)
		}
	}

	crtPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	if outFileName != "" {
		if err := ioutil.WriteFile(outFileName+".crt", crtPEM, 0644); err != nil {
			panic(err)
		}
		if err := ioutil.WriteFile(outFileName+".key", keyPEM, 0644); err != nil {
			panic(err)
		}

	} else {
		fmt.Printf("\n%s\n%s\n", string(crtPEM), string(keyPEM))
	}
}
