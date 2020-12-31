package store

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base32"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

// var (
// 	Debug = true
// )

func PrintAssert(cond bool, s string, args ...interface{}) {
	if !cond {
		PrintExit(s, args...)
	}
}

// func PrintDebug(s string, args ...interface{}) {
// 	if !Debug {
// 		return
// 	}
// 	PrintAlways(s, args...)
// }

func PrintExit(s string, args ...interface{}) {
	PrintAlways(s, args...)
	os.Exit(1)
}

func PrintAlways(s string, args ...interface{}) {
	fmt.Printf(s, args...)
}

func updateLast(sig string) {
	if err := ioutil.WriteFile("./last", []byte(sig), 0777); err != nil {
		PrintAlways("\nERROR: Unable to write %q to %q\n", sig, ".last")
	}
}

func getLast() string {
	bytes, err := ioutil.ReadFile("./last")
	if err != nil {
		return "none"
	} else {
		return string(bytes)
	}
}

// func GetRecieptinRaw(sig string, servAddr string) map[string]interface{} {
// 	req := Message{Version: 1, Type: "get", Sig: sig}
// 	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
// 	res, _ := http.Post(servAddr, "application/json", bytes.NewReader(jsonOutput))
// 	body, _ := ioutil.ReadAll(res.Body)
// 	var fileRecepit map[string]interface{}
// 	json.Unmarshal(body, &fileRecepit)
// 	return fileRecepit
// }

func GetRecieptinRaw(sig string, servAddr string) (map[string]interface{}, error) {
	req := Message{Version: 1, Type: "get", Sig: sig}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	res, _ := http.Post(servAddr, "application/json", bytes.NewReader(jsonOutput))
	body, _ := ioutil.ReadAll(res.Body)
	var msg Message
	err := json.Unmarshal(body, &msg)

	var fileRcpRaw map[string]interface{}
	err = json.Unmarshal(msg.Data, &fileRcpRaw)
	if err != nil {
		return nil, err
	}
	return fileRcpRaw, nil
}
func GetRecieptinBytes(sig string, servAddr string) ([]byte, error) {
	req := Message{Version: 1, Type: "get", Sig: sig}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	res, _ := http.Post(servAddr, "application/json", bytes.NewReader(jsonOutput))
	body, _ := ioutil.ReadAll(res.Body)
	var msg Message
	err := json.Unmarshal(body, &msg)
	if err != nil {
		return nil, err
	}
	return msg.Data, nil
}

func sign(fileRcpString *string, privateKeyPath string) string {
	privateKeyByte, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		PrintExit("err", err)
	}

	// add signature at end of file receiept
	var privateKeyBlock *pem.Block
	privateKeyBlock, _ = pem.Decode(privateKeyByte)
	privateKey, _ := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)

	if last := len(*fileRcpString) - 1; last >= 0 && (*fileRcpString)[last] == '}' {
		*fileRcpString = (*fileRcpString)[:last]
	}
	s256 := sha256.Sum256([]byte(*fileRcpString))
	rawSig, _ := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, s256[:])

	signature := base32.StdEncoding.EncodeToString(rawSig[:])
	*fileRcpString = fmt.Sprintf("%s%s%s%s%s%s", *fileRcpString, ",", "\n", `"signature": "`, signature, `"}`)
	return signature
}
