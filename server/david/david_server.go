package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"file-transfer/messages"
	"file-transfer/util"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"syscall"
)

const metadata_filename = ".metadata"

func saveChecksum(fileName string, checksum []byte) error {
	f, err := os.OpenFile(metadata_filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	if _, err = fmt.Fprintf(f, "%s %x\n", fileName, checksum); err != nil {
		f.Close()
		return err
	}
	f.Close()
	return nil
}

// Save a file sent by the client to disk.
func handleStorage(msgHandler *messages.MessageHandler, request *messages.StorageRequest) {
	log.Println("Attempting to store", request.FileName)

	var stat syscall.Statfs_t
	if err := syscall.Statfs(".", &stat); err != nil {
		msgHandler.SendResponse(false, "Failed to check disk space: "+err.Error())
		return
	}
	available := stat.Bavail * uint64(stat.Bsize)
	if available < request.Size {
		msgHandler.SendResponse(false, "Insufficient disk space")
		return
	}
	file, err := os.OpenFile(request.FileName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		msgHandler.SendResponse(false, err.Error())
		msgHandler.Close()
		return
	}

	msgHandler.SetReadDeadline(util.DeadlineSeconds(request.Size))
	msgHandler.SendResponse(true, "Server is ready to receive data")
	md5 := md5.New()
	w := io.MultiWriter(file, md5)
	if _, err := io.CopyN(w, msgHandler, int64(request.Size)); err != nil {
		fmt.Println("Error receiving file:", err)
	}
	file.Close()

	serverChecksum := md5.Sum(nil)
	clientChecksum := request.Checksum

	if util.VerifyChecksum(serverChecksum, clientChecksum) {
		msgHandler.SendResponse(true, "File stored successfully")
		log.Printf("Successfully stored '%s'\n", request.FileName)
		if err := saveChecksum(request.FileName, serverChecksum); err != nil {
			log.Println("Error saving checksum:", err)
		}
	} else {
		msgHandler.SendResponse(false, "Checksum verification failed")
		log.Printf("FAILED to store '%s'. Invalid checksum.", request.FileName)
		os.Remove(request.FileName)
	}
}

func getChecksum(fileName string) ([]byte, error) {
	f, err := os.Open(metadata_filename)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var name string
		var checksum string
		if _, err := fmt.Sscanf(scanner.Text(), "%s %s", &name, &checksum); err != nil {
			f.Close()
			return nil, err
		}
		if name == fileName {
			f.Close()
			checksumBytes, err := hex.DecodeString(checksum)
			if err != nil {
				return nil, err
			}
			return checksumBytes, nil
		}
	}

	f.Close()
	return nil, fmt.Errorf("Checksum not found for file: %s", fileName)
}

func handleRetrieval(msgHandler *messages.MessageHandler, request *messages.RetrievalRequest) {
	log.Println("Attempting to retrieve", request.FileName)

	// Get file size and make sure it exists
	info, err := os.Stat(request.FileName)
	if err != nil {
		msgHandler.SendRetrievalResponse(false, err.Error(), 0, nil)
		return
	}

	checksum, err := getChecksum(request.FileName)
	if err != nil {
		msgHandler.SendRetrievalResponse(false, err.Error(), 0, nil)
		file, err := os.Open(request.FileName)
		if err != nil {
			fmt.Println("Error opening file to save checksum:", err)
		}
		md5 := md5.New()
		if _, err := io.Copy(md5, file); err != nil {
			fmt.Println("Error reading file to save checksum:", err)
		}
		file.Close()
		checksum = md5.Sum(nil)
		if err := saveChecksum(request.FileName, checksum); err != nil {
			fmt.Println("Error saving checksum:", err)
		}
	}

	file, err := os.Open(request.FileName)
	if err != nil {
		msgHandler.SendRetrievalResponse(false, err.Error(), 0, nil)
		return
	}

	msgHandler.SetWriteDeadline(util.DeadlineSeconds(uint64(info.Size())))
	msgHandler.SendRetrievalResponse(true, "Server is ready to send", uint64(info.Size()), checksum)
	if _, err := io.CopyN(msgHandler, file, info.Size()); err != nil {
		fmt.Println("Error sending file:", err)
	}
	file.Close()
}

func handleClient(msgHandler *messages.MessageHandler) {
	defer msgHandler.Close()

	for {
		wrapper, err := msgHandler.Receive()
		if err != nil {
			if err != io.EOF {
				log.Println("Error receiving message:", err)
			}
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
			log.Printf("Unexpected message type: %T\n", msg)
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
