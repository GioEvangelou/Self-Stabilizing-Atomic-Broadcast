package main

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
	"fmt"
	"syscall"
	"time"
)

var terminate chan os.Signal
var folderName string

// Initializer - Method that initializes all required processes
func initializer(id int, n int, clients int, scenario int, rem int,
								transientProb float64, runSSABC bool) {

	variables.Initialize(id, n, clients, rem)
	variables.SetABCAlgorithm(runSSABC)
	config.InitializeScenario(scenario, transientProb)

	user_dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal( err )
	}
	logger_dir := user_dirname+"/go/src/BFTWithoutSignatures/logger/"
	
	if len(folderName) == 0{
		if runSSABC {
			folderName += "SSABC"
		} else {
			folderName += "ABC"
		}
		folderName += "-S:" + strconv.Itoa(n) + "-C:" + strconv.Itoa(clients) + "-Sc:" + config.Scenario + "/"
	}
	logger_dir += folderName

	// create directories if they do not exist
	if _, err := os.Stat(logger_dir+"logs/out/"); os.IsNotExist(err) {
		err := os.MkdirAll(logger_dir+"logs/out/", 0770)
		if err != nil {
			log.Fatal(err)
		}
	}
	if _, err := os.Stat(logger_dir+"logs/error/"); os.IsNotExist(err) {
		err := os.MkdirAll(logger_dir+"logs/error/", 0770)
		if err != nil {
			log.Fatal(err)
		}
	}

	logger.InitializeLogger(logger_dir+"logs/out/", logger_dir+"logs/error/")

	if variables.Remote {
		config.InitializeIP()
	} else {
		config.InitializeLocal()
	}

	var s string
	s = fmt.Sprint("ID:", variables.ID, " | N:", variables.N, " | F:",
		variables.F, " | Clients:", variables.Clients, " | Scenario:", config.Scenario,
		" | Remote:", variables.Remote, " | Algorithm:")

	if runSSABC {
		s = fmt.Sprint(s, "Self Stabilized Atomic Broadcast | Transient:", config.Transient,
		" | TransientProbability:", config.TransientProbability, "\n\n")
	} else {
		s = fmt.Sprint(s, "Atomic Broadcast\n\n")
	}
	logger.OutLogger.Print(s)

	threshenc.ReadKeys(user_dirname+"/go/src/BFTWithoutSignatures/threshenc/keys/")


	messenger.InitializeMessenger()
	messenger.Subscribe()

	if (config.Scenario == "IDLE") && (variables.Byzantine) {
		logger.ErrLogger.Println(config.Scenario)
		return
	}

	messenger.TransmitMessages()
	if runSSABC {
		modules.InitiateSelfStabilizedAtomicBroadcast()
	} else {
		modules.InitiateAtomicBroadcast()
	}
	time.Sleep(2 * time.Second) // Wait 2s before start accepting requests to initiate all maps
	modules.RequestHandler()
}

func cleanup() {
	terminate = make(chan os.Signal, 1)
	signal.Notify(terminate,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		for range terminate {
			if (config.Scenario == "IDLE") && (variables.Byzantine) {
				logger.OutLogger.Printf("\n\nMessage Complexity: 0.00 msgs\nMessage Size: 0.000 MB\n\n")
			} else {
				logger.OutLogger.Printf(
					"\n\nMessage Complexity: %.2f msgs\nMessage Size: %.3f MB\n\n",
					float64(variables.MsgComplexity/modules.Aid),
					float64(float64(variables.MsgSize/int64(modules.Aid))/1048576)) // OR 1000000
			}

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

func main() {
	args := os.Args[1:]
	if len(args) == 2 && string(args[0]) == "generate_keys" {
		N, _ := strconv.Atoi(args[1])
		user_dirname, err := os.UserHomeDir()
		if err != nil {
			log.Fatal( err )
		}
		threshenc.GenerateKeys(N, user_dirname+"/go/src/BFTWithoutSignatures/threshenc/keys/")

	} else if len(args) == 7 || len(args) == 8 {
		id, _ := strconv.Atoi(args[0])
		n, _ := strconv.Atoi(args[1])
		clients, _ := strconv.Atoi(args[2])
		scenario, _ := strconv.Atoi(args[3])
		remote, _ := strconv.Atoi(args[4])
		transientProb, _ := strconv.ParseFloat(args[5], 64)
		runSSABC, _ := strconv.ParseBool(args[6])
		if !runSSABC {transientProb=-1}

		if len(args) == 8 {
			folderName = args[7]
		}

		initializer(id, n, clients, scenario, remote, transientProb, runSSABC)
		cleanup()

		done := make(chan interface{}) // To keep the server running
		<-done

	} else {
		log.Fatal("Arguments should be '<ID> <N> <Clients> <Scenario> <Remote> <Transient_Probability> <Run_SSABC?>'")
	}
}
