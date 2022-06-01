package modules

import (
	"BFTWithoutSignatures/logger"
	"BFTWithoutSignatures/messenger"
	"BFTWithoutSignatures/types"
	"BFTWithoutSignatures/variables"
	"bytes"
	"encoding/gob"
	"encoding/json"
	"math"
	"strconv"
	"time"

	"BFTWithoutSignatures/faults"
	"BFTWithoutSignatures/config"
	"fmt"
)

type MT = types.SSVCMessageTuple

type ssvcmsg struct {
	SSVCMessage types.SSVCMessage
	From      int
}


// SelfStabilizedVectorConsensus - The method that is called to initiate the SSVC module
func SelfStabilizedVectorConsensus(ssvcid int, initVal []byte) {

	// START Variables initialization
	getValue := MT{Sender: variables.ID, Value:initVal}
	msg := make(map[int]map[string][]MT)	// received messages
	for i:=0; i < variables.N; i++ {	// initialize maps of messages for every process
		msg = flushProcessMsg(msg, i)
	}
	msg[variables.ID]["init"] = append(msg[variables.ID]["init"],
																				MT{Sender: variables.ID,Value: initVal})
	round := 0	// to run non SS MVC for tests

	messenger.SSVCMutex.Lock()
	if _, in := messenger.SSVCChannel[ssvcid]; !in {
		messenger.SSVCChannel[ssvcid] = make(chan struct {
			SSVCMessage types.SSVCMessage
			From      int
		})
	}
	msgChannel := messenger.SSVCChannel[ssvcid]
	messenger.SSVCMutex.Unlock()

	// channels to coordinate algorithm execution and message receipt
	startReadingMessages := make(chan bool)
	pauseReadingMessages := make(chan bool)
	quitReadingMessages := make(chan bool, 1)
	// processes' decisions - for testing purposes
	ssvcDecisions := make(map[int]map[int][]byte)

	messenger.SSVCDecisionsMutex.Lock()
	if _, in := messenger.SSVCDecisionsChannel[ssvcid]; !in {
		messenger.SSVCDecisionsChannel[ssvcid] = make(chan struct {
			Vector map[int][]byte
			From      int
		})
	}
	decisionsChannel := messenger.SSVCDecisionsChannel[ssvcid]
	messenger.SSVCDecisionsMutex.Unlock()

	// incoming messages
	messageQueue := []ssvcmsg{}

	/****************************************************************************/
	// to run tests with transient faults - not part of the algorithm
	transientFaults := config.Transient && config.TestExecution
	/****************************************************************************/
	// END Variables initialization

	/* ---------------------------Execute Algorithm----------------------------- */
	go func (){
		for {

			// handle new received messages
			pauseReadingMessages <- true
			msg = handleNewSSVCMessages(messageQueue, msg)
			messageQueue = []ssvcmsg{}
			startReadingMessages <- true

			// check if at least F + 1 processes have decided the same value
			decision_loop:
			for {
				select{
				case dmessage := <- decisionsChannel:
					ssvcDecisions[dmessage.From] = dmessage.Vector
					if len(ssvcDecisions) > variables.F {
						for _, d := range ssvcDecisions {
							occurence:=0
							for _, v := range ssvcDecisions {
								if areVectorsEqual(d, v) {
									occurence++
									if occurence > variables.F {	// decide this value
										vect := make(map[int][]byte)
										for k,v := range d{
											vect[k] = make([]byte, len(v))
											copy(vect[k], v)
										}

										logger.OutLogger.Print("SSVC: decide-", vect, "\n")
										SSVCAnswer[ssvcid] <- vect
										quitReadingMessages <- true
										broadcastSSVCDecision(vect, ssvcid)

										return
									}
								}
							}
						}
					}
				default: // no message to get
					break decision_loop
				}
			}

			// Consistency checks
			if sendersCorrect,_ := areSendersCorrect(msg, variables.ID); !sendersCorrect ||
				!isReadyConsistent(msg, variables.ID) || !isEchoConsistent(msg, variables.ID) {
				msg = flushProcessMsg(msg, variables.ID) // empty every message type set
			}

			if _, in := containsMessage(msg[variables.ID]["init"], getValue); !in {
				msg[variables.ID]["init"] = append(msg[variables.ID]["init"], getValue)
			}

			for i:=0; i < variables.N; i++ {	// for every process
				correctSenders, incorrectTypes := areSendersCorrect(msg,i)
				conflict, conflictTypes := messageConflictExists (msg[i])
				if !isInitCorrect(msg[i]["init"], i) {	// empty messages by i
					msg = flushProcessMsg(msg, i)
				}
				if !correctSenders || conflict {	// flush certain types only
					flushTypes := append(incorrectTypes, conflictTypes...)

					for _, ftype := range flushTypes {
						msg[i][ftype] = []MT{}
					}
				}

				// send echo message for every init
				for _, initm := range msg[i]["init"] {
					if _, in := containsMessage(msg[variables.ID]["echo"], initm); !in {	// only 1 echo per init
						msg[variables.ID]["echo"] = append(msg[variables.ID]["echo"], initm)
					}
				}
			}

			// send ready message based on echo and ready messages
			// calculate echo/ready with enough support
			echoMsgs := valueThresholdOccurence(msg, int(math.Ceil(float64(variables.N + variables.F)/2)), "echo")
			readyMsgs := valueThresholdOccurence(msg, variables.F + 1, "ready")

			for _,  m := range echoMsgs {
				if _, in := containsMessage(msg[variables.ID]["ready"], m); !in { // only 1 ready per init
					msg[variables.ID]["ready"] = append(msg[variables.ID]["ready"], m)
				}
			}
			for _, m := range readyMsgs {
				if _, in := containsMessage(msg[variables.ID]["ready"], m); !in { // only 1 ready per init
					msg[variables.ID]["ready"] = append(msg[variables.ID]["ready"], m)
				}
			}

			/************************************************************************/
			// TRANSIENT TESTS - NOT PART OF THE ALGORITHM (can be removed)
			if transientFaults && (len(msg[variables.ID]["ready"])>0) {
				transientFaults=false

				// createTransientMsg(initFault, echoFault, readyFault, senderFault,valueFault
				msg = faults.CreateSSVCTransientMsg(true, true, true, true,true, msg,
					 ssvcid, config.TransientProbability)

			}
			/************************************************************************/

			broadcastSSVCMessage(types.SSVCMessage{SSVCid: ssvcid, Content: msg[variables.ID]})

			// Built the vector with the values received
			vector := make(map[int][]byte, variables.N)
			// initialize vector with DEFAULT values
			for i := 0; i < variables.N; i++ {
				vector[i] = variables.DEFAULT
			}

			// calculate ready with enough support for RB_deliver
			readyMsgs = valueThresholdOccurence(msg, 2*variables.F + 1, "ready")

			// fill vector with 2f+1 supported ready messages in sender's position
			for _, m := range readyMsgs {
				vector[m.Sender] = m.Value
			}

			if isVectorPopulated(vector) {	// n-f non default entries

				// Compute the MVC identifier and convert vector to bytes
				mvcid := ComputeUniqueIdentifier(ssvcid, round)
				round++	// for debugging purposes

				// Convert vector to bytes
				MVCAnswer[mvcid] = make(chan []byte, 1)
				w, err := json.Marshal(vector)
				if err != nil {
					logger.ErrLogger.Fatal(err)
				}

				logger.OutLogger.Print("SSVC: deliver len-", len(readyMsgs), " vector-", vector, " --> MVC\n")

				go MultiValuedConsensus(mvcid, w)
				quit := make(chan bool)
				go func(){
					for {
						select{
						case <- quit:
							return
						default:
							time.Sleep(time.Second/5)
							broadcastSSVCMessage(types.SSVCMessage{SSVCid: ssvcid, Content: msg[variables.ID]})
						}
					}
				}()
				v := <-MVCAnswer[mvcid]
				quit <- true

				//var vect map[int][]byte
				vect := make(map[int][]byte)

				if bytes.Equal(v, variables.PSI){	// transient error
					for i := 0; i < variables.N; i++ {
						vect[i] = variables.PSI
					}
				} else if bytes.Equal(v, variables.DEFAULT) || len(msg[variables.ID]["init"])==0{
					for i := 0; i < variables.N; i++ {
						vect[i] = variables.DEFAULT
					}
				} else {
					err = json.Unmarshal(v, &vect)
					if err != nil {
						logger.ErrLogger.Fatal(err)
					}
				}

				logger.OutLogger.Print("SSVC: decide-", vect, "\n")
				SSVCAnswer[ssvcid] <- vect

				quitReadingMessages <- true
				broadcastSSVCDecision(vect, ssvcid)

				return
			}
		}
	}()

	/* ---------------------------Receive Messages----------------------------- */
	go func(){
		for {
			select{	// check if must quit receiving
			case <- quitReadingMessages:
				return
			default:
				select{	// pause receiving
				case <- pauseReadingMessages:
					<- startReadingMessages	// wait for read signal
				default:
					select{	// read message
					case message := <- msgChannel:

						if !isSSVCMessageInQueue(messageQueue, message){
							messageQueue = append(messageQueue, message)
						}
					default:
					}
				}
			}
		}
	}()
}


