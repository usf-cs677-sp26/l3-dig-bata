package main

import (
	"crypto/md5"
	"file-transfer/messages"
	"file-transfer/util"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"syscall"
)

func spaceAvailable(request *messages.StorageRequest) bool {

	var stat syscall.Statfs_t
	syscall.Statfs(".", &stat)
	avail := stat.Bavail * uint64(stat.Bsize)
	//fmt.Printf("diskspace: %d\n", avail)

	size := uint64(request.Size)

	if size > avail {
		return false
	}

	return true
}

func handleStorage(msgHandler *messages.MessageHandler, request *messages.StorageRequest) {
	log.Println("Attempting to store", request.FileName)

	// 2. Ensure there is enough space avail. on the disk
	if !spaceAvailable(request) {
		msgHandler.SendResponse(false, "Not enough disk space.")
		msgHandler.Close()
		return
	}

	// 1. Make sure the file doesn't already existe
	file, err := os.OpenFile(request.FileName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		msgHandler.SendResponse(false, err.Error())
		msgHandler.Close()
		return
	}

	// 3. Send an "OK" response to the client so it knows it can begin sending the file
	msgHandler.SendResponse(true, "Ready for data")

	// 4. Receive the data and store (write) the file
	md5 := md5.New()
	w := io.MultiWriter(file, md5)
	io.CopyN(w, msgHandler, int64(request.Size)) /* Write and checksum as we go */
	file.Close()

	serverCheck := md5.Sum(nil)

	// 5. Verify its checksum against the checksum sent by the client
	if util.VerifyChecksum(serverCheck, request.Checksum) {
		log.Println("Successfully stored file.")
		msgHandler.SendResponse(true, "Successfully stored file.")
	} else {
		log.Println("FAILED to store file. Invalid checksum.")
		msgHandler.SendResponse(false, "FAILED to store file. Invalid checksum.")
	}

	// 6. Respond to the client with the status of the tranfer
	// 7. Disconnect the client
	msgHandler.Close()
}

func handleRetrieval(msgHandler *messages.MessageHandler, request *messages.RetrievalRequest) {
	log.Println("Attempting to retrieve", request.FileName)

	// Get file size and make sure it exists
	info, err := os.Stat(request.FileName)
	if err != nil {
		msgHandler.SendResponse(false, err.Error())
		msgHandler.Close()
		return
	}

	file, _ := os.Open(request.FileName)
	md5 := md5.New()
	io.Copy(md5, file) // Checksum and transfer file at same time
	file.Close()

	checksum := md5.Sum(nil)

	msgHandler.SendRetrievalResponse(true, "Ready to send", uint64(info.Size()), checksum)

	file, _ = os.Open(request.FileName)
	io.CopyN(msgHandler, file, info.Size())
	file.Close()

	// Disconnect the client when the transfer is complete
	msgHandler.Close()
}

func handleClient(msgHandler *messages.MessageHandler) {
	defer msgHandler.Close()

	for {
		wrapper, err := msgHandler.Receive()
		if err != nil {
			log.Println(err)
			return
		}

		switch msg := wrapper.Msg.(type) {
		case *messages.Wrapper_StorageReq:
			handleStorage(msgHandler, msg.StorageReq)
			continue
		case *messages.Wrapper_RetrievalReq:
			handleRetrieval(msgHandler, msg.RetrievalReq)
			continue
		case nil:
			log.Println("Received an empty message, terminating client")
			return
		default:
			log.Printf("Unexpected message type: %T", msg)
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Not enough arguments. Usage: %s port [download-dir]\n", os.Args[0])
		os.Exit(1)
	}

	port := os.Args[1]
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalln(err.Error())
		os.Exit(1)
	}
	defer listener.Close()

	dir := "."
	if len(os.Args) >= 3 {
		dir = os.Args[2]
	}
	if err := os.Chdir(dir); err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Listening on port:", port)
	fmt.Println("Download directory:", dir)
	for {
		if conn, err := listener.Accept(); err == nil {
			log.Println("Accepted connection", conn.RemoteAddr())
			handler := messages.NewMessageHandler(conn)
			go handleClient(handler)
		}
	}
}
