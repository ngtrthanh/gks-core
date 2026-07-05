// Command seal_export produces a detached Ed25519 signature over the CNF export
// so external auditors can verify its integrity without database access.
//
//	seal_export [cnfPath] [sigPath]
//	  cnfPath default ../export/dump.cnf
//	  sigPath default <cnfPath>.sig
//
// The signing keypair (PKCS#8 / PKIX PEM) is loaded from SEAL_PRIVATE_KEY /
// SEAL_PUBLIC_KEY, or generated on first run. Uses only the standard library.
//
// SECURITY: the generated private key is written to the export/ artifact dir
// for local use only. Do NOT commit it; in production hold it in an HSM/KMS.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"os"
)

func arg(i int, def string) string {
	if len(os.Args) > i && os.Args[i] != "" {
		return os.Args[i]
	}
	return def
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func loadOrCreateKey(keyPath, pubPath string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	if b, err := os.ReadFile(keyPath); err == nil {
		blk, _ := pem.Decode(b)
		if blk == nil {
			return nil, nil, fmt.Errorf("no PEM block in %s", keyPath)
		}
		k, err := x509.ParsePKCS8PrivateKey(blk.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("parse private key: %w", err)
		}
		priv, ok := k.(ed25519.PrivateKey)
		if !ok {
			return nil, nil, fmt.Errorf("%s is not an Ed25519 private key", keyPath)
		}
		return priv, priv.Public().(ed25519.PublicKey), nil
	}

	// Generate a fresh keypair.
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}
	pkix, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(keyPath,
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}), 0o600); err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(pubPath,
		pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkix}), 0o644); err != nil {
		return nil, nil, err
	}
	log.Printf("generated new Ed25519 keypair: %s (private, mode 0600) / %s (public)", keyPath, pubPath)
	log.Printf("SECURITY: keep the private key secret; do not commit it to VCS")
	return priv, pub, nil
}

func main() {
	cnfPath := arg(1, "../export/dump.cnf")
	sigPath := arg(2, cnfPath+".sig")
	keyPath := getenv("SEAL_PRIVATE_KEY", "../export/ed25519_key.pem")
	pubPath := getenv("SEAL_PUBLIC_KEY", "../export/ed25519_key.pub")

	priv, pub, err := loadOrCreateKey(keyPath, pubPath)
	if err != nil {
		log.Fatalf("key: %v", err)
	}

	data, err := os.ReadFile(cnfPath)
	if err != nil {
		log.Fatalf("read %s: %v", cnfPath, err)
	}

	sig := ed25519.Sign(priv, data)
	if err := os.WriteFile(sigPath,
		[]byte(base64.StdEncoding.EncodeToString(sig)+"\n"), 0o644); err != nil {
		log.Fatalf("write signature: %v", err)
	}

	fmt.Printf("SEALED %s (%d bytes)\n", cnfPath, len(data))
	fmt.Printf("  algorithm  : Ed25519 (detached)\n")
	fmt.Printf("  signature  : %s\n", sigPath)
	fmt.Printf("  public key : %s\n", pubPath)
	fmt.Printf("  public key (base64): %s\n", base64.StdEncoding.EncodeToString(pub))
}
