# Key - Value Store with sync function

This project implement a key value store that supports: CRD operation at server plus a sync

## Client command

### put: upload a file/directories from a client to remote server;

#### usage:

`go run client/client.go put {server address} {file_location}`  
`output: receipt json`

&nbsp;

### sync: sync server b's data to server a two servers

#### usage:

`go run client/client.go sync {server1 addr} {server2 addr} {Tree Height}`  
`output: number of data passed in bytes, request and chunk pulled`

&nbsp;

### del: delete a specific sig or all from a server

#### usage:

`go run client/client.go del {server_addr} {sig}`  
`output: none`

&nbsp;

### info: get info from a server

#### usage:

`go run client/client.go info {server_addr}`  
`output: server status`

### desc: get file_reciept from a server

#### usage:

`go run client/client.go desc {server_addr} {sig}`  
`output: file reciept print on screen`

&nbsp;

### get: get a file receipt or raw bytes from server, written in bloblocal folder

#### usage:

`go run client/client.go get {server address} {sig} {new file name}`  
`output: nothing but generate new recipt in blob folder`

&nbsp;

### getfile: get a file or directory from server, written in bloblocal folder

#### usage:

`go run client/client.go getfile {server address} {sig} {new file name}`  
`output: nothing but generate new file in blob folder`
