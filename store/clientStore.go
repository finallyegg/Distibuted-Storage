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
	"errors"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"strings"
	"time"
)

//=====================================================================
func HashFile(addr string, path string) string {
	// read file by maxChunk bytes and store each file data into an array
	lenList := rkMain(path)

	file, _ := os.Open(path)
	fileInfo, _ := file.Stat()

	var reciept ObjectFile
	reciept.ModTime = time.Now()
	reciept.Name = fileInfo.Name()
	reciept.Mode = fileInfo.Mode()
	reciept.Type = "file"
	reciept.Version = 1

	for i := 0; i < len(lenList); i++ {
		var message Message
		bufferChunk := make([]byte, lenList[i])
		file.Read(bufferChunk)

		message.Sig = ""
		message.Version = 1
		message.Data = bufferChunk
		message.Name = fileInfo.Name()
		message.Mode = fileInfo.Mode()
		message.ModTime = fileInfo.ModTime()
		message.Type = "put"
		jsonOutput, _ := json.MarshalIndent(message, "", "    ")
		response, err := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		body, _ := ioutil.ReadAll(response.Body)
		var example = Message{}
		json.Unmarshal(body, &example)
		sig := example.Sig
		reciept.Data = append(reciept.Data, sig)
		// println(sig)
	}
	jsonOutput, _ := json.MarshalIndent(reciept, "", "    ")
	var final_msg Message
	final_msg.Version = 1
	final_msg.Type = "put"
	final_msg.Data = jsonOutput
	jsonOutput, _ = json.MarshalIndent(final_msg, "", "    ")
	response, err := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	body, _ := ioutil.ReadAll(response.Body)
	var example = Message{}
	json.Unmarshal(body, &example)
	sig := example.Sig
	file.Close()
	return sig

}

func HashDir(addr string, path string) string {
	allFileNames := []string{}
	allFileSigs := []string{}

	root, err := os.Open(path)

	dirInfo, err := root.Stat()
	files, err := ioutil.ReadDir(path)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, file := range files {
		allFileNames = append(allFileNames, file.Name())
		//println(file.Name())
		if file.IsDir() {
			allFileSigs = append(allFileSigs, HashDir(addr, root.Name()+"/"+file.Name()))
		} else {
			allFileSigs = append(allFileSigs, HashFile(addr, root.Name()+"/"+file.Name()))
		}
	}
	// root.Close()
	dirReceipt := ObjectDir{Version: 1, Type: "dir", Name: dirInfo.Name(), PrevSig: "", FileNames: allFileNames, FileSigs: allFileSigs}
	jsonOutput, _ := json.MarshalIndent(dirReceipt, "", "    ")
	msg := Message{Version: 1, Type: "put", Data: jsonOutput}
	jsonOutput, _ = json.MarshalIndent(msg, "", "    ")
	response, err := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	body, _ := ioutil.ReadAll(response.Body)
	var example = Message{}
	json.Unmarshal(body, &example)
	root.Close()
	return example.Sig
}

func CmdPut(addr string, path string) {
	// path is a name of file
	// Do some chunking
	file, _ := os.Open(path)

	fileInfo, _ := file.Stat()
	var sig string
	if fileInfo.IsDir() {
		sig = HashDir(addr, path)
	} else {
		sig = HashFile(addr, path)
	}
	file.Close()
	updateLast(sig)
	fmt.Println(sig)
}

func CmdDel(addr string, sig string) {
	if sig == "last" {
		sig = getLast()
	}
	req := Message{Version: 1, Type: "delete", Sig: sig}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	res, err := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if res.StatusCode != 200 {
		fmt.Fprintln(os.Stderr, res.StatusCode)
	}
}

