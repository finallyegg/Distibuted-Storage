package store

import (
	"bytes"
	"crypto"
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
	"strconv"
	"strings"
	"time"
)

var (
	publicKeyPath                       = ""    // publicKey path
	requiredAnchorSigned                = false // is signed anchor required?
	publicKey            *rsa.PublicKey = nil
	dbPath                              = ""
)

// verify take a file sig and verify if it signed by same public key
func verify(fileContent []byte, signedSigature string) bool {
	if publicKey == nil {
		return true
	}

	fileContentString := string(fileContent)
	fileContentString = strings.TrimSpace(fileContentString)

	// extract "true content for hash function"
	ind := strings.Index(fileContentString, `"signature"`)
	fileContentString = fileContentString[:ind-2]
	s256 := sha256.Sum256([]byte(fileContentString))

	signatureed, _ := base32.StdEncoding.DecodeString(signedSigature)
	err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, s256[:], signatureed)
	if err != nil {
		return false
	}
	return true
}

func Serve(servAddr string, sqlPath string, pk string, anchorSigned bool) {
	publicKeyPath = pk
	requiredAnchorSigned = anchorSigned
	dbPath = sqlPath
	if publicKeyPath != "" {
		// if we found public Key, then update the global public key var
		publicKeyByte, _ := ioutil.ReadFile(publicKeyPath)
		var publicKeyBlock *pem.Block
		publicKeyBlock, _ = pem.Decode(publicKeyByte)
		publicKey, _ = x509.ParsePKCS1PublicKey(publicKeyBlock.Bytes)
	}
	treeMap := make(map[string]map[string]*TreeNode)
	anchorMap := make(map[string]string)
	chainMap := make(map[string][]string)
	timeMap := make(map[string]string)
	nodeMap := make(map[uint64][]string)

	CreateDB(dbPath) // init DB
	http.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		var reqMsg = Message{}
		err := json.Unmarshal(body, &reqMsg)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
		}
		switch reqMsg.Type {

		// case "sign":
		// 	fileRcpRaw := reqMsg.Data
		// 	signedSig := reqMsg.Refsig
		// 	if !verify(fileRcpRaw, signedSig) {
		// 		http.Error(w, "Signature failed", http.StatusBadRequest)
		// 		break
		// 	}
		// 	updateFileDB(dbPath, reqMsg.Sig, fileRcpRaw)
		// 	respMsg := Message{Version: 1, Type: "signresp", Sig: reqMsg.Sig}
		// 	w.Header().Set("Content-Type", "application/json")
		// 	json.NewEncoder(w).Encode(respMsg)

		case "anchor":
			anchorName := reqMsg.Name
			sh := sha256.Sum256(reqMsg.Data)
			receiptHash := "sha256_32_" + base32.StdEncoding.EncodeToString(sh[:])
			if requiredAnchorSigned {
				// if anchor has no sig
				if reqMsg.Refsig == "" || !verify(reqMsg.Data, reqMsg.Refsig) {
					http.Error(w, "Signature failed", http.StatusBadRequest)
					break
				}
			}
			if publicKey != nil && reqMsg.Refsig != "" {
				if !verify(reqMsg.Data, reqMsg.Refsig) {
					http.Error(w, "Signature failed", http.StatusBadRequest)
					break
				}
			}
			InsertIntoDB(sqlPath, receiptHash, reqMsg.Data)

			anchorMap[anchorName] = receiptHash
			chainMap[anchorName] = append(chainMap[anchorName], receiptHash)

			msg := Message{Version: 1, Type: "anchorresp", Sig: receiptHash}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(msg)
		case "claimHead":

			nodeBuf := DNode2{}
			json.Unmarshal(reqMsg.Data, &nodeBuf)

			anchorName := "HEAD"
			sh := sha256.Sum256(reqMsg.Data)
			receiptHash := "sha256_32_" + base32.StdEncoding.EncodeToString(sh[:])

			InsertIntoDB(sqlPath, receiptHash, reqMsg.Data)
			chainMap[anchorName] = append(chainMap[anchorName], receiptHash)
			timeMap[nodeBuf.Attrs.Mtime.Format("2006-1-2T15:04")] = receiptHash

			msg := Message{Version: 1, Type: "claimresp", Sig: receiptHash}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(msg)

		case "selectClaim":
			timeString := reqMsg.Info
			parsedTime, _ := parseLocalTime(timeString)
			formattedTime := parsedTime.Format("2006-1-2T15:04")
			headSig, contained := timeMap[formattedTime]
			fmt.Println(timeMap)
			if !contained {
				http.Error(w, "NO SUCH HEAD", http.StatusBadRequest)
			}
			msg := Message{Version: 1, Type: "claimresp", Sig: headSig}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(msg)

		case "claim":
			anchorName := reqMsg.Name
			_, contains := anchorMap[anchorName]
			if !contains {
				http.Error(w, "NO ANCHOR ERROR", http.StatusBadRequest)
			}
			sh := sha256.Sum256(reqMsg.Data)
			receiptHash := "sha256_32_" + base32.StdEncoding.EncodeToString(sh[:])
			if requiredAnchorSigned {
				// if anchor has no sig
				if reqMsg.Sig == "" || !verify(reqMsg.Data, reqMsg.Sig) {
					http.Error(w, "Signature failed", http.StatusBadRequest)
					break
				}
			}
			if publicKey != nil && reqMsg.Sig != "" {
				if !verify(reqMsg.Data, reqMsg.Sig) {
					http.Error(w, "Signature failed", http.StatusBadRequest)
					break
				}
			}
			InsertIntoDB(sqlPath, receiptHash, reqMsg.Data)

			chainMap[anchorName] = append(chainMap[anchorName], receiptHash)

			msg := Message{Version: 1, Type: "claimresp", Sig: receiptHash}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(msg)

		case "content":
			anchorName := reqMsg.Name
			rootHashArray, contains := chainMap[anchorName]
			if contains {
				rootHash := rootHashArray[len(rootHashArray)-1]
				data, err := GetFromDB(sqlPath, rootHash)
				if err != nil {
					http.Error(w, "GET DB ERROR", http.StatusInternalServerError)
				}
				var msg Message
				json.Unmarshal(data, &msg)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(msg)
			} else {
				http.Error(w, "NO ANCHOR ERROR", http.StatusInternalServerError)
			}

		case "rootanchor":
			anchorName := reqMsg.Name
			rootHash, contains := anchorMap[anchorName]
			if contains {
				var msg Message
				msg.Sig = rootHash
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(msg)
			} else {
				http.Error(w, "NO ANCHOR ERROR", http.StatusInternalServerError)
			}

		case "lastclaim":
			anchorName := reqMsg.Name
			rootHashArray, contains := chainMap[anchorName]
			if contains {
				rootHash := rootHashArray[len(rootHashArray)-1]
				var msg Message
				msg.Sig = rootHash
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(&msg)
			} else {
				http.Error(w, "NO ANCHOR ERROR", http.StatusInternalServerError)
			}

		case "chain":
			anchorName := reqMsg.Name
			rootHashArray, contains := chainMap[anchorName]
			if contains {
				var msg Message
				finalArray := append([]string{anchorMap[anchorName]}, rootHashArray...)
				msg.Type = "chainresp"
				msg.Prevsig = finalArray[0]
				msg.Chain = finalArray[:]
				w.Header().Set("Content-Type", "application/json")
				jsonOutput, _ := json.MarshalIndent(msg, "", "    ")
				w.Write(jsonOutput)
				if err != nil {
					http.Error(w, "ENCODE", http.StatusInternalServerError)
				}
			} else {
				http.Error(w, "NO ANCHOR ERROR", http.StatusInternalServerError)
			}

		case "putDNode":
			data := reqMsg.Data
			sh := sha256.Sum256(data)
			receiptHash := "sha256_32_" + base32.StdEncoding.EncodeToString(sh[:])
			InsertIntoDB(sqlPath, receiptHash, data)

			nodeBuf := DNode2{}
			json.Unmarshal(data, &nodeBuf)
			Inode := nodeBuf.Attrs.Inode
			nodeMap[Inode] = append(nodeMap[Inode], receiptHash)

			msg := Message{Version: 1, Type: "putDNoderesp", Sig: receiptHash}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(msg)

		case "put":

			data := reqMsg.Data
			sh := sha256.Sum256(data)
			receiptHash := "sha256_32_" + base32.StdEncoding.EncodeToString(sh[:])
			InsertIntoDB(sqlPath, receiptHash, data)
			msg := Message{Version: 1, Type: "putresp", Sig: receiptHash}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(msg)

		case "getAllVersions":
			inode, err := strconv.ParseUint(reqMsg.Name, 10, 64)
			if err != nil {
				fmt.Println("NOT VALID U64 INT")
			}
			sigs, contained := nodeMap[inode]
			respMsg := Message{}
			if contained {
				respMsg.Chain = make([]string, len(sigs))
			} else {
				fmt.Println("NOT FOUND", inode)
				respMsg.Chain = make([]string, 0)
			}
			copy(respMsg.Chain, sigs)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(respMsg)

		case "get":
			sig := reqMsg.Sig
			data, err := GetFromDB(sqlPath, sig)
			if err != nil {
				http.Error(w, "GET DB ERROR", http.StatusInternalServerError)
			}
			msg := Message{Version: 1, Type: "getresp", Data: data}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(msg)

		case "delete":
			treeMap = make(map[string]map[string]*TreeNode)
			sig := reqMsg.Sig
			err := DelFromDB(sqlPath, sig)
			if err != nil {
				http.Error(w, "DEL ERROR", http.StatusInternalServerError)
			}
			msg := Message{Version: 1, Type: "delresp"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(msg)

		case "getfile":
			sig := reqMsg.Sig
			data, err := GetFromDB(sqlPath, sig)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				http.Error(w, "Not Found", http.StatusNotFound)
			} else {
				msg := Message{Version: 1, Type: "getfileresp"}
				var fileRecepit = ObjectFile{}
				err = json.Unmarshal(data, &fileRecepit)
				if err != nil {
					http.Error(w, "Not Reciept", http.StatusInternalServerError)
				} else {
					msg.ModTime = fileRecepit.ModTime
					msg.Mode = fileRecepit.Mode
					msg.Name = fileRecepit.Name
					var fileData []byte
					for i := 0; i < len(fileRecepit.Data); i++ {
						tempSig := fileRecepit.Data[i]
						data, err := GetFromDB(sqlPath, tempSig)
						if err != nil {
							// os.Exit(1)
						}
						fileData = append(fileData, data...)
					}
					msg.Data = fileData
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(msg)
				}
			}
		case "info":
			str, err := InfoDB(sqlPath)
			if err != nil {
				http.Error(w, "Info Failed", http.StatusInternalServerError)
			}
			msg := Message{Version: 1, Type: "inforesp", Info: str}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(msg)

		case "sync":
			height := reqMsg.TreeHeight
			s2 := reqMsg.TreeTarget
			totalSize := 0
			totalRequest := 0
			totalChunk := 0
			// step 2
			msg := Message{Version: 1, Type: "tree", TreeHeight: height}
			jsonOutput, _ := json.MarshalIndent(msg, "", "    ")

			res, _ := http.Post(s2, "application/json", bytes.NewReader(jsonOutput))
			if res.StatusCode != 200 {
				fmt.Fprintln(os.Stderr, res.StatusCode)
			}
			totalSize += len(jsonOutput)
			// fmt.Println("root")
			totalRequest++

			//step 4
			root := TreeNode{}
			ConstructTree(&root, 1, height)
			treeMap[root.Sig] = FillTree(sqlPath, &root, height)

			// res contains root hash from s2
			body, _ := ioutil.ReadAll(res.Body)
			var m1 = Message{}
			json.Unmarshal(body, &m1)
			totalSize += len(body)

			if m1.Node.Sig != root.Sig {
				compareNode(m1.Node.Sig, &root, m1.Node, s2, &totalSize, &totalChunk, &totalRequest, sqlPath)
				// compareNode(s2RootSig string, s1Node *TreeNode, s2Node *TreeNode, s2Addr string, totalSize *int, totalChunk *int, totalRequest *int, DBpath string)
			}

			str := fmt.Sprintf("sync took %d requests, %d bytes, pulled %d chunks", totalRequest, totalSize, totalChunk)
			msg = Message{Version: 1, Type: "syncresp", Info: str}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(msg)

		case "tree":
			// step 2
			if reqMsg.TreeSig == "" {
				height := reqMsg.TreeHeight
				root := TreeNode{}
				ConstructTree(&root, 1, height) //Construct empty tree
				treeMap[root.Sig] = FillTree(sqlPath, &root, height)
				msg := Message{Version: 1, Type: "treeresp", Node: &root}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(msg)
			} else {
				targetSig := reqMsg.TreeSig
				node := treeMap[targetSig][reqMsg.Sig]
				msg := Message{Version: 1, Type: "treeresp", Node: node}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(msg)
			}
			// case "updateReciept":
			// 	sig := reqMsg.Sig
			// 	data := example.Data
			// 	updateFileDB(sqlPath, sig, data)
			// 	msg := Message{Version: 1, Type: "updateRecieptresp", Node: node}
		}

	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		sig := (r.URL.Path)[1:]
		data, err := GetFromDB(sqlPath, sig)
		if err != nil {
			http.Error(w, "Not Found", http.StatusNotFound)
		} else {
			var fileRecepit = ObjectFile{}
			err = json.Unmarshal(data, &fileRecepit)
			if err != nil {
				http.Error(w, "Not Reciept", http.StatusInternalServerError)
			} else {
				var fileData []byte
				for i := 0; i < len(fileRecepit.Data); i++ {
					tempSig := fileRecepit.Data[i]
					data, err := GetFromDB(sqlPath, tempSig)
					if err != nil {
						// os.Exit(1)
					}
					fileData = append(fileData, data...)
				}
				applicationType := http.DetectContentType(fileData)
				w.Header().Set("Content-Type", applicationType)
				fmt.Fprintf(w, "%s", fileData)
			}
		}
	})
	err := http.ListenAndServe(servAddr, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func compareNode(s2RootSig string, s1Node *TreeNode, s2Node *TreeNode, s2Addr string, totalSize *int, totalChunk *int, totalRequest *int, DBpath string) {
	if len(s1Node.children) == 0 {
		s1Map := make(map[string]int)
		for i, v := range s1Node.ChunkSigs {
			s1Map[v] = i
		}

		s2Map := make(map[string]int)
		for i, v := range s2Node.ChunkSigs {
			s2Map[v] = i
		}

		for v := range s2Map {
			_, contains := s1Map[v]
			if contains == false {
				req := Message{Version: 1, Type: "get", Sig: v}
				jsonOutput, _ := json.MarshalIndent(req, "", "    ")
				*totalSize += len(jsonOutput)
				res, err := http.Post(s2Addr, "application/json", bytes.NewReader(jsonOutput))
				body, err := ioutil.ReadAll(res.Body)
				*totalSize += len(body)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				// fmt.Println("chunk %s", v)
				*totalRequest++
				*totalChunk++
				var m1 = Message{}
				json.Unmarshal(body, &m1)
				InsertIntoDB(DBpath, v, m1.Data)
			}
		}
	} else {

		s1Map := make(map[string]int)
		for i, v := range s1Node.ChildSigs {
			s1Map[v] = i
		}

		s2Map := make(map[string]int)
		for i, v := range s2Node.ChildSigs {
			if v != "sha256_32_GDHHHHVH6CD4TQNZLCIYHUOFYZNTA67HTVHTVFPMDD4KAWF4ZADA====" {
				s2Map[v] = i
			}

		}

		///
		for v := range s2Map {
			_, contains := s1Map[v]
			if contains == false {
				i := s2Map[v]
				req := Message{Version: 1, Type: "tree", Sig: s2Node.ChildSigs[i], TreeSig: s2RootSig}
				jsonOutput, _ := json.MarshalIndent(req, "", "    ")
				*totalSize += len(jsonOutput)
				res, err := http.Post(s2Addr, "application/json", bytes.NewReader(jsonOutput))
				body, err := ioutil.ReadAll(res.Body)
				*totalSize += len(body)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				var m1 = Message{}
				json.Unmarshal(body, &m1)
				// fmt.Println(v)
				*totalRequest++
				next := m1.Node
				// (s2RootSig string, s1Node *TreeNode, s2Node *TreeNode, s2Addr string, totalSize *int, totalChunk *int, totalRequest *int, DBpath string)
				compareNode(s2RootSig, s1Node.children[i], next, s2Addr, totalSize, totalChunk, totalRequest, DBpath)
			}
		}
	}

}
func parseLocalTime(tm string) (time.Time, error) {
	loc, _ := time.LoadLocation("America/New_York")
	return time.ParseInLocation("2006-1-2T15:04", tm, loc)
}
