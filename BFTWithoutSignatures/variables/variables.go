package variables

import "sync"

var (
	// ID - This processor's id.
	ID int

	// N - Number of processors
	N int

	// F - Number of faulty processors
	F int

	// Byzantine - If the processor is byzantine or not
	Byzantine bool

	// Clients - Size of Clients Set
	Clients int

	// Remote - If we are running locally or remotely
	Remote bool

	// DEFAULT - The default value that is used in the algorithms (bottom)
	DEFAULT []byte

	// PSI - The psi value that is used in self-stabilized algorithms
	PSI []byte

	// Server metrics regarding the experiment evaluation
	MsgComplexity int
	MsgSize       int64
	MsgMutex      sync.RWMutex

	// Use SelfStabilizedAtomicBroadcast or not 
	RunSSABC bool
)

// Initialize - Variables initializer method
func Initialize(id int, n int, c int, rem int) {
	ID = id
	N = n
	F = (N - 1) / 3

	if ID < F {
		Byzantine = true
	} else {
		Byzantine = false
	}

	Clients = c

	if rem == 1 {
		Remote = true
	} else {
		Remote = false
	}

	DEFAULT = []byte("")
	PSI = []byte("Ψ")

	MsgComplexity = 0
	MsgSize = 0
	MsgMutex = sync.RWMutex{}
}

func SetABCAlgorithm(useSSABC bool) {
	RunSSABC = useSSABC
}
