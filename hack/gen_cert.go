package main

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/3scale-ops/marin3r/pkg/util/pki"
	"github.com/spf13/cobra"
)

var (
	signerCertPath string
	signerKeyPath  string
	commonName     string
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
	cmd.Flags().StringVar(&commonName, "common-name", "localhost", "Common name for the certificate")
	// time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
	cmd.Flags().StringVar(&notBefore, "not-before", "", "Start of the certificate's validity period, in RFC3339 format as in '2006-01-02T15:04:05Z'")
	cmd.Flags().StringVar(&notAfter, "not-after", "", "End of the the certificate's validity period, in RFC3339 format as in '2006-01-02T15:04:05Z'")
	cmd.Flags().BoolVar(&isServer, "is-server-certificate", false, "Set true if the certificate is issued for server purposes (defaults to false)")
	cmd.Flags().BoolVar(&isCA, "is-ca-certificate", false, "Set true if the certificate is a certification authority (defaults to false)")
	cmd.Flags().IntVar(&keySize, "key-size", 2048, "Size of the RSA key")
	cmd.Flags().StringVar(&outFileName, "out", "", "Name of the output file. The extension '.crt' will be appended to the certificate file name and the "+
		"extension '.key' will be appended to the key file name. Stdout output if unset.")

	cmd.MarkFlagRequired("not-before")
	cmd.MarkFlagRequired("not-after")

}

func main() {
	cmd.Execute()
}

func run(cmd *cobra.Command, args []string) {

	before, err := time.Parse(time.RFC3339, notBefore)
	if err != nil {
		panic(err)
	}

	after, err := time.Parse(time.RFC3339, notAfter)
	if err != nil {
		panic(err)
	}

	var issuerCert *x509.Certificate
	var issuerKey interface{}

	if signerCertPath != "" && signerKeyPath != "" {
		var err error

		// CA signed
		scb, err := ioutil.ReadFile(signerCertPath)
		if err != nil {
			panic(err)
		}
		issuerCert, err = pki.LoadX509Certificate(scb)
		if err != nil {
			panic(err)
		}

		skb, err := ioutil.ReadFile(signerKeyPath)
		if err != nil {
			panic(err)
		}
		issuerKey, err = pki.DecodePrivateKeyBytes(skb)
		if err != nil {
			panic(err)
		}
	}

	crt, key, err := pki.GenerateCertificate(
		issuerCert,
		issuerKey,
		commonName,
		after.Sub(before),
		isServer,
		isCA,
		func() []string {
			if isServer {
				return []string{commonName}
			} else {
				return []string{}
			}
		}()...,
	)
	if err != nil {
		panic(err)
	}

	if outFileName != "" {
		if err := ioutil.WriteFile(outFileName+".crt", crt, 0644); err != nil {
			panic(err)
		}
		if err := ioutil.WriteFile(outFileName+".key", key, 0644); err != nil {
			panic(err)
		}

	} else {
		fmt.Printf("\n%s\n%s\n", string(crt), string(key))
	}
}
