go run ./client/client.go -s localhost:8888 -p public del all    
go run ./client/client.go -s localhost:8888 -p public anchor a1 
go run ./client/client.go -s localhost:8888 -p public put testfile
go run ./client/client.go -s localhost:8888 -p public claim a1 rootsig last
go run ./client/client.go -s localhost:8888 -p public chain a1 