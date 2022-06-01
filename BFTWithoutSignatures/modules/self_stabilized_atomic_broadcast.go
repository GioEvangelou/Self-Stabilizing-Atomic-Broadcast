package modules

import (
	"BFTWithoutSignatures/logger"
	"BFTWithoutSignatures/messenger"
	"BFTWithoutSignatures/types"
	"BFTWithoutSignatures/variables"
	"bytes"
	"encoding/gob"
	"encoding/json"
  "sort"
	"math"
	"strconv"
	"time"

	"fmt"
	"BFTWithoutSignatures/faults"
	"BFTWithoutSignatures/config"
)

type ssabcMT = types.SSABCMessageTuple
type ssabcmsg struct {
	SSABCMessage types.SSABCMessage
	From      int
}
type ssabcMessageID struct{
	Sender int
	Num uint32
}

var (
  // SSVCAnswer - Channel to receive the answer from SSVC
	SSVCAnswer map[int]chan map[int][]byte
	getValue ssabcMT
	readRequest, handleNewRequest chan bool
	ssnum uint32
)

// InitiateAtomicBroadcast - The method that is called to initiate the ABC module
func InitiateSelfStabilizedAtomicBroadcast() {
	// init channels and getValue
	readRequest = make(chan bool)
	handleNewRequest = make(chan bool)
	ssnum = 0
	getValue = ssabcMT{Sender: -1, Num: math.MaxUint32, Value: variables.DEFAULT}
	SSVCAnswer = make(map[int]chan map[int][]byte)

	go ssabcAlgorithm()
}

// SelfStabilizedAtomicBroadcast - The method that is called to broadcast a new SSABC value
func SelfStabilizedAtomicBroadcast(m []byte){
	<- handleNewRequest
	getValue = ssabcMT{Sender: variables.ID, Num: ssnum, Value: m}	// add new request
	ssnum++
	readRequest <- true
}

