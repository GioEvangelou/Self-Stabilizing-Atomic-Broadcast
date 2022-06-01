#!/bin/bash

N=4
CLIENTS=5
REM=0

SCEN=0	# SCENARIO
TRANSIENT=0	# TRANSIENT PROBABILITY
SSABC=0 # SSABC == 0 -> run ABC algorithm | SSABC == 1 -> run SSABC algorithm

go install BFTWithoutSignatures

BFTWithoutSignatures generate_keys $N

for (( ID=0; ID<$N; ID++ ))
do
	BFTWithoutSignatures $ID $N $CLIENTS $SCEN $REM $TRANSIENT $SSABC &
done