func CmdGet(addr, sig string, newName string, newPath string) error {
	if sig != "last" {
		updateLast(sig)
	} else {
		sig = getLast()
	}
	_, err := ioutil.ReadDir(newPath)
	if err != nil {
		if os.IsNotExist(err) {
			os.Mkdir(newPath, 0777)
			// println(newPath)
		} else {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	req := Message{Version: 1, Type: "get", Sig: sig}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	res, err := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
	if res.StatusCode != 200 {
		fmt.Fprintln(os.Stderr, res.StatusCode)
		return nil
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var m1 = Message{}
	json.Unmarshal(body, &m1)
	err = ioutil.WriteFile(newPath+"/"+newName, m1.Data, 0777) // TODO PathName Bug
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
func CmdGetFile(addr, sig string, newName string, newPath string) error {
	if sig != "last" {
		updateLast(sig)
	} else {
		sig = getLast()
	}
	_, err := ioutil.ReadDir(newPath)
	if err != nil {
		if os.IsNotExist(err) {
			os.Mkdir(newPath, 0777)
		} else {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	req := Message{Version: 1, Type: "get", Sig: sig}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	res, err := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var m1 = Message{}
	json.Unmarshal(body, &m1)

	var fileRecepit map[string]interface{}
	err = json.Unmarshal(m1.Data, &fileRecepit)
	if err != nil {
		return errors.New("Not Found")
	}
	// means we got a reciept
	if fileRecepit["Type"] == "dir" {
		var fileRecepit = ObjectDir{}
		err = json.Unmarshal(m1.Data, &fileRecepit)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for i := 0; i < len(fileRecepit.FileSigs); i++ {
			fileSig := fileRecepit.FileSigs[i]
			fileName := fileRecepit.FileNames[i]
			newpath := newPath + "/" + newName
			CmdGetFile(addr, fileSig, fileName, newpath)
		}
	} else if fileRecepit["Type"] == "file" {
		req := Message{Version: 1, Type: "getfile", Sig: sig}
		jsonOutput, _ := json.MarshalIndent(req, "", "    ")
		res, _ := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
		body, _ := ioutil.ReadAll(res.Body)
		var m1 = Message{}
		json.Unmarshal(body, &m1)
		ioutil.WriteFile(newPath+"/"+newName, m1.Data, 0666)
	}
	return nil
}

func CmdGetFileNoJSON(addr string, sig string, newName string) error {
	if sig != "last" {
		updateLast(sig)
	} else {
		sig = getLast()
	}
	url := addr + "/" + sig
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	filename := newName
	bodyByte, err := ioutil.ReadAll(res.Body)
	filetype := http.DetectContentType(bodyByte)
	s, _ := mime.ExtensionsByType(filetype)

	path := "./bloblocal/" + filename + s[len(s)-1]
	println(path)
	err = ioutil.WriteFile(path, bodyByte, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return nil
}

func CmdDesc(addr, sig string) string {
	if sig != "last" {
		updateLast(sig)
	} else {
		sig = getLast()
	}
	req := Message{Version: 1, Type: "get", Sig: sig}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	res, err := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if res.StatusCode != 200 {
		msg, _ := ioutil.ReadAll(res.Body)
		println(string(msg))
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var m1 = Message{}
	json.Unmarshal(body, &m1)
	var fileRecepit map[string]interface{}
	err = json.Unmarshal(m1.Data, &fileRecepit)

	var retVal string
	if err != nil {
		retVal = string(len(m1.Data))
	} else {
		retVal = string(m1.Data)
	}
	return retVal
}

func CmdInfo(addr string) error {
	req := Message{Version: 1, Type: "info"}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	res, err := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
	if res.StatusCode != 200 {
		fmt.Fprintln(os.Stderr, res.StatusCode)
	}
	body, err := ioutil.ReadAll(res.Body)
	var m1 = Message{}
	json.Unmarshal(body, &m1)
	println(m1.Info)
	return err
}

func CmdSync(s1 string, s2 string, height int) error {
	req := Message{Version: 1, Type: "sync", TreeTarget: s2, TreeHeight: height}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	res, _ := http.Post(s1, "application/json", bytes.NewReader(jsonOutput))
	if res.StatusCode != 200 {
		fmt.Fprintln(os.Stderr, res.StatusCode)
	}
	body, err := ioutil.ReadAll(res.Body)
	var m1 = Message{}
	json.Unmarshal(body, &m1)
	println(m1.Info)
	return err
}

func CmdSign(addr string, sig string, privateKeyPath string) string {
	if sig == "last" {
		sig = getLast()
	}
	privateKeyByte, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		PrintExit("err", err)
	}
	var privateKeyBlock *pem.Block
	privateKeyBlock, _ = pem.Decode(privateKeyByte)
	privateKey, _ := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)

	fileDesc := CmdDesc(addr, sig)
	fileDesc = strings.TrimSpace(fileDesc)

	if last := len(fileDesc) - 1; last >= 0 && fileDesc[last] == '}' {
		fileDesc = fileDesc[:last]
	}
	s256 := sha256.Sum256([]byte(fileDesc))
	rawSig, _ := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, s256[:])

	signature := base32.StdEncoding.EncodeToString(rawSig[:])
	fileDesc = fmt.Sprintf("%s%s%s%s%s%s", fileDesc, ",", "\n", `"signature": "`, signature, `"}`)

	req := Message{Version: 1, Type: "put", Sig: sig, Data: []byte(fileDesc)}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	res, _ := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
	if res.StatusCode != 200 {
		fmt.Fprintln(os.Stderr, res.StatusCode)
	}
	body, _ := ioutil.ReadAll(res.Body)
	var m1 = Message{}
	json.Unmarshal(body, &m1)
	fmt.Println(m1.Sig)
	updateLast(m1.Sig)
	return m1.Sig
}

func CmdVerify(addr string, sig string, publicKeyPath string) {
	if sig == "last" {
		sig = getLast()
	}
	publicKeyByte, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		PrintExit("err", err)
	}
	var publicKeyBlock *pem.Block
	publicKeyBlock, _ = pem.Decode(publicKeyByte)
	publicKey, _ := x509.ParsePKCS1PublicKey(publicKeyBlock.Bytes)

	fileDesc := CmdDesc(addr, sig)
	fileDesc = strings.TrimSpace(fileDesc)

	var fileRecepit map[string]interface{}
	json.Unmarshal([]byte(fileDesc), &fileRecepit)
	signature := fileRecepit["signature"]

	ind := strings.Index(fileDesc, `"signature"`)
	fileDesc = fileDesc[:ind-2]
	s256 := sha256.Sum256([]byte(fileDesc))

	signatureed, _ := base32.StdEncoding.DecodeString(signature.(string))
	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, s256[:], signatureed)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Verification succeeded")
	}
}

func CmdCreateAnchor(addr string, anchorName string, privateKeyP string) string {
	req := Message{Type: "anchor", Name: anchorName}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	sh := sha256.Sum256(jsonOutput)
	req.RandomID = base32.StdEncoding.EncodeToString(sh[:])
	jsonOutput, _ = json.MarshalIndent(req, "", "    ")

	if privateKeyP != "" {
		//Sign
		privateKeyByte, err := ioutil.ReadFile(privateKeyP)
		if err != nil {
			PrintExit("err", err)
		}
		var privateKeyBlock *pem.Block
		privateKeyBlock, _ = pem.Decode(privateKeyByte)
		privateKey, _ := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)

		fileDesc := string(jsonOutput)
		if last := len(fileDesc) - 1; last >= 0 && fileDesc[last] == '}' {
			fileDesc = fileDesc[:last]
		}
		s256, _ := base32.StdEncoding.DecodeString(req.RandomID)
		rawSig, _ := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, s256[:])

		signature := base32.StdEncoding.EncodeToString(rawSig[:])
		fileDesc = fmt.Sprintf("%s%s%s%s%s%s", fileDesc, ",", "\n", `"signature": "`, signature, `"}`)
		res, _ := http.Post(addr, "application/json", bytes.NewReader([]byte(fileDesc)))
		if res.StatusCode != 200 {
			fmt.Fprintln(os.Stderr, res.StatusCode)
		}
		body, _ := ioutil.ReadAll(res.Body)
		var m1 = Message{}
		json.Unmarshal(body, &m1)
		fmt.Println(m1.Sig)
		updateLast(m1.Sig)
		return m1.Sig
	}
	res, _ := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
	if res.StatusCode != 200 {
		fmt.Fprintln(os.Stderr, res.StatusCode)
	}
	body, _ := ioutil.ReadAll(res.Body)
	var m1 = Message{}
	json.Unmarshal(body, &m1)
	fmt.Println(m1.Sig)
	updateLast(m1.Sig)
	return m1.Sig

}

