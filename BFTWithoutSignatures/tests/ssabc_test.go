package tests

import (
	"BFTWithoutSignatures/config"
	"BFTWithoutSignatures/logger"
	"BFTWithoutSignatures/messenger"
	"BFTWithoutSignatures/modules"
	"BFTWithoutSignatures/threshenc"
	"BFTWithoutSignatures/variables"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestSSABroadcast(t *testing.T) {
	args := os.Args[5:]
	if len(args) == 6 {
		id, _ := strconv.Atoi(args[0])
		n, _ := strconv.Atoi(args[1])
		clients, _ := strconv.Atoi(args[2])
		scenario, _ := strconv.Atoi(args[3])
		remote, _ := strconv.Atoi(args[4])
		transientProb, _ := strconv.ParseFloat(args[5], 64)

		initializeForTestSSAbc(id, n, clients, scenario, remote, transientProb)
		config.SetTest(true)
	} else {
		log.Fatal("Arguments should be '<id> <n> <clients> <scenario> <remote> <transient_probability>'")
	}

	// dummy requests reads
	go func(){
		for {
			<- modules.Delivered
		}
	}()

	/*** Start Testing ***/
	modules.InitiateSelfStabilizedAtomicBroadcast()
	time.Sleep(2 * time.Second)

	if (variables.ID % 2) == 0 {
		modules.SelfStabilizedAtomicBroadcast([]byte("A"))
	} else {
		modules.SelfStabilizedAtomicBroadcast([]byte("B"))
	}

	modules.SelfStabilizedAtomicBroadcast([]byte("AEK"))

	if variables.ID == 0 {
		modules.SelfStabilizedAtomicBroadcast([]byte("ABCD"))
	}

	if (variables.ID % 2) == 1 {
		modules.SelfStabilizedAtomicBroadcast([]byte("test"))
	}

	/*** End Testing ***/

	done := make(chan interface{}) // To keep the server running
	<-done
}

// Initializes the environment for the test
func initializeForTestSSAbc(id int, n int, clients int, scenario int, rem int, transientProb float64) {
	variables.Initialize(id, n, clients, rem)

	user_dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal( err )
	}
	user_dirname = user_dirname + "/go/src"
	logger.InitializeLogger(user_dirname+"/tests/out/", user_dirname+"/tests/error/")

	if variables.Remote {
		config.InitializeIP()
	} else {
		config.InitializeLocal()
	}
	config.InitializeScenario(scenario, transientProb)

	logger.OutLogger.Print(
		"ID:", variables.ID, " | N:", variables.N, " | F:", variables.F, " | Clients:",
		variables.Clients, " | Scenario:", config.Scenario, " | Remote:", variables.Remote,
		" | Transient:", config.Transient, " | TransientProbability:", config.TransientProbability, "\n\n",
	)

	threshenc.ReadKeys(user_dirname+"/tests/keys/")

	messenger.InitializeMessenger()
	messenger.Subscribe()
	messenger.TransmitMessages()

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		for range terminate {
			for i := 0; i < variables.N; i++ {
				if i == variables.ID {
					continue // Not myself
				}
				messenger.ReceiveSockets[i].Close()
				messenger.SendSockets[i].Close()
			}

			for i := 0; i < variables.Clients; i++ {
				messenger.ServerSockets[i].Close()
				messenger.ResponseSockets[i].Close()
			}
			os.Exit(0)
		}
	}()
}