// Execution of the SSABC algorithm
func ssabcAlgorithm() {

	// START Variables initialization
	msg := make(map[int]map[string][]ssabcMT)	// received messages
	for i:=0; i < variables.N; i++ {	// initialize maps of messages for every process
		msg = flushProcessSSABCMsg(msg, i)
	}

	aDeliveredTotal := []ssabcMT{}	// every message that was AB delivered
	rDelivered := []ssabcMT{}			// RB delivered messages
	aDeliveredMT := []ssabcMT{}			// AB delivered messages
	aDelivered := [][]byte{}			// RB delivered messages

	echoMsgs := []ssabcMT{}
	readyMsgs := []ssabcMT{}
	noReq := true

	// channels to coordinate algorithm execution and message receipt
	startReadingMessages := make(chan bool)
	pauseReadingMessages := make(chan bool)
	quitReadingMessages := make(chan bool, 1)

	// incoming messages
	messageQueue := []ssabcmsg{}

	aid = 1	// just to indicate the sequence - not part of the algorithm
	ssvcid := 0	// to get the results from SSVC

	/****************************************************************************/
	// to run tests with transient faults - not part of the algorithm
	transientFaults := config.Transient
	/****************************************************************************/
	// END Variables initialization


	/* ---------------------------Execute Algorithm----------------------------- */
	go func (){
		for {

			// handle new request if exists
			if areSSABCMessagesEqual(getValue, ssabcMT{Sender: -1, Num: math.MaxUint32,
				 Value: variables.DEFAULT}) {
				select{
				case handleNewRequest <- true:
					<- readRequest
				default:
				}
			}

			// handle new received messages
			pauseReadingMessages <- true
			msg = handleNewSSABCMessages(messageQueue, msg)
			messageQueue = []ssabcmsg{}
			startReadingMessages <- true

			// Consistency checks
			if sendersCorrect,_ := areSSABCSendersCorrect(msg, variables.ID); !sendersCorrect ||
				!isSSABCReadyConsistent(msg, variables.ID) || !isSSABCEchoConsistent(msg, variables.ID) ||
				!isNumConsistent(msg, ssnum) || ssnum >= math.MaxUint32{
				msg = flushProcessSSABCMsg(msg, variables.ID) // empty every message type set
				ssnum = 0
			}

			// no request received yet
			noReq = getValue.Sender == -1 && getValue.Num == math.MaxUint32 &&
			 bytes.Equal(getValue.Value, variables.DEFAULT)

			if ssnum == 0 && !noReq{	// transient fault was detected
				getValue.Num = ssnum
				ssnum++
			}

			// RBcast
			if _, in := containsSSABCMessage(msg[variables.ID]["init"], getValue); !in && !noReq{
				msg[variables.ID]["init"] = append(msg[variables.ID]["init"], getValue)
			}

			for i:=0; i < variables.N; i++ {	// for every process
				correctSenders, incorrectTypes := areSSABCSendersCorrect(msg,i)
				conflict, conflictTypes := ssabcMessageConflictExists (msg[i])

				if !isSSABCInitCorrect(msg[i]["init"], i) {	// empty messages by i
					msg = flushProcessSSABCMsg(msg, i)
				} else if !correctSenders || conflict {	// flush certain types only
					flushTypes := append(incorrectTypes, conflictTypes...)

					for _, ftype := range flushTypes {
						msg[i][ftype] = []ssabcMT{}
					}
				}

				// send echo message for every init
				for _, initm := range msg[i]["init"] {
					if _, in := containsSSABCMessage(msg[variables.ID]["echo"], initm); !in {	// only 1 echo per init
						msg[variables.ID]["echo"] = append(msg[variables.ID]["echo"], initm)
					}
				}
			}

			// send ready message based on echo and ready messages
			// calculate echo/ready with enough support
			echoMsgs = ssabcValueThresholdOccurence(msg, int(math.Ceil(float64(variables.N + variables.F)/2)), "echo")
			readyMsgs = ssabcValueThresholdOccurence(msg, variables.F + 1, "ready")

			for _,  m := range echoMsgs {
				if _, in := containsSSABCMessage(msg[variables.ID]["ready"], m); !in { // only 1 ready per init
					msg[variables.ID]["ready"] = append(msg[variables.ID]["ready"], m)
				}
			}
			for _, m := range readyMsgs {
				if _, in := containsSSABCMessage(msg[variables.ID]["ready"], m); !in { // only 1 ready per init
					msg[variables.ID]["ready"] = append(msg[variables.ID]["ready"], m)
				}
			}


			/************************************************************************/
			// TRANSIENT TESTS - NOT PART OF THE ALGORITHM (can be removed)
			if transientFaults && (len(msg[variables.ID]["ready"])>0) {
				transientFaults=false
				// transient (initFault, echoFault, readyFault, senderFault,valueFault, numFault
				msg = faults.CreateSSABCTransientMsg(true, true, true, true, true, true,
					msg , config.TransientProbability)
			}
			/************************************************************************/

	    broadcastSSABCMessage(types.SSABCMessage{Content: msg[variables.ID]})

			// calculate ready with enough support for RB_deliver
			rDelivered = ssabcValueThresholdOccurence(msg, 2*variables.F + 1, "ready")
			rDelivered = removeABDelivered(rDelivered, aDeliveredTotal)

			if len(rDelivered) > 0 {	// something to propose

				// Convert vector to bytes
				SSVCAnswer[ssvcid] = make(chan map[int][]byte, 1)
				w, err := json.Marshal(rDelivered)
				if err != nil {
					logger.ErrLogger.Fatal(err)
				}

				go SelfStabilizedVectorConsensus(ssvcid, w)
				quit := make(chan bool)
				go func(){
					for {
						select{
						case <- quit:
							return
						default:
							time.Sleep(time.Second/5)
							broadcastSSABCMessage(types.SSABCMessage{Content: msg[variables.ID]})
						}
					}
				}()
				v := <-SSVCAnswer[ssvcid]
				quit <- true
				ssvcid++

				vector, errorOccurred := transformVector(v)

				if !errorOccurred {	// SSVC did not return bottom or transient error

					aDelivered, aDeliveredMT = getABDelivered(vector)

					// Sort messages in aDeliver and then deliver them
					sort.Slice(aDelivered, func(i, j int) bool {
						return bytes.Compare(aDelivered[i], aDelivered[j]) < 0
					})

					// client response
					Delivered <- struct {
						Id    int
						Value [][]byte
					}{aid, aDelivered}


					aDeliveredTotal = append(aDeliveredTotal, aDeliveredMT...)
					// read new request if the currect has been AB delivered
					if _, in := containsSSABCMessage(aDeliveredMT, getValue); in {
						getValue = ssabcMT{Sender: -1, Num: math.MaxUint32, Value: variables.DEFAULT}
					}

					logger.OutLogger.Print(".SSABC: ssaDelivered-", aDelivered, "\n")
					logger.OutLogger.Print(".SSABC: len-", len(aDelivered), "\n")

					// remove delivered items from msg
					msg = removeDeliveredItems(msg, aDeliveredMT)
					aid++
				}
			}
		}
	}()

	/* ---------------------------Receive Messages----------------------------- */
	go func(){
		msgChannel := messenger.SSABCChannel
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

						if !isSSABCMessageInQueue(messageQueue, message){
							messageQueue = append(messageQueue, message)
						}
					default:
					}
				}
			}
		}
	}()
}