func printmsg(msg map[int]map[string][]MT){
	for p := range msg{
		fmt.Println(p)
		for _,t := range []string{"init", "echo", "ready"} {
			fmt.Println(t)
			for _, m := range msg[p][t]{
				fmt.Print("("+strconv.Itoa(m.Sender)+" "+string(m.Value[:])+"), ")
			}
			fmt.Println()
		}
	}
}

// Check if 2 decided vectors are equal
func areVectorsEqual (a,b map[int][]byte) bool {
	if len(a) != len(b) {return false}
	for ak, av := range a {
		if bv, in := b[ak]; !in || !bytes.Equal(av, bv) {return false}
	}
	return true
}

// Perform the required checks on new SSVC messages
func handleNewSSVCMessages(messageQueue []ssvcmsg, msg map[int]map[string][]MT) map[int]map[string][]MT{

	for i:=0; i < len(messageQueue); i++{
		message := messageQueue[i]
		sender := message.From
		var copied int
		pmsg := map[string][]MT{}	// sender's msg copy

		for _, mtype := range []string{"echo", "ready"} {
			// copy echo/ready
			pmsg[mtype] = make([]MT, len(msg[sender][mtype]))
			copied = copy(pmsg[mtype], msg[sender][mtype])
			if copied != len(msg[sender][mtype]) {
				fmt.Println("copied "+strconv.Itoa(copied)+" len "+strconv.Itoa(len(msg[sender][mtype])))
				panic("Copied less "+mtype+" elements")
			}
		}

		for _, mtype := range []string{"echo", "ready"} {
			// add new echo and ready messages
			for _, m := range message.SSVCMessage.Content[mtype] {
				if _, in := containsMessage(msg[sender][mtype], m); !in { // add message only 1 time
					msg[sender][mtype] = append(msg[sender][mtype], m)
					pmsg[mtype] = append(pmsg[mtype], m)
				}
			}
		}

		// add init if no conflicting echo/ready values
		if conflict, _ := messageConflictExists (pmsg); !conflict {
			for _, m := range message.SSVCMessage.Content["init"] {
				if _, in := containsMessage(msg[sender]["init"], m); !in { // add init only 1 time
					msg[sender]["init"] = append(msg[sender]["init"], m)
				}
			}
		}
	}
	return msg
}

