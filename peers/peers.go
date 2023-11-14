package main
/*
import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	proto "hw04/grpc"

	"google.golang.org/grpc"
)

type Peer struct {
	proto.UnimplementedNodeManagementServer
	id                        int64
	Timestamp                 int64
	hightestSeen              int64
	Acknowledged              map[int64]bool
	clients                   map[int64]proto.NodeManagementClient
	ctx                       context.Context
	usingCriticalSection      bool
	requestingCriticalSection bool
	mu                        sync.Mutex
}

func main() {
	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32)
	ownPort := int64(arg1) + 5000

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := &Peer{
		id:                        ownPort,
		Timestamp:                 1,
		hightestSeen:              1,
		Acknowledged:              make(map[int64]bool),
		clients:                   make(map[int64]proto.NodeManagementClient),
		ctx:                       ctx,
		usingCriticalSection:      false,
		requestingCriticalSection: false,
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", ownPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	proto.RegisterNodeManagementServer(s, p)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to server %v", err)
		}
	}()

	for i := 0; i < 3; i++ {
		port := int64(5000) + int64(i)

		if port == ownPort {
			continue
		}

		var conn *grpc.ClientConn
		fmt.Printf("Trying to dial: %v\n", port)
		conn, err := grpc.Dial(fmt.Sprintf(":%v", port), grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			log.Fatalf("Could not connect: %s", err)
		}
		defer conn.Close()
		c := proto.NewNodeManagementClient(conn)
		p.clients[port] = c
	}

	for true {
		i := rand.Intn(1e2 / 2)
		if i == 42 {
			p.sendPingToAll()
		}
		randomSleep(1e2, 1e2)
	}
}

func (p *Peer) InitialContact(ctx context.Context, req *proto.Request) (*proto.Reply, error) {
	id := req.Id
	if p.hightestSeen < req.Timestamp {
		p.hightestSeen = req.Timestamp
	}
	for p.usingCriticalSection {
		//fmt.Println("using critical section waiting for it to stop")
		randomSleep(1e2, 3e2)
	}
	for p.compare(req.Id, req.Timestamp) && p.requestingCriticalSection {
		I := 0
		if I%2 == 0 {
			//fmt.Println("I am ahead in the que")
		}
		I++
		randomSleep(1e2, 3e2)
	}
	p.Acknowledged[id] = true
	rep := &proto.Reply{Id: p.id, Acknowledge: p.Acknowledged[id]}
	return rep, nil
}

func (p *Peer) compare(incommingID int64, incommingTime int64) bool {
	if incommingTime == p.Timestamp {
		if incommingID > p.id {
			return true
		}
	} else if incommingTime > p.Timestamp {
		return true
	} else {
		return false
	}
	return false
}

func (p *Peer) sendPingToAll() {
	p.requestingCriticalSection = true
	p.IncrementTimestamp()
	request := &proto.Request{Id: p.id, Timestamp: p.Timestamp}
	count := 0

	for id, client := range p.clients {
		reply, err := client.InitialContact(p.ctx, request)
		if err != nil {
			fmt.Println("something went wrong")
		}
		if reply.Acknowledge {
			count++
		}
		fmt.Printf("\nGot reply from id %v: %v\n count now at: %d\n", id, reply.Acknowledge, count)

	}
	if count == 2 {
		p.working()
		p.IncrementTimestamp()
		p.requestingCriticalSection = false
	}

}

func (p *Peer) IncrementTimestamp() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.hightestSeen > p.Timestamp {
		p.Timestamp = p.hightestSeen
	}
	p.Timestamp++
	fmt.Printf("Timestamp at: %d\n", p.Timestamp)
}

func (p *Peer) working() {
	p.usingCriticalSection = true
	fmt.Println("Using critical section")
	randomSleep(5e3, 5e3)
	fmt.Println("I am done using critical section")
	p.usingCriticalSection = false
}

func randomSleep(minimum int, rangeofNumbers int) {
	waitTimer := (rand.Intn(rangeofNumbers) + minimum)
	//fmt.Printf("Wait  time in mS: %d\n", waitTimer)
	time.Sleep(time.Duration(float64(waitTimer) * float64(time.Millisecond)))
}
*/