func ssabcPrintmsg(msg map[int]map[string][]ssabcMT){
	for p := range msg{
		fmt.Println(p)
		for _,t := range []string{"init", "echo", "ready"} {
			fmt.Println(t)
			for _, m := range msg[p][t]{
				fmt.Print("(s:",m.Sender," n:",m.Num," v:"+
				string(m.Value[:])+"), ")
			}
			fmt.Println()
		}
	}
}

// Remove AB delivered items from msg
func removeDeliveredItems(msg map[int]map[string][]ssabcMT, aDelivered []ssabcMT) map[int]map[string][]ssabcMT{
	for _, abm := range aDelivered {
		for p,_ := range msg {
			for _, mtype := range []string{"init", "echo", "ready"} {
				if i, in := containsSSABCMessage(msg[p][mtype], abm); in { // remove from msg
					size := len(msg[p][mtype])
					msg[p][mtype][i] = msg[p][mtype][size-1]
					msg[p][mtype] = msg[p][mtype][:size-1]
				}
			}
		}
	}

	return msg
}

// Returns whether 2 ssabc messages are equal or not
func areSSABCMessagesEqual(m, t ssabcMT) bool {
	return t.Sender == m.Sender && bytes.Equal(m.Value, t.Value) &&
		t.Num == m.Num
}

// Remove messages from RBdelivered that was previously ABdelivered
func removeABDelivered(rDelivered, aDelivered []ssabcMT) []ssabcMT {
	for _, b := range aDelivered {
		if i, in := containsSSABCMessage(rDelivered, b); in {
			rDelivered[i] = rDelivered[len(rDelivered)-1]
    	rDelivered = rDelivered[:len(rDelivered)-1]
		}
	}

	return rDelivered
}

// Get messages to be AB delivered (>F occurence) based on SSVC decision
func getABDelivered(vector map[int][]ssabcMT) ([][]byte, []ssabcMT) {
	messages := []ssabcMT{}
	messagesCounts := []int{}

	for _, values := range vector {
		for _, m := range values {
			if index, in := containsSSABCMessage(messages, m); in {
				messagesCounts[index]++
			} else{
				messages = append(messages, m)
				messagesCounts = append(messagesCounts, 1)
			}
		}
	}

	aDelivered := [][]byte{}
	aDeliveredMT := []ssabcMT{}
	for i := range messages {
		if messagesCounts[i] > variables.F {
			aDelivered = append(aDelivered, messages[i].Value)
			aDeliveredMT = append(aDeliveredMT, messages[i])
		}
	}

	return aDelivered, aDeliveredMT
}