// Check if received message already exists in message queue
func isSSVCMessageInQueue(messageQueue []ssvcmsg, newmsg ssvcmsg) bool {

	for _, msg := range messageQueue{
		if msg.From == newmsg.From { // for every message in queue
			m := msg.SSVCMessage.Content
			equalMessages := true
			for i:=0; i < 3 && equalMessages; i++{ // for every type
				mtype := []string{"init", "echo", "ready"}[i]
				newm := newmsg.SSVCMessage.Content[mtype]
				if len(m[mtype]) == len(newm) {
					for _, nm := range newm { // every msg in type
						if _, in := containsMessage(m[mtype], nm); !in {
							equalMessages = false
							break
						}
					}
				} else {
					equalMessages = false
					break
				}
			}
			if equalMessages {return equalMessages}	// found an identical message
		}
	}
	return false
}

// Check if a message is contained in slice
// Return true, the position of its appearance in the slice, otherwise false,-1
func containsMessage(messages []MT, m MT) (int, bool) {
	for i, t := range messages {
		if t.Sender == m.Sender && bytes.Equal(m.Value, t.Value) {return i, true}
	}
	return -1, false
}

// Flush (Initialize) message (msg) maps of process p if not initialized
func flushProcessMsg(msg map[int]map[string][]MT, p int) map[int]map[string][]MT {
	if _, in := msg[p]; !in {
		msg[p] = map[string][]MT{}
	}
	msg[p]["init"] = []MT{}
	msg[p]["echo"] = []MT{}
	msg[p]["ready"] = []MT{}
	return msg
}

// Checks if sender IDs of messages are correct (between [0,N))
func areSendersCorrect(msg map[int]map[string][]MT, p int) (bool, []string) {
	correct := true
	incorrectTypes := []string{}

	for _, mtype := range []string{"init","echo","ready"} {
		for _, m := range msg[p][mtype] {
			if m.Sender < 0 || m.Sender >= variables.N {
				incorrectTypes = append(incorrectTypes, mtype)
				correct = false
				break
			}
		}
	}

	return correct, incorrectTypes
}

