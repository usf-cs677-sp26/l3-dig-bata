package util

import (
	"log"
	"reflect"
)

const MinAcceptedBps = 1048576 // 1 MiB/s

func VerifyChecksum(serverCheck []byte, clientCheck []byte) bool {
	log.Printf("Server checksum: %x\n", serverCheck)
	log.Printf("Client checksum: %x\n", clientCheck)
	if reflect.DeepEqual(clientCheck, serverCheck) {
		log.Println("Checksums match")
		return true
	} else {
		log.Println("Checksums DO NOT match")
		return false
	}
}

func DeadlineSeconds(size uint64) uint64 {
	timeoutSeconds := max(size / MinAcceptedBps, 1)
	return timeoutSeconds
}
