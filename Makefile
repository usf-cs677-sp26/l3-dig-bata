all: bin/josh/client bin/josh/server

bin/josh/client: client/josh_client.go messages/message_handler.go util/util.go
	mkdir -p bin/josh
	go build -o bin/josh/client ./client/

bin/josh/server: server/josh_server.go messages/message_handler.go util/util.go
	mkdir -p bin/josh
	go build -o bin/josh/server ./server/

clean:
	rm -rf bin/
