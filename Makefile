all: bin/client bin/server bin/josh/client bin/david/client bin/josh/server bin/david/server

bin/client: client/client.go messages/message_handler.go util/util.go
	go build -o bin/client client/client.go

bin/server: server/server.go messages/message_handler.go util/util.go
	go build -o bin/server server/server.go

bin/josh/client: client/josh_client.go messages/message_handler.go util/util.go
	go build -o bin/josh/client client/josh_client.go

bin/josh/server: server/josh_server.go messages/message_handler.go util/util.go
	go build -o bin/josh/server server/josh_server.go

bin/david/client: client/david_client.go messages/message_handler.go util/util.go
	go build -o bin/david/client client/david_client.go

bin/david/server: server/david_server.go messages/message_handler.go util/util.go
	go build -o bin/david/server server/david_server.go

clean:
	rm -rf bin/{client,server}
