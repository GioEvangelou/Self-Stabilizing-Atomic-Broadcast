package messenger

import (
	"BFTWithoutSignatures_Client/config"
	"BFTWithoutSignatures_Client/logger"
	"BFTWithoutSignatures_Client/types"
	"BFTWithoutSignatures_Client/variables"
	"bytes"
	"encoding/gob"
	"time"

	"github.com/pebbe/zmq4"
)

// Sockets
var (
	// Context to initialize sockets
	Context *zmq4.Context

	// ServerSockets - Send requests to servers
	ServerSockets map[int]*zmq4.Socket

	// ResponseSockets - Receive responses from servers
	ResponseSockets map[int]*zmq4.Socket
)

// Channels for messages
var (
	// RequestChannel - Channel to put the requests in
	RequestChannel = make(map[int]chan types.ClientMessage, 100)

	// ResponseChannel - Channel to put the responses in
	ResponseChannel = make(chan types.Reply)
)

// InitializeMessenger - Initializes the ZeroMQ sockets
func InitializeMessenger() {
	Context, err := zmq4.NewContext()
	if err != nil {
		logger.ErrLogger.Fatal(err)
	}

	// Initialization of a socket pair to communicate with each one of the servers
	ServerSockets = make(map[int]*zmq4.Socket, variables.N)
	ResponseSockets = make(map[int]*zmq4.Socket, variables.N)
	for i := 0; i < variables.N; i++ {

		// ServerSockets initialization to send requests
		ServerSockets[i], err = Context.NewSocket(zmq4.REQ)
		if err != nil {
			logger.ErrLogger.Fatal(err)
		}
		var serverAddr string
		if !variables.Remote {
			serverAddr = config.GetServerAddressLocal(i)
		} else {
			serverAddr = config.GetServerAddress(i)
		}
		err = ServerSockets[i].Connect(serverAddr)
		if err != nil {
			logger.ErrLogger.Fatal(err)
		}
		logger.OutLogger.Println("Request to Server", i, "on", serverAddr)

		// ResponseSockets initialization to get the response back from the servers
		ResponseSockets[i], err = Context.NewSocket(zmq4.SUB)
		if err != nil {
			logger.ErrLogger.Fatal(err)
		}
		var responseAddr string
		if !variables.Remote {
			responseAddr = config.GetResponseAddressLocal(i)
		} else {
			responseAddr = config.GetResponseAddress(i)
		}
		err = ResponseSockets[i].Connect(responseAddr)
		if err != nil {
			logger.ErrLogger.Fatal(err)
		}
		ResponseSockets[i].SetSubscribe("")
		logger.OutLogger.Println("Response from Server", i, "on", responseAddr)

		// Init request channel
		RequestChannel[i] = make(chan types.ClientMessage)
	}

	logger.OutLogger.Print("------------------------------------------------\n\n")
}

// SendRequest - Puts the messages in the request channel to be transmitted
func SendRequest(message types.ClientMessage, to int) bool {
	timeout := time.NewTicker(10 * time.Second)
	select {
	case RequestChannel[to] <- message:
		return true
	case <-timeout.C:
		<-RequestChannel[to]
		return false
	}
}

// TransmitRequests - Transmits the requests to the server [go started from main]
func TransmitRequests() {
	for i := 0; i < variables.N; i++ {
		go func(i int) { // Initializes them with a goroutine and waits forever
			for message := range RequestChannel[i] {
				w := new(bytes.Buffer)
				encoder := gob.NewEncoder(w)
				err := encoder.Encode(message)
				if err != nil {
					logger.ErrLogger.Fatal(err)
				}

				_, err = ServerSockets[i].SendBytes(w.Bytes(), 0)
				if err != nil {
					logger.ErrLogger.Fatal(err)
				}

				_, err = ServerSockets[i].Recv(0)
				if err != nil {
					logger.ErrLogger.Fatal(err)
				}
				logger.OutLogger.Printf("SENT [%d, %c] to %d", message.Num, message.Value, i)
			}
		}(i)
	}
}

// Subscribe - Handles the responses from the servers
func Subscribe() {
	for i := 0; i < variables.N; i++ {
		go func(i int) { // Initialize them with a goroutine and waits forever
			for {
				message, err := ResponseSockets[i].RecvBytes(0)
				if err != nil {
					logger.ErrLogger.Fatal(err)
				}

				go handleResponse(message)
			}
		}(i)
	}
}

// Put server's response in ResponseChannel to be handled
func handleResponse(msg []byte) {
	var message types.Reply
	buffer := bytes.NewBuffer(msg)
	decoder := gob.NewDecoder(buffer)
	err := decoder.Decode(&message)
	if err != nil {
		logger.ErrLogger.Fatal(err)
	}

	logger.OutLogger.Println("RECEIVED REP from", message.From, "for", message.Value)
	ResponseChannel <- message
}
