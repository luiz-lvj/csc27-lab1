package main

import (
	"fmt"
	"net"
	"os"
	"encoding/json"
)

func CheckError(err error) {
	if err != nil {
		fmt.Println("Err: ", err)
		os.Exit(0)
	}
}

type Message struct {
	Message string `json:"message"`
	Sender int `json:"sender"`
	Clock int `json:"clock"`
}

func main(){
	
	var port string = ":10001" 
	bufferSize := 1024
	Address, err := net.ResolveUDPAddr("udp", port)
	CheckError(err)

	Connection, err := net.ListenUDP("udp", Address)
	CheckError(err)

	defer Connection.Close()

	buffer := make([]byte, bufferSize)
	fmt.Println("Listening on port ", port)

	for {

		fmt.Println("Waiting for message...")
		n, _, err := Connection.ReadFromUDP(buffer)
		CheckError(err)
		var msg Message

		err = json.Unmarshal([]byte(buffer[:n]), &msg)
		if(err != nil){
			fmt.Println("Error: ", err)
			os.Exit(0)
		}

		// the received message is a ll

		fmt.Println("Message Received: ", msg)
		fmt.Println("Sender: ", msg.Sender)

	}
}