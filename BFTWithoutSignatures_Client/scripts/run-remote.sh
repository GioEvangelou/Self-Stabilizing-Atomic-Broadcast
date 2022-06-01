#!/bin/bash

N=10
CLIENTS=25

for (( ID=0; ID<$CLIENTS; ID++ ))
do
	go run BFTWithoutSignatures_Client $ID $N $CLIENTS 1 &
done