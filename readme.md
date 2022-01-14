### WAL Reader
Utility for reading Hyperledger Fabric SmartBFT WAL files.

**Install**

    go install gitlab.n-t.io/atmz/walreader@latest
    
**Read WAL file**

    walreader -f some.wal

**Read dir with WAL files**

    walreader -f /some/directory
    
**Read and pipe output to file**

    walreader -f /some/directory -to wal.log

#### Automatic processing of multiple WALs from multiple orderers
1. Stop target orderers

         docker stop ----time=2 orderer1 orderer2 orderer3
   
2. Run script:

        ./walscollector.sh orderer1 orderer2 orderer3
        
`wals.log` file and `wals/` dir will be created. `wals.log` contains interpretation of all WALs produced by `walreader`. `wals/` dir contains raw WALs.