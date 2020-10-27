package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func Serve(servAddr string, sqlPath string) {
	treeMap := make(map[string]map[string]*TreeNode)
	anchorMap := make(map[string]string)
	chainMap := make(map[string][]string)

	CreateDB(sqlPath) // init DB
	http.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		var example = Message{}
		err := json.Unmarshal(body, &example)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
		}
		switch example.Type {
		case "anchor":
			anchorName := example.Name
			receiptHash := "sha256_32_" + example.RandomID
			var op map[string]interface{}
			json.Unmarshal(body, &op)
			anchorByte, _ := json.MarshalIndent(op, "", "    ")
			InsertIntoDB(sqlPath, receiptHash, anchorByte)

			anchorMap[anchorName] = receiptHash
			msg := Message{Version: 1, Type: "anchorresp", Sig: receiptHash}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(msg)

		case "updateAnchor":
			anchorName := example.Name
			anchorMap[anchorName] = example.Sig
			msg := Message{Version: 1, Type: "anchorresp", Sig: example.Sig}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(msg)

		case "claim":
			anchorName := example.Name
			rootHash, contains := anchorMap[anchorName]
			if contains {
				var op map[string]interface{}
				json.Unmarshal(body, &op)
				op["Refsig"] = rootHash
				if op["Adds"] == nil {
					op["Adds"] = make(map[string]string)
				}
				op["Adds"].(map[string]string)[example.RootsigKey] = example.RootsigValue
				previousArray, contains := chainMap[anchorName]
				if contains {
					op["Prevsig"] = previousArray[len(previousArray)-1]
				} else {
					op["Prevsig"] = rootHash
				}

				receiptHash := "sha256_32_" + example.RandomID
				claimByte, _ := json.MarshalIndent(op, "", "    ")
				InsertIntoDB(sqlPath, receiptHash, claimByte)
				chainMap[anchorName] = append(chainMap[anchorName], receiptHash)
				msg := Message{Version: 1, Type: "claimresp", Sig: receiptHash}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(msg)
			} else {
				http.Error(w, "NO ANCHOR ERROR", http.StatusInternalServerError)
			}

		case "content":
			anchorName := example.Name
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
			anchorName := example.Name
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
			anchorName := example.Name
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
			anchorName := example.Name
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

		case "put":
			data := example.Data
			sh := sha256.Sum256(data)
			receiptHash := "sha256_32_" + base32.StdEncoding.EncodeToString(sh[:])
			InsertIntoDB(sqlPath, receiptHash, data)
			msg := Message{Version: 1, Type: "putresp", Sig: receiptHash}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(msg)

		case "get":
			sig := example.Sig
			data, err := GetFromDB(sqlPath, sig)
			if err != nil {
				http.Error(w, "GET DB ERROR", http.StatusInternalServerError)
			}
			msg := Message{Version: 1, Type: "getresp", Data: data}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(msg)

		case "delete":
			treeMap = make(map[string]map[string]*TreeNode)
			sig := example.Sig
			err := DelFromDB(sqlPath, sig)
			if err != nil {
				http.Error(w, "DEL ERROR", http.StatusInternalServerError)
			}
			msg := Message{Version: 1, Type: "delresp"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(msg)

		case "getfile":
			sig := example.Sig
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
			height := example.TreeHeight
			s2 := example.TreeTarget
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
			if example.TreeSig == "" {
				height := example.TreeHeight
				root := TreeNode{}
				ConstructTree(&root, 1, height) //Construct empty tree
				treeMap[root.Sig] = FillTree(sqlPath, &root, height)
				msg := Message{Version: 1, Type: "treeresp", Node: &root}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(msg)
			} else {
				targetSig := example.TreeSig
				node := treeMap[targetSig][example.Sig]
				msg := Message{Version: 1, Type: "treeresp", Node: node}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(msg)
			}
			// case "updateReciept":
			// 	sig := example.Sig
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
