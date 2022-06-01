package faults

import (
	"BFTWithoutSignatures/variables"
	"BFTWithoutSignatures/config"
  "strconv"
)

func ByzantineValuesSSABC(msg map[int]map[string][]ssabcMT) map[int]map[string][]ssabcMT {
  // change values to their corresponding sending ones
	halfScenario := config.Scenario == "HALF_&_HALF"
	for i := 0; i < variables.N; i++ {
    msg[i]["init"] = []ssabcMT{}
  	msg[i]["echo"] = []ssabcMT{}
  	msg[i]["ready"] = []ssabcMT{}

		valueToSend := "0"
		if halfScenario {
			valueToSend = strconv.Itoa(i % 2)
		}
		msg[i]["init"] = []ssabcMT{{Sender:i,Num:0,Value:[]byte(valueToSend)}}
		for _,t := range []string{"echo", "ready"} {
			for j:=0; j<variables.N;j++{
				msg[i][t] = append(msg[i][t], ssabcMT{Sender:j,Num:0,Value:[]byte(valueToSend)})
			}
		}
	}
  return msg
}

func ByzantineValuesSSVC(msg map[int]map[string][]ssvcMT) map[int]map[string][]ssvcMT {
  // change values to their corresponding sending ones
	halfScenario := config.Scenario == "HALF_&_HALF"
	for i := 0; i < variables.N; i++ {
    msg[i]["init"] = []ssvcMT{}
  	msg[i]["echo"] = []ssvcMT{}
  	msg[i]["ready"] = []ssvcMT{}

		valueToSend := "0"
		if halfScenario {
			valueToSend = strconv.Itoa(i % 2)
		}
		msg[i]["init"] = []ssvcMT{{Sender:i,Value:[]byte(valueToSend)}}
		for _,t := range []string{"echo", "ready"} {
			for j:=0; j<variables.N;j++{
				msg[i][t] = append(msg[i][t], ssvcMT{Sender:j,Value:[]byte(valueToSend)})
			}
		}
	}
  return msg
}
