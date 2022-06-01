package app

import (
	"BFTWithoutSignatures_Client/logger"
	"BFTWithoutSignatures_Client/messenger"
	"BFTWithoutSignatures_Client/types"
	"BFTWithoutSignatures_Client/variables"
	"math/rand"
	"time"
)

var (
	runes = []rune("!\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~")

	replies  = make(map[int]map[int]bool) // num, from
	accepted = make(map[int]bool)         // if this num is accepted

	// Client metrics regarding the experiment evaluation
	sentTime  = make(map[int]time.Time)
	OpLatency = time.Duration(0)
	num       = 0
	Total     = 0
)

func Client() {
	rand.Seed(int64((variables.ID + 3) * 9000))              // Pseudo-Random Generator
	time.Sleep(time.Duration(variables.ID%10) * time.Second) // Wait a bit before sending 1st request

	sendRune()

	go func() {
		for message := range messenger.ResponseChannel {
			if _, in := replies[message.Value][message.From]; in {
				continue // Only one value can be received from each server
			}
			if replies[message.Value] == nil {
				replies[message.Value] = make(map[int]bool)
			}
			replies[message.Value][message.From] = true

			// If more than F+1 with the same value, accept the array.
			if len(replies[message.Value]) >= (variables.F+1) && !accepted[message.Value] {
				accepted[message.Value] = true
				OpLatency += time.Since(sentTime[message.Value])
				Total++
				logger.OutLogger.Print("RECEIVED ACK for ", message.Value, " [",
					time.Since(sentTime[message.Value]), "]\n")

				if num < 2 {
					sendRune()
				}
			}
		}
	}()
}

func sendRune() {
	num++
	message := types.NewClientMessage(variables.ID, num, runes[rand.Intn(len(runes))])
	randServer := rand.Intn(variables.N)

	for i := 0; i < (variables.F + 1); i++ {
		to := (randServer + i) % variables.N
		flag := messenger.SendRequest(message, to)
		if !flag {
			randServer = rand.Intn(variables.N)
			i--
		}
	}

	sentTime[num] = time.Now()
}
