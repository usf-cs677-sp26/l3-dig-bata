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
	"strings"
)

// Send a file to the server for storage.
func put(msgHandler *messages.MessageHandler, fileName string) int {
	fmt.Println("PUT", fileName)

	// Get file size and make sure it exists
	info, err := os.Stat(fileName)
	if err != nil {
		log.Fatalln(err)
	}
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalln(err)
	}
	md5 := md5.New()
	if _, err := io.Copy(md5, file); err != nil {
		log.Fatalln(err)
	}
	checksum := md5.Sum(nil)
	if _, err := file.Seek(0, 0); err != nil {
		log.Fatalln(err)
	}

	// Tell the server we want to store this file
	msgHandler.SendStorageRequest(fileName, uint64(info.Size()), checksum)
	if ok, msg := msgHandler.ReceiveResponse(); !ok {
		log.Fatalln("Server does not accept storage request:", msg)
	}

	msgHandler.SetWriteDeadline(util.DeadlineSeconds(uint64(info.Size())))
	if _, err := io.CopyN(msgHandler, file, info.Size()); err != nil {
		fmt.Println("Error sending file:", err)
	}
	file.Close()

	if ok, msg := msgHandler.ReceiveResponse(); !ok {
		log.Fatalln("Server failed to store file:", msg)
	}

	fmt.Println("Storage complete!")
	return 0
}

// Get a file from the server and save it to the specified directory.
func get(msgHandler *messages.MessageHandler, fileName string, dir string) int {
	full_path := dir + "/" + fileName
	fmt.Println("GET", fileName)

	file, err := os.OpenFile(full_path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	msgHandler.SendRetrievalRequest(fileName)
	ok, msg, size, serverChecksum := msgHandler.ReceiveRetrievalResponse()
	if !ok {
		file.Close()
		os.Remove(full_path)
		log.Fatalln("Server rejected retrieval request:", msg)
	}

	md5 := md5.New()
	w := io.MultiWriter(file, md5)
	msgHandler.SetReadDeadline(util.DeadlineSeconds(size))
	if _, err := io.CopyN(w, msgHandler, int64(size)); err != nil {
		fmt.Println("Error receiving file:", err)
	}
	file.Close()

	clientChecksum := md5.Sum(nil)

	if util.VerifyChecksum(serverChecksum, clientChecksum) {
		log.Println("Successfully retrieved file.")
	} else {
		os.Remove(full_path)
		log.Println("FAILED to retrieve file. Invalid checksum.")
	}

	return 0
}

func main() {
	if len(os.Args) < 4 {
		fmt.Printf("Not enough arguments. Usage: %s server:port put|get file-name [download-dir]\n", os.Args[0])
		os.Exit(1)
	}

	host := os.Args[1]
	conn, err := net.Dial("tcp", host)
	if err != nil {
		log.Fatalln(err.Error())
	}
	msgHandler := messages.NewMessageHandler(conn)
	defer conn.Close()

	action := strings.ToLower(os.Args[2])
	if action != "put" && action != "get" {
		log.Fatalln("Invalid action", action)
	}

	fileName := os.Args[3]

	dir := "."
	if len(os.Args) >= 5 {
		dir = os.Args[4]
	}
	openDir, err := os.Open(dir)
	if err != nil {
		log.Fatalln(err)
	}
	openDir.Close()

	switch action {
	case "put":
		os.Exit(put(msgHandler, fileName))
	case "get":
		os.Exit(get(msgHandler, fileName, dir))
	}
}