// Checks if for every echo message of process p, an init message exists
// Only for msg[variables.ID] entries
func isEchoConsistent (msg map[int]map[string][]MT, p int) bool {
	pEchoMsgs := msg[p]["echo"]	// echo messages of process p

	for _, m := range pEchoMsgs {
		if _, sin := msg[m.Sender]; sin {	// check existance of correct sender
			if _, in := containsMessage(msg[m.Sender]["init"], m); !in { // corresponding init check
				return false
			}
		} else { return false }
	}
	return true
}

// Checks if ready messages of process p are consistent (either f+1 have seen a
// ready message or (n+f)/2 have seen an echo one)
// Only for msg[variables.ID] entries
func isReadyConsistent (msg map[int]map[string][]MT, p int) bool {
	pReadyMsgs := msg[p]["ready"]	// ready messages of process p

	echoMsgs := valueThresholdOccurence(msg, int(math.Ceil(float64(variables.N + variables.F)/2)), "echo")
	readyMsgs := valueThresholdOccurence(msg, variables.F + 1, "ready")

	for _, m := range pReadyMsgs {
		_, inThresholdEcho := containsMessage(echoMsgs, m)
		_, inThresholdReady := containsMessage(readyMsgs, m)
		if !(inThresholdEcho || inThresholdReady) {
			return false
		}
	}

	return true
}

// Checks if at most 1 init message exists and it is correctly formatted
func isInitCorrect (messages []MT, id int) bool {
	if len(messages) == 0 {return true}
	if len(messages) == 1 {
			return messages[0].Sender == id
	}
	return false
}

// Checks if conflicting echo/ready messages have been sent by a process
// (2 messages from a sender that have different values)
func messageConflictExists (messages map[string][]MT) (bool, []string) {
	messageTypes := [2]string{"echo", "ready"}

	conflictTypes := []string{}
	conflict := false
	for _, mtype := range messageTypes {
		senders := make(map[int]bool)	// type messages senders
		for _, message := range messages[mtype] {
			sender := message.Sender
			// []MT is built as a set (no duplicates)
			// if sender already exists then 2 messages with different values but
			// identical sender exist -> confict
			if _, in := senders[sender]; in {
				conflict = true
				conflictTypes = append(conflictTypes, mtype)
				break
			} else {
				senders[sender] = true
			}
		}
	}
	return conflict, conflictTypes
}

// Checks if vector created contains at least n-f non DEFAULT values
func isVectorPopulated (vector map[int][]byte) bool{
	nonDefaultCounter := 0
	for i := 0; i < variables.N; i++ {
		if !bytes.Equal(vector[i], variables.DEFAULT) {
			nonDefaultCounter++
		}
	}
	return nonDefaultCounter >= variables.N - variables.F
}

// Returns every value v that has been send as a messageType message from
// >=s processes (at least s)
// messageType must be either echo or ready
func valueThresholdOccurence (msg map[int]map[string][]MT, s int,
	messageType string) []MT {
		if !(messageType == "echo" || messageType == "ready") {
			panic("Module self_stabilized_vector_consensus Function"+
				"valueThresholdOccurence: messageType must be either echo or ready")
		}

		values := []MT{}	// values appearing >=s times (with their sender)
		messages := []MT{}	// every message of type messageType sent
		messageCounts := []int{}	// counter for every message in messages

		//	count the occurence of each message
		for process := range msg {
			for _, message := range msg[process][messageType] {
				if index, in := containsMessage(messages, message); in {
					messageCounts[index]++
				} else {
					messages = append(messages, message)
					messageCounts = append(messageCounts, 1)
				}
			}
		}

		//	return the messages which occur at least s times
		for i := 0; i < len(messages); i++ {
			if messageCounts[i] >= s {
				values = append(values, messages[i])
			}
		}

		return values
}

// Send every message from myself to every process
func broadcastSSVCMessage(ssvcm types.SSVCMessage) {
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	err := encoder.Encode(ssvcm)
	if err != nil {
		logger.ErrLogger.Fatal(err)
	}

	message := types.NewMessage(w.Bytes(), "SSVC")
	messenger.Broadcast(message)
}

// Send every message from myself to every process
func broadcastSSVCDecision(vector map[int][]byte, ssvcid int) {
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	err := encoder.Encode(struct{
		Vector map[int][]byte
		SSVCid int}{Vector: vector, SSVCid: ssvcid})
	if err != nil {
		logger.ErrLogger.Fatal(err)
	}

	message := types.NewMessage(w.Bytes(), "SSVCDS")
	messenger.Broadcast(message)
}
