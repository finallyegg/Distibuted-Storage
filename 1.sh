# go run serve.go -s 5555
go run client.go genkeys
mv key.private old.private
mv key.public old.public
go run client.go genkeys
go run client.go -s localhost:8888 put status.html
go run client.go -s localhost:8888 desc last
go run client.go -s localhost:8888 -q key.private sign last
go run client.go -s localhost:8888 desc last
go run client.go -s localhost:8888 -p key.public verify last
go run client.go -s localhost:8888 -p old.public verify last
go run client.go -s localhost:8888 -p key.public verify last
