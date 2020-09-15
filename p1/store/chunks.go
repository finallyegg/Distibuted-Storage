package store

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"io"
	"io/ioutil"
	//"errors"
	"fmt"
	"os"
	"time"
)

type ObjectFile struct {
	Version int
	Type    string
	Name    string
	ModTime time.Time
	Mode    os.FileMode
	PrevSig string
	Data    []string
}

type ObjectDir struct {
	Version   int
	Type      string
	Name      string
	PrevSig   string
	FileNames []string
	FileSigs  []string
}

var CHUNK_DIR = os.Getenv("HOME") + "/.blobstore/"

const CHUNKSIZE = 8192

//=====================================================================
func IsEmpty(name string) bool {
	f, err := os.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true
	}
	return false // Either not empty or error, suits both cases
}

func HashDir(path string)string{

	allFileNames :=[]string{}
	allFileSigs :=[]string{}

	root,err := os.Open(path)

	dirInfo, err := root.Stat()
	files, err:= ioutil.ReadDir(path)

	if err !=nil{
		fmt.Fprintln(os.Stderr,err)
		os.Exit(1)
	}
	for _,file := range files{
		allFileNames = append(allFileNames,file.Name())
		//println(file.Name())
		if file.IsDir(){
			allFileSigs = append(allFileSigs, HashDir(root.Name() + "/" + file.Name()))
		}else{
			allFileSigs = append(allFileSigs, HashFile(root.Name() + "/" + file.Name()))
		}
	}
	root.Close()

	dirReceipt := ObjectDir{Version: 1,Type: "dir",Name: dirInfo.Name(),PrevSig: "",FileNames: allFileNames,FileSigs: allFileSigs}

	jsonOutput ,_ := json.MarshalIndent(dirReceipt, "", "    ")
	dirResult := jsonOutput
	//println(string(jsonOutput))
	sh := sha256.Sum256(dirResult)
	receiptHash := "sha256_32_" + base32.StdEncoding.EncodeToString(sh[:])
	ioutil.WriteFile("./blobstore/"+receiptHash,dirResult,0744)
	return receiptHash
}
func HashFile(path string) string{
	// read file by maxChunk bytes and store each file data into an array
	file, _ := os.Open(path)
	fileNames :=[]string{}
	fileInfo ,_ := file.Stat()
	for{
		bufferChunk := make([]byte,CHUNKSIZE)
		bytesRead , err := file.Read(bufferChunk)
		if err != nil{
			if err != io.EOF{
				if err !=nil{
					fmt.Fprintln(os.Stderr,err)
				}
			}
			break
		}
		bufferChunk = bufferChunk[:bytesRead]
		sha256Hash := sha256.Sum256(bufferChunk)
		bin32 := base32.StdEncoding.EncodeToString(sha256Hash[:])	// Convert to base32
		fileName := "sha256_32_" + bin32
		ioutil.WriteFile("./blobstore/"+fileName,bufferChunk,0744)
		fileNames = append(fileNames,fileName)
	}
	fileReceipt := ObjectFile{Version:1,Type: "file",Name: fileInfo.Name(),ModTime: time.Now(),Mode: fileInfo.Mode(),PrevSig: "",Data: fileNames}
	//println(string(fileReceipt.Data))
	jsonOutput ,_ := json.MarshalIndent(fileReceipt, "", "    ")
	fileResult := jsonOutput
	sh := sha256.Sum256(fileResult)
	receiptHash := "sha256_32_" + base32.StdEncoding.EncodeToString(sh[:])
	ioutil.WriteFile("./blobstore/"+receiptHash,fileResult,0744)
	return receiptHash
}
func CmdPut(arg string) {
	PrintAlways("cmd PUT %s\n", arg)
	_, err:=ioutil.ReadDir("./blobstore")
	if err != nil {
		if os.IsNotExist(err){
			os.Mkdir("./blobstore",os.ModePerm)
		}else {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	file , err := os.Open(arg)
	if err != nil {
		fmt.Fprintln(os.Stderr,err)
	}
	fileInfo,_ := file.Stat()
	if fileInfo.IsDir(){
		println(HashDir(arg))
	}else {
		println(HashFile(arg))
	}
	file.Close()
}

//==================================================================
func assembleFile(sig string, newPath string){
	directoryPath := "./blobstore/"
	data,_ := ioutil.ReadFile(directoryPath+sig)
	var fileRecepit ObjectFile
	json.Unmarshal(data,&fileRecepit)
	//println(string(data))
	fileData := []byte{}
	for _,dataName := range fileRecepit.Data{
		chunkData , _:= ioutil.ReadFile(directoryPath+dataName)
		fileData = append(fileData, chunkData...)
	}
	ioutil.WriteFile(newPath + "/" + fileRecepit.Name,fileData,0644) // TODO PathName Bug
}

func assembleDir(sig string, newPath string){
	directoryPath := "./blobstore/"
	data,_ := ioutil.ReadFile(directoryPath+sig)
	var dirReceipt ObjectDir
	json.Unmarshal(data,&dirReceipt)
	//println(string(data))
	//dirData := []byte{}
	newPath = newPath + "/" + dirReceipt.Name
	err := os.Mkdir(newPath,os.ModePerm)
	if err != nil{
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _,sigName := range dirReceipt.FileSigs{
		fileByte , _:= ioutil.ReadFile(directoryPath+sigName)
		var example map[string]interface{}
		json.Unmarshal(fileByte,&example)
		if example["Type"] == "file"{
			assembleFile(sigName,newPath)
		}else if example["Type"] == "dir"{
			assembleDir(sigName,newPath)
		}
	}
}

func CmdGet(sig string, newPath string) error {
	_, err:=ioutil.ReadDir(newPath)
	if err != nil {
		if os.IsNotExist(err){
			os.Mkdir(newPath,os.ModePerm)
		}else {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	directoryPath := "./blobstore/"
	PrintAlways("cmd GET sig %q, for %q\n", sig, newPath)
	data,err := ioutil.ReadFile(directoryPath+sig)
	if err!= nil{
		fmt.Fprintln(os.Stderr,err)
		os.Exit(1)
	}
	var fileRecepit map[string]interface{}
	err = json.Unmarshal(data,&fileRecepit)
	if err!= nil{
		fmt.Fprintln(os.Stderr,err)
		os.Exit(1)
	}
	if fileRecepit["Type"] == "dir"{
		assembleDir(sig,newPath)
	}else if fileRecepit["Type"] == "file"{
		assembleFile(sig,newPath)
	}
	return nil
}

//=====================================================================

func CmdDesc(sig string) {
	//PrintAlways("cmd DESC sig %q\n", sig, "newPath")
	sig = "./blobstore/" + sig
	data,err := ioutil.ReadFile(sig)
	if err!= nil{
		fmt.Fprintln(os.Stderr,err)
		os.Exit(1)
	}
	var example map[string]interface{}
	err = json.Unmarshal(data,&example)
	if err!= nil{
		fmt.Fprintln(os.Stdout,len(data))
		os.Exit(0)
	}
	//fmt.Println(example["Type"] )
	fmt.Fprintln(os.Stderr,string(data))

}
