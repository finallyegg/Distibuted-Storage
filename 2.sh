go run client.go -s localhost:8888 del all
go run client.go -s localhost:8888 anchor peteroot
go run client.go -s localhost:8888 put status.html 
go run client.go -s localhost:8888 claim peteroot rootsig last
go run client.go -s localhost:8888 content peteroot
go run client.go -s localhost:8888 get last x
cat x
go run client.go -s localhost:8888 put status2.html
go run client.go -s localhost:8888 claim peteroot rootsig last
go run client.go -s localhost:8888 content peteroot
go run client.go -s localhost:8888 get last x
cat x
go run client.go -s localhost:8888 rootanchor peteroot
go run client.go -s localhost:8888 chain peteroot
