package faults

// Implementation of every transient fault

import (
	"BFTWithoutSignatures/logger"
	"BFTWithoutSignatures/types"
	"BFTWithoutSignatures/variables"
	"BFTWithoutSignatures/config"
	"time"
	"math/rand"
	"math"
	"bytes"
)

type ssvcMT = types.SSVCMessageTuple
type ssabcMT = types.SSABCMessageTuple


func CreateSSVCTransientMsg(initFault, echoFault, readyFault, senderFault,
    valueFault bool, msg map[int]map[string][]ssvcMT, ssvcid int, p float64) map[int]map[string][]ssvcMT{

	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	if r.Float64() < p {
		faultTypes := []string{}

		// init
		if initFault {
		  faultTypes = append(faultTypes, "init")
		}
		// echo
		if echoFault {
		  faultTypes = append(faultTypes, "echo")
		}
		// ready
		if readyFault {
		  faultTypes = append(faultTypes, "ready")
		}

		// transient fault occurence
		for _, ftype := range faultTypes {
			logger.OutLogger.Print("SSVC: ",ftype," transient fault:\n")
			for id:=0;id<variables.N;id++{
				for i,_ := range msg[id][ftype]{
		        if senderFault{ // change sender
							logger.OutLogger.Print(id," sender --> sender + 1\n")
		  				msg[id][ftype][i].Sender = msg[id][ftype][i].Sender+1
		        }
		        if valueFault{ // change value
							logger.OutLogger.Print(id," values --> all '0'\n")
		          msg[id][ftype][i].Value = []byte("0")
		        }
					}
			}
		}
	}
	return msg
}

// for debugging purposes with 3 tests in TestSSVConsensus
// sender transient is changing sender to sender + 1
// checks if every value is the expected
func IsSSVCTransientRemoved (msg map[int]map[string][]ssvcMT, ssvcid int) bool {

	var value []byte
	for id:=0;id<variables.N;id++{
		// is id-th process Byzantine (in execution where Byzantine exist)
		isByzantine := id < variables.F && config.Scenario != "NORMAL"
		if !isByzantine {
			for _,t := range []string{"init","echo","ready"}{
				for i,_ := range msg[id][t]{ // check if 0 msg exists in any message
					if msg[id][t][i].Sender >= variables.F {	// not Byzantine message
						// value of each process based on the test
						switch ssvcid {
						case 1:
							if msg[id][t][i].Sender % 2 == 0 {
								value = []byte("AEK")
							} else {
								value = []byte("aek")
							}
						case 2: value = []byte("AEK")
						case 3: value = []byte("LFC")
						}

						if  bytes.Compare(msg[id][t][i].Value,value)!=0 ||
								msg[id][t][i].Sender < 0 || msg[id][t][i].Sender >= variables.N {
							return false
						}
					}
				}
			}
		}
	}
	logger.OutLogger.Print("SSVC: Cleared Transient")
	return true
}

func CreateSSABCTransientMsg(initFault, echoFault, readyFault, senderFault,
    valueFault, numFault bool, msg map[int]map[string][]ssabcMT, p float64) map[int]map[string][]ssabcMT{

	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	if r.Float64() < p {
		faultTypes := []string{}

		// init
		if initFault {
		  faultTypes = append(faultTypes, "init")
		}
		// echo
		if echoFault {
		  faultTypes = append(faultTypes, "echo")
		}
		// ready
		if readyFault {
		  faultTypes = append(faultTypes, "ready")
		}

		// transient fault occurence
		for _, ftype := range faultTypes {
			logger.OutLogger.Print("SSABC: ",ftype," transient fault:\n")
			for id:=0;id<variables.N;id++{
				for i,_ := range msg[id][ftype]{
		        if senderFault{ // change sender
							logger.OutLogger.Print(id," sender --> sender + 1\n")
		  				msg[id][ftype][i].Sender = msg[id][ftype][i].Sender+1
		        }
		        if valueFault{ // change value
							logger.OutLogger.Print(id," values --> all '0'\n")
		          msg[id][ftype][i].Value = []byte("0")
		        }
						if numFault{ // change num
							logger.OutLogger.Print(id," num --> all MaxUint32 / 2\n")
		          msg[id][ftype][i].Num = math.MaxUint32/2
		        }
					}
			}
		}
	}
	return msg
}

// for debugging purposes with 3 tests in TestSSVConsensus
// sender transient is changing sender to sender + 1
// checks if every value is not "0" ("0" are not sent by processes in tests)
// checks if message num is less than the process' num on messages sent by me
func IsSSABCTransientRemoved (msg map[int]map[string][]ssabcMT, num uint32) bool {

	byzantineScenario := config.Scenario != "NORMAL"
	for id:=0;id<variables.N;id++{
		// is id-th process Byzantine (in execution where Byzantine exist)
		isByzantine := id < variables.F && byzantineScenario
		if !isByzantine {
			for _,t := range []string{"init","echo","ready"}{
				for i,_ := range msg[id][t]{ // check if 0 msg exists in any message
					if msg[id][t][i].Sender >= variables.F || !byzantineScenario{	// not Byzantine message
						if  bytes.Compare(msg[id][t][i].Value, []byte("0"))==0 ||
								msg[id][t][i].Sender < 0 || msg[id][t][i].Sender >= variables.N ||
								(msg[id][t][i].Sender==variables.ID && msg[id][t][i].Num >= num) ||
								msg[id][t][i].Num == math.MaxUint32/2{
							return false
						}
					}
				}
			}
		}
	}
	logger.OutLogger.Print("SSABC: Cleared Transient")
	return true
}
