#!/bin/bash
rm ~/go/src/tests/out/*
TEST=TestSSVConsensus  # TestSSVConsensus, TestSSABroadcast, TestMVConsensus , TestVConsensus, TestABroadcast, TestBConsensus, TestBvBroadcast

N=7
CLIENTS=1
SCEN=4
REM=0
TRANSIENT=1

go install BFTWithoutSignatures

BFTWithoutSignatures generate_keys $N

for (( ID=0; ID<$N; ID++ ))
do
	if [[ "$TEST" =~ ^(TestSSABroadcast|TestSSVConsensus)$ ]]; then
		go test -v -run $TEST /home/csdeptucy/go/src/BFTWithoutSignatures/tests -args $ID $N $CLIENTS $SCEN $REM $TRANSIENT &
	else
		go test -v -run $TEST /home/csdeptucy/go/src/BFTWithoutSignatures/tests -args $ID $N $CLIENTS $SCEN $REM &
	fi
done
