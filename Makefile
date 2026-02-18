all: bin/josh/client bin/josh/server bin/david/client bin/david/server

bin/josh/client: client/josh_client.go messages/message_handler.go util/util.go
	mkdir -p bin/josh
	go build -o bin/josh/client ./client/

bin/josh/server: server/josh_server.go messages/message_handler.go util/util.go
	mkdir -p bin/josh
	go build -o bin/josh/server ./server/

bin/david/client: client/david/david_client.go messages/message_handler.go util/util.go
	go build -o bin/david/client client/david/david_client.go

bin/david/server: server/david/david_server.go messages/message_handler.go util/util.go
	go build -o bin/david/server server/david/david_server.go

clean:
	rm -rf bin/
