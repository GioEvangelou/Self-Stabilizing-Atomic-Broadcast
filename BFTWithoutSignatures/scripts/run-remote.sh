#!/bin/bash

MACHINE=$1

N=10
CLIENTS=25
SCEN=0

TRANSIENT=0
SSABC=1 # SSABC == 0 -> run ABC algorithm | SSABC == 1 -> run SSABC algorithm

if [ $MACHINE -eq 2 ]
then
	go install BFTWithoutSignatures
	go install BFTWithoutSignatures_Client
	go run BFTWithoutSignatures generate_keys $N
fi

ID=$MACHINE
echo "STARTED $ID"
go run BFTWithoutSignatures $ID $N $CLIENTS $SCEN 1 $TRANSIENT $SSABC &

if [ $MACHINE -eq 2 ] || [ $MACHINE -eq 3 ]
then
	TEMP=(($MACHINE%2))
	for (( ID=(($TEMP+8)); ID<$N; ((ID+=2)) ))
	do
		echo "STARTED $ID"
		go run BFTWithoutSignatures $ID $N $CLIENTS $SCEN 1 $TRANSIENT $SSABC &
	done
fi