func CmdClaimAnchor(addr string, anchorName string, rootsigKey string, rootsigValue string, privateKeyP string) string {
	if rootsigValue == "last" {
		rootsigValue = getLast()
	}

	anchorHash := CmdRootanchor(addr, anchorName)
	fileDesc := CmdDesc(addr, anchorHash)

	var fileRecepit map[string]interface{}
	json.Unmarshal([]byte(fileDesc), &fileRecepit)
	signature, contains := fileRecepit["signature"]

	if contains {
		// veryify
		privateKeyByte, err := ioutil.ReadFile(privateKeyP)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Verification FAILED\n")
			return ""
		}
		var privateKeyBlock *pem.Block
		privateKeyBlock, _ = pem.Decode(privateKeyByte)
		privateKey, _ := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
		publicKey := privateKey.PublicKey
		s256, _ := base32.StdEncoding.DecodeString(anchorHash[10:])
		signatureed, _ := base32.StdEncoding.DecodeString(signature.(string))
		err = rsa.VerifyPKCS1v15(&publicKey, crypto.SHA256, s256[:], signatureed)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Verification FAILED\n")
			return ""
		}

	}

	req := Message{Type: "claim", Name: anchorName, RootsigKey: rootsigKey, RootsigValue: rootsigValue}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")

	if privateKeyP != "" {
		privateKeyByte, err := ioutil.ReadFile(privateKeyP)
		if err != nil {
			PrintExit("err", err)
		}
		var privateKeyBlock *pem.Block
		privateKeyBlock, _ = pem.Decode(privateKeyByte)
		privateKey, _ := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)

		s256 := sha256.Sum256(jsonOutput)
		req.RandomID = base32.StdEncoding.EncodeToString(s256[:])
		jsonOutput, _ := json.MarshalIndent(req, "", "    ")

		rawSig, _ := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, s256[:])
		signature := base32.StdEncoding.EncodeToString(rawSig[:])

		fileDesc := string(jsonOutput)

		if last := len(fileDesc) - 1; last >= 0 && fileDesc[last] == '}' {
			fileDesc = fileDesc[:last]
		}

		fileDesc = fmt.Sprintf("%s%s%s%s%s%s", fileDesc, ",", "\n", `"signature": "`, signature, `"}`)

		res, _ := http.Post(addr, "application/json", bytes.NewReader([]byte(fileDesc)))
		if res.StatusCode != 200 {
			fmt.Fprintln(os.Stderr, res.StatusCode)
		}
		body, _ := ioutil.ReadAll(res.Body)
		var m1 = Message{}
		json.Unmarshal(body, &m1)
		fmt.Println(m1.Sig)
		updateLast(m1.Sig)
		return m1.Sig
	}

	s256 := sha256.Sum256(jsonOutput)
	req.RandomID = base32.StdEncoding.EncodeToString(s256[:])
	jsonOutput, _ = json.MarshalIndent(req, "", "    ")
	res, err := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
	if err != nil {
		fmt.Println(err)
	}
	if res.StatusCode != 200 {
		fmt.Fprintln(os.Stderr, res.StatusCode)
	}
	body, _ := ioutil.ReadAll(res.Body)
	var m1 = Message{}
	json.Unmarshal(body, &m1)
	fmt.Println(m1.Sig)
	updateLast(m1.Sig)
	return m1.Sig

}

