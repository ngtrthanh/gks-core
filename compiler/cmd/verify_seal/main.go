// Command verify_seal checks a detached Ed25519 signature over the CNF export
// and prints "VERDICT: AUTHENTIC" or "VERDICT: TAMPERED".
//
//	verify_seal [cnfPath] [sigPath] [pubKeyPath]
//	  defaults: ../export/dump.cnf, <cnfPath>.sig, ../export/ed25519_key.pub
//
// Exit code 0 = AUTHENTIC, 1 = TAMPERED (or unreadable inputs). Stdlib only.
package main

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

func arg(i int, def string) string {
	if len(os.Args) > i && os.Args[i] != "" {
		return os.Args[i]
	}
	return def
}

func tampered(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	fmt.Println("VERDICT: TAMPERED")
	os.Exit(1)
}

func main() {
	cnfPath := arg(1, "../export/dump.cnf")
	sigPath := arg(2, cnfPath+".sig")
	pubPath := arg(3, "../export/ed25519_key.pub")

	data, err := os.ReadFile(cnfPath)
	if err != nil {
		tampered("cannot read CNF %s: %v", cnfPath, err)
	}

	sigText, err := os.ReadFile(sigPath)
	if err != nil {
		tampered("cannot read signature %s: %v", sigPath, err)
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigText)))
	if err != nil {
		tampered("signature not valid base64: %v", err)
	}

	pubPEM, err := os.ReadFile(pubPath)
	if err != nil {
		tampered("cannot read public key %s: %v", pubPath, err)
	}
	blk, _ := pem.Decode(pubPEM)
	if blk == nil {
		tampered("no PEM block in public key %s", pubPath)
	}
	pk, err := x509.ParsePKIXPublicKey(blk.Bytes)
	if err != nil {
		tampered("parse public key: %v", err)
	}
	pub, ok := pk.(ed25519.PublicKey)
	if !ok {
		tampered("%s is not an Ed25519 public key", pubPath)
	}

	if ed25519.Verify(pub, data, sig) {
		fmt.Println("VERDICT: AUTHENTIC")
		fmt.Printf("  cnf       : %s (%d bytes)\n", cnfPath, len(data))
		fmt.Printf("  signature : %s (Ed25519, verified)\n", sigPath)
		return
	}
	tampered("Ed25519 signature does not match %s", cnfPath)
}
