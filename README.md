# DistributedMutualExclusion
path inside the folder called peers.
Open 3 terminals in total, 
type in the first terminal go run . -id=1
type in the second terminal go run . -id=2
type in the third terminal go run . -id=3
this will create 3 different peer nodes all communicating with each other.
Watch the 3 terminals try to enter the critical section,
Notice which peer succeeds while the other peers try again
