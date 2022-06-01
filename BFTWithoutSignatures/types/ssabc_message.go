package types

import (
	"BFTWithoutSignatures/logger"
	"bytes"
	"encoding/gob"
)

// Tuple containing the sender and the corresponding value of the sender
type SSABCMessageTuple struct {
	Sender int
	Num	uint32
	Value []byte
}

// Self Stabilizing Atomic Broadcast - SSABCMessage message struct
type SSABCMessage struct {
	Content map[string][]SSABCMessageTuple
}

// NewSSABCMessage - Creates a new SS ABC message
func NewSSABCMessage(content map[string][]SSABCMessageTuple) SSABCMessage {
	return SSABCMessage{Content: content}
}

// GobEncode - SS Atomic Broadcast message encoder
func (abcm SSABCMessage) GobEncode() ([]byte, error) {
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	err := encoder.Encode(abcm.Content)
	if err != nil {
		logger.ErrLogger.Fatal(err)
	}
	return w.Bytes(), nil
}

// GobDecode - SS Atomic Broadcast message decoder
func (abcm *SSABCMessage) GobDecode(buf []byte) error {
	r := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(r)
	err := decoder.Decode(&abcm.Content)
	if err != nil {
		logger.ErrLogger.Fatal(err)
	}
	return nil
}
