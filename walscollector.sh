#!/bin/bash
mkdir wals
for ORDERER in "$@"
do
   mkdir wals/$ORDERER
   sudo docker cp $ORDERER:/var/hyperledger/production/orderer/etcdraft/wal wals/$ORDERER/
done
sudo chown -R ubuntu: wals/ 
$GOPATH/bin/walreader -f="wals/" -to="wals.log"