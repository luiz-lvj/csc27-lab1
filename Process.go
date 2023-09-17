package main

import (
	"fmt"
	"net"
	"os"
	"bufio"
	"strconv"
	"encoding/json"
	"time"
	"sync"
)

type Message struct {
	Message string `json:"message"`
	Sender int `json:"sender"`
	Clock int `json:"clock"`
}


var serverConnection *net.UDPConn
var connections []*net.UDPConn
var processId int
var internalClock int
var port string
var numberProcesses int
var host string
var queueRequests []int
var queueReplies []int
var muted sync.Mutex
var state string


func listenInput(userInput chan string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _, _ := reader.ReadLine()
		userInput <- string(text)
		fmt.Println("Input: ", string(text))
	}
}

func useCS(){
	fmt.Println("Entrei na CS...")
	state = "HELD"
	msg := Message{
		Message: "release",
		Sender: processId,
		Clock: internalClock,
	}
	fmt.Println("Message", msg)
	jsonMsg, err := json.Marshal(msg)
	byteMsg := []byte(jsonMsg)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(0)
	}

	//send release message to server as json stringified
	_, err = serverConnection.Write(byteMsg)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(0)
	}

	time.Sleep(time.Second * 20)
}

func sendMsg(message string, clock int, id int){
	
	msg := Message{
		Message: message,
		Sender: processId,
		Clock: clock,
	}
	
	jsonMsg, err := json.Marshal(msg)
	byteMsg := []byte(jsonMsg)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(0)
	}

	connections[id-1].Write(byteMsg)
}

func multicast(message string, clock int){

	for i := 1; i <= numberProcesses; i++ {
		if i != processId {
			sendMsg(message, clock, i)
		}
	}
}

func ricartAgrawala(){
	state = "WANTED"
	muted.Lock()
	internalClock++
	muted.Unlock()
	logicalClock:= internalClock
	multicast("request", logicalClock)

	for {
		muted.Lock()
		if len(queueReplies) == numberProcesses - 1 {
			fmt.Println("Received all replies.")
			break
		}
		muted.Unlock()
	}
	muted.Unlock()
	useCS()
	state = "HELD"
	muted.Lock()
	fmt.Println("Sai da CS...")
	for i := 0; i < len(queueRequests); i++ {
		fmt.Println("Sending reply to ", queueRequests[i])
		sendMsg("reply", logicalClock, queueRequests[i])
	}
	muted.Unlock()
	
	state = "RELEASED"
}

func doServerJob(){
	for {
		buffer := make([]byte, 1024)
		n, _, err := connections[processId -1].ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error: ", err)
			os.Exit(0)
		}

		var msg Message
		json.Unmarshal(buffer[:n], &msg)

		if msg.Sender == processId {
			fmt.Println("Same process.")
			continue
		}

		if msg.Message == "request" {
			priotity := internalClock < msg.Clock || (internalClock == msg.Clock && processId < msg.Sender)
			if state == "HELD" || (state == "WANTED" && priotity) {
				muted.Lock()

				queueRequests = append(queueRequests, msg.Sender)
				if internalClock < msg.Clock {
					internalClock = msg.Clock +1
				} else {
					internalClock++
				}
				muted.Unlock()
			} else {
				muted.Lock()
				fmt.Println("Sending reply to ", msg.Sender)
				if internalClock < msg.Clock {
					internalClock = msg.Clock +1
				} else {
					internalClock++
				}
				replyClock := internalClock
				sendMsg("reply", replyClock, msg.Sender)
				muted.Unlock()
			}
		} else if msg.Message == "reply" {
			muted.Lock()
			if internalClock < msg.Clock {
				internalClock = msg.Clock +1
			} else {
				internalClock++
			}
			queueReplies = append(queueReplies, msg.Sender)
			muted.Unlock()
		}
		
	}
}


func main(){
	serverPort := "10001"
	processId, _ = strconv.Atoi(os.Args[1])
	fmt.Println("Process ID: ", os.Args[1])
	port = os.Args[processId+1]
	numberProcesses = len(os.Args) - 2
	connections = make([]*net.UDPConn, numberProcesses)
	fmt.Println("Number of processes: ", numberProcesses)
	host = "127.0.0.1"

	state = "RELEASED"


	for i := 1; i <= numberProcesses; i++ {
		addr, err := net.ResolveUDPAddr("udp", host + os.Args[i+1])
		if(err != nil){
			fmt.Println("Error: ", err)
			os.Exit(0)
		}
		
		if i != processId {
			connections[i-1], err = net.DialUDP("udp", nil, addr)
			if(err != nil){
				fmt.Println("Error: ", err)
				os.Exit(0)
			}
			
		} else {
			connections[i-1], err = net.ListenUDP("udp", addr)
			if(err != nil){
				fmt.Println("Error: ", err)
				os.Exit(0)
			}
		}
	}

	

	serverAddr, err := net.ResolveUDPAddr("udp", host + ":" + serverPort)
	fmt.Println("server addr: ", serverAddr)
	if(err != nil){
		fmt.Println("Error: ", err)
		os.Exit(0)
	}

	serverConnection, err = net.DialUDP("udp", nil, serverAddr)
	if(err != nil){
		fmt.Println("Error: ", err)
		os.Exit(0)
	}

	defer serverConnection.Close()
	for i := 0; i < numberProcesses; i++ {
		defer connections[i].Close()
	}


	userInput := make(chan string)
	internalClock = 0
	
	go listenInput(userInput)
	go doServerJob()

	for {
		select {
			case input, valid := <- userInput:
				if valid {
					if input == "x"{
						if state == "HELD" || state == "WANTED" {
							fmt.Println("x ignorado\n")
						} else {
							muted.Lock()
							internalClock++
							muted.Unlock()

							go ricartAgrawala()
						}

					}
				} else if input == strconv.Itoa(processId) {
					muted.Lock()
					internalClock++
					muted.Unlock()
					fmt.Println("Updating internal clock...\nInternal Clock: ", internalClock)
				} else{
					fmt.Println("Invalid input.")
				}
			default:
				time.Sleep(time.Second * 1)
		}
		time.Sleep(time.Second * 1)

	}
	
	
}