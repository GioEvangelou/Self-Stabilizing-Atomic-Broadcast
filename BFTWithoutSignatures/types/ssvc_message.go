package types

import (
	"BFTWithoutSignatures/logger"
	"bytes"
	"encoding/gob"
)

// Tuple containing the sender and the corresponding value of the sender
type SSVCMessageTuple struct {
	Sender int
	Value []byte
}

// Self Stabilizing Vector Consensus SSVCMessage - SSVector consensus message struct
type SSVCMessage struct {
	SSVCid int
	Content map[string][]SSVCMessageTuple
}

// NewSSVCMessage - Creates a new Self Stabilizing VC message
func NewSSVCMessage(id int, content map[string][]SSVCMessageTuple) SSVCMessage {
	return SSVCMessage{SSVCid: id, Content: content}
}

// GobEncode - Vector consensus message encoder
func (ssvcm SSVCMessage) GobEncode() ([]byte, error) {
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	err := encoder.Encode(ssvcm.SSVCid)
	if err != nil {
		logger.ErrLogger.Fatal(err)
	}
	err = encoder.Encode(ssvcm.Content)
	if err != nil {
		logger.ErrLogger.Fatal(err)
	}
	return w.Bytes(), nil
}

// GobDecode - Vector consensus message decoder
func (ssvcm *SSVCMessage) GobDecode(buf []byte) error {
	r := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(r)
	err := decoder.Decode(&ssvcm.SSVCid)
	if err != nil {
		logger.ErrLogger.Fatal(err)
	}
	err = decoder.Decode(&ssvcm.Content)
	if err != nil {
		logger.ErrLogger.Fatal(err)
	}
	return nil
}
