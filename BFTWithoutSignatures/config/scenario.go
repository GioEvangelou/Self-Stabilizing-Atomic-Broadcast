package config

import (
	"BFTWithoutSignatures/logger"
	"math/big"
)

var (
	Scenario string

	scenarios = map[int]string{
		0: "NORMAL",      // Normal execution
		1: "IDLE",        // Byzantine processes remain idle (send nothing)
		2: "BC_ATTACK",   // Byzantine processes send wrong bytes for BC
		3: "HALF_&_HALF", // Byzantine processes send different messages to half the servers
		4: "BZ_ALL",			// Byzantine processes send the same value to every server
	}

	Transient bool
	TransientProbability float64

	TestExecution bool
)

func InitializeScenario(s int, transientProb float64) {
	if s >= len(scenarios) {
		logger.ErrLogger.Println("Scenario out of bounds! Executing with NORMAL scenario ...")
		s = 0
	}

	Scenario = scenarios[s]

	Transient = big.NewFloat(transientProb).Cmp(big.NewFloat(0)) > 0
	TransientProbability = transientProb
}

// Indicates a test execution - from files in tests folder (for debugging)
func SetTest(isTest bool) {
	TestExecution = isTest
}
