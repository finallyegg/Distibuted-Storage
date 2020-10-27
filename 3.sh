# go run serve.go -s 5555 -p key.public
go run client.go -s localhost:8888 del all
go run client.go -s localhost:8888 -q key.private anchor peteroot
go run client.go -s localhost:8888 desc last
go run client.go -s localhost:8888 put status.html
go run client.go -s localhost:8888 -q old.private claim peteroot rootsig last
go run client.go -s localhost:8888 put status.html
go run client.go -s localhost:8888 -q key.private claim peteroot rootsig last
go run client.go -s localhost:8888 desc last
go run client.go -s localhost:8888 -q key.private anchor keleher
# go run client.go -s localhost:8888 -q keleher.private anchor keleher
