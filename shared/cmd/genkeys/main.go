package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func main() {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)

	publicKeyBase64 := base64.StdEncoding.EncodeToString(pub)
	privateKeyBase64 := base64.StdEncoding.EncodeToString(priv)

	fmt.Println("--- SECRETS ---")
	fmt.Printf("ACCESS_TOKEN_PUBLIC_KEY=%s\n", publicKeyBase64)
	fmt.Printf("ACCESS_TOKEN_PRIVATE_KEY=%s\n", privateKeyBase64)
}
