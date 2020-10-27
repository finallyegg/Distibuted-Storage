package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
)

func main() {
	var publicKeyBlock pem.Block
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKeyBlock.Type = "RSA PUBLIC KEY"
	publicKeyBlock.Bytes = x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)

	var privateKeyBlock pem.Block
	privateKeyBlock.Type = "RSA PRIVATE KEY"
	privateKeyBlock.Bytes = x509.MarshalPKCS1PrivateKey(privateKey)

	publicBody := pem.EncodeToMemory(&publicKeyBlock)
	ioutil.WriteFile("public", publicBody, 0777)

	privateBody := pem.EncodeToMemory(&privateKeyBlock)
	ioutil.WriteFile("private", privateBody, 0777)
}
