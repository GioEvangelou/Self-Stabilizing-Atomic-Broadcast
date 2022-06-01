package modules

import (
	"BFTWithoutSignatures/logger"
	"BFTWithoutSignatures/messenger"
	"BFTWithoutSignatures/types"
	"BFTWithoutSignatures/variables"
	"bytes"
	"encoding/gob"
	"log"
	"strconv"
)

var (
	// Delivered - Channel to receive delivered messages from ABC
	Delivered = make(chan struct {
		Id    int
		Value [][]byte
	})
	Aid = 1

	// Array - The array that has to be in consensus
	Array = make([]rune, 0)

	cidNum = make([]string, 0)
)

// RequestHandler - The module that handles requests received from clients and replies to them
func RequestHandler() {

	// Accepts the requests from the clients and calls ABC
	go func() {
		for message := range messenger.RequestChannel {
			var m types.ClientMessage
			buffer := bytes.NewBuffer(message)
			decoder := gob.NewDecoder(buffer)
			err := decoder.Decode(&m)
			if err != nil {
				logger.ErrLogger.Fatal(err)
			}

			id := (strconv.Itoa(m.Cid) + " " + strconv.Itoa(m.Num))
			if notStringInSlice(id, cidNum) {
				if variables.RunSSABC { // run SSABC
					SelfStabilizedAtomicBroadcast(message)
				} else { // run ABC
					AtomicBroadcast(message)
				}
			}
		}
	}()

	// Gets the delivered result from ABC, appends it in the Array and replies to the client
	go func() {
		for message := range Delivered {
			willSend := false
			for _, v := range message.Value {
				var m types.ClientMessage
				buffer := bytes.NewBuffer(v)
				decoder := gob.NewDecoder(buffer)
				err := decoder.Decode(&m)
				if err != nil {
					logger.ErrLogger.Printf(err.Error()," fault occurred")
				} else {
					//willSend = true
					id := (strconv.Itoa(m.Cid) + " " + strconv.Itoa(m.Num))
					if notStringInSlice(id, cidNum) {
						willSend = true
						cidNum = append(cidNum, id)
						Array = append(Array, m.Value)
						go func(){
							messenger.ReplyClient(types.NewReplyMessage(m.Num), m.Cid)
						}()
					}
				}
			}

			if willSend {
				//Aid = message.Id
				logger.OutLogger.Printf("%d.REQH: array-%c\n", Aid, Array)
				log.Printf("%d | %d.REQH: array (%d) - %c\n", variables.ID, Aid, len(Array), Array)
				Aid++
			}
		}
	}()
}

func notStringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return false
		}
	}
	return true
}