func CmdContent(addr string, anchorName string) {
	req := Message{Type: "content", Name: anchorName}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	res, err := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
	if err != nil {
		fmt.Println(err)
	}
	if res.StatusCode != 200 {
		fmt.Fprintln(os.Stderr, res.StatusCode)
	}
	body, _ := ioutil.ReadAll(res.Body)
	var m1 = Message{}
	json.Unmarshal(body, &m1)
	fmt.Println(m1.Adds["rootsig"])
	updateLast(m1.Adds["rootsig"])
}

func CmdRootanchor(addr string, anchorName string) string {
	req := Message{Type: "rootanchor", Name: anchorName}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	res, err := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
	if err != nil {
		fmt.Println(err)
	}
	if res.StatusCode != 200 {
		fmt.Fprintln(os.Stderr, res.StatusCode)
	}
	body, _ := ioutil.ReadAll(res.Body)
	var m1 = Message{}
	json.Unmarshal(body, &m1)
	updateLast(m1.Sig)
	return m1.Sig
}

func CmdLastClaim(addr string, anchorName string) {
	req := Message{Type: "lastclaim", Name: anchorName}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	res, err := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
	if err != nil {
		fmt.Println(err)
	}
	body, _ := ioutil.ReadAll(res.Body)
	var m1 = Message{}
	json.Unmarshal(body, &m1)
	fmt.Println(m1.Sig)
	updateLast(m1.Sig)
}

func CmdChain(addr string, anchorName string) {
	req := Message{Type: "chain", Name: anchorName}
	jsonOutput, _ := json.MarshalIndent(req, "", "    ")
	res, err := http.Post(addr, "application/json", bytes.NewReader(jsonOutput))
	if err != nil {
		PrintExit("ERROR", err)
	}
	if res.StatusCode != 200 {
		fmt.Fprintln(os.Stderr, res.StatusCode)
	}
	var m1 = Message{}
	// body, err := ioutil.ReadAll(res.Body)
	// fmt.Println(string(body))
	err = json.NewDecoder(res.Body).Decode(&m1)
	if err != nil {
		PrintExit("ERROR", err)
	}
	for _, v := range m1.Chain {
		fmt.Println(v)
	}
}