// Transforms a vector of bytes (SSVC decision) to a vector of []ssabcMT messages
// Returns the vector of []ssabcMT and whether bottom of PSI was returned from SSVC
func transformVector(ssvcVect map[int][]byte) (map[int][]ssabcMT, bool) {
	vector := make(map[int][]ssabcMT)
	errorCount := 0
	for k, v := range ssvcVect {
		if bytes.Equal(v, variables.DEFAULT) || bytes.Equal(v, variables.PSI) {
			errorCount++
		} else {
			temp := make([]ssabcMT, 0)
			err := json.Unmarshal(v, &temp)
			if err != nil {
				logger.ErrLogger.Printf(err.Error()," Byzantine message")
				temp = []ssabcMT{}
			}
			vector[k] = temp
		}
	}
	errorOccurred := errorCount == variables.N
	return vector, errorOccurred
}

// Check if messages from me have num smaller than my num counter
func isNumConsistent(msg map[int]map[string][]ssabcMT, num uint32) bool {
	for _, mtype := range []string{"init", "echo", "ready"} {
		for _, m := range msg[variables.ID][mtype] {
			if m.Sender == variables.ID && m.Num >= num {return false}
		}
	}
	return true
}

// Perform the required checks on new SSABC messages
func handleNewSSABCMessages(messageQueue []ssabcmsg, msg map[int]map[string][]ssabcMT) map[int]map[string][]ssabcMT{

	processesWithReqs := make(map[int]ssabcMT)
	everySender := make(map[int]bool)
	everySender[variables.ID] = true
	if len(msg[variables.ID]["init"]) > 0 && msg[variables.ID]["init"][0].Sender == variables.ID{
		processesWithReqs[variables.ID]=msg[variables.ID]["init"][0]
	}

	for i:=0; i < len(messageQueue); i++{

		message := messageQueue[i]
		sender := message.From
		content := message.SSABCMessage.Content

		everySender[sender] = true
		if len(content["init"]) > 0 && content["init"][0].Sender == sender {
			processesWithReqs[sender] = content["init"][0]
		}

		var copied int
		pmsg := map[string][]ssabcMT{}	// sender's msg copy

		for _, mtype := range []string{"echo", "ready"} {
			// copy echo/ready
			pmsg[mtype] = make([]ssabcMT, len(msg[sender][mtype]))
			copied = copy(pmsg[mtype], msg[sender][mtype])
			if copied != len(msg[sender][mtype]) {
				fmt.Println("copied "+strconv.Itoa(copied)+" len "+strconv.Itoa(len(msg[sender][mtype])))
				panic("Copied less "+mtype+" elements")
			}
		}

		for _, mtype := range []string{"echo", "ready"} {
			// add new echo and ready messages
			for _, m := range content[mtype] {
				if _, in := containsSSABCMessage(msg[sender][mtype], m); !in { // add message only 1 time
					msg[sender][mtype] = append(msg[sender][mtype], m)
					pmsg[mtype] = append(pmsg[mtype], m)
				}
			}
		}

		// add init if no conflicting echo/ready values
		if conflict, _ := ssabcMessageConflictExists (pmsg); !conflict {
			for _, m := range content["init"] {
				if _, in := containsSSABCMessage(msg[sender]["init"], m); !in { // add init only 1 time
					msg[sender]["init"] = append(msg[sender]["init"], m)
				}
			}
		}
	}

	// remove messages from processes that didn't send a request
	for pid,_ := range everySender {
		req, in := processesWithReqs[pid]
		for id,_ := range msg {
			for _, mtype := range []string{"init", "echo", "ready"} {
				temp := []ssabcMT{}
				for _, m := range msg[id][mtype] {
					if m.Sender != pid || (in && areSSABCMessagesEqual(m, req)){
						 temp = append(temp, m)
					 }
				}
				msg[id][mtype] = temp
			}
		}
	}

	return msg
}

// Check if received message already exists in message queue
func isSSABCMessageInQueue(messageQueue []ssabcmsg, newmsg ssabcmsg) bool {

	for _, msg := range messageQueue{
		if msg.From == newmsg.From { // for every message in queue
			m := msg.SSABCMessage.Content
			equalMessages := true
			for i:=0; i < 3 && equalMessages; i++{ // for every type
				mtype := []string{"init", "echo", "ready"}[i]
				newm := newmsg.SSABCMessage.Content[mtype]
				if len(m[mtype]) == len(newm) {
					for _, nm := range newm { // every msg in type
						if _, in := containsSSABCMessage(m[mtype], nm); !in {
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
func containsSSABCMessage(messages []ssabcMT, m ssabcMT) (int, bool) {
	for i, t := range messages {
		if areSSABCMessagesEqual(t, m) { return i, true }
	}
	return -1, false
}

// Flush (Initialize) message (msg) maps of process p if not initialized
func flushProcessSSABCMsg(msg map[int]map[string][]ssabcMT, p int) map[int]map[string][]ssabcMT {
	if _, in := msg[p]; !in {
		msg[p] = map[string][]ssabcMT{}
	}
	msg[p]["init"] = []ssabcMT{}
	msg[p]["echo"] = []ssabcMT{}
	msg[p]["ready"] = []ssabcMT{}
	return msg
}

// Checks if sender IDs of messages are correct (between [0,N))
func areSSABCSendersCorrect(msg map[int]map[string][]ssabcMT, p int) (bool, []string) {
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
func isSSABCEchoConsistent (msg map[int]map[string][]ssabcMT, p int) bool {
	pEchoMsgs := msg[p]["echo"]	// echo messages of process p

	for _, m := range pEchoMsgs {
		if _, sin := msg[m.Sender]; sin {	// check existance of correct sender
			if _, in := containsSSABCMessage(msg[m.Sender]["init"], m); !in { // corresponding init check
				return false
			}
		} else { return false }
	}
	return true
}

// Checks if ready messages of process p are consistent (either f+1 have seen a
// ready message or (n+f)/2 have seen an echo one)
// Only for msg[variables.ID] entries
func isSSABCReadyConsistent (msg map[int]map[string][]ssabcMT, p int) bool {
	pReadyMsgs := msg[p]["ready"]	// ready messages of process p

	echoMsgs := ssabcValueThresholdOccurence(msg, int(math.Ceil(float64(variables.N + variables.F)/2)), "echo")
	readyMsgs := ssabcValueThresholdOccurence(msg, variables.F + 1, "ready")

	for _, m := range pReadyMsgs {
		_, inThresholdEcho := containsSSABCMessage(echoMsgs, m)
		_, inThresholdReady := containsSSABCMessage(readyMsgs, m)
		if !(inThresholdEcho || inThresholdReady) {
			return false
		}
	}

	return true
}

// Checks if at most 1 init message exists with each id and it is correctly formatted
func isSSABCInitCorrect (messages []ssabcMT, id int) bool {
	if len(messages) == 0 {return true}
	if len(messages) == 1 {
			return messages[0].Sender == id
	}
	return false
}

// Checks if conflicting echo/ready messages have been sent by a process
// (2 messages from a sender that have different values)
func ssabcMessageConflictExists (messages map[string][]ssabcMT) (bool, []string) {

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

// Returns every value v that has been send as a messageType message from
// >=s processes (at least s)
// messageType must be either echo or ready
func ssabcValueThresholdOccurence (msg map[int]map[string][]ssabcMT, s int,
	messageType string) []ssabcMT {
		if !(messageType == "echo" || messageType == "ready") {
			panic("Module self_stabilized_vector_consensus Function"+
				"ssabcValueThresholdOccurence: messageType must be either echo or ready")
		}

		values := []ssabcMT{}	// values appearing >=s times (with their sender)
		messages := []ssabcMT{}	// every message of type messageType sent
		messageCounts := []int{}	// counter for every message in messages

		//	count the occurence of each message
		for process := range msg {
			for _, message := range msg[process][messageType] {
				if index, in := containsSSABCMessage(messages, message); in {
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
func broadcastSSABCMessage(ssabcm types.SSABCMessage) {
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	err := encoder.Encode(ssabcm)
	if err != nil {
		logger.ErrLogger.Fatal(err)
	}

	message := types.NewMessage(w.Bytes(), "SSABC")
	messenger.Broadcast(message)
}
