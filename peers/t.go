package main

import (
	//"bufio"
	"context"
	"flag"
	"log"
	"net"
	"sync"
	"math/rand"
	//"os"
	"strconv"
	"time"

	proto "github.com/MrPrip/DistributedMutualExclusion/proto"
	"google.golang.org/grpc"
)

var (
	clientID = flag.Int("id", 0, "Client id")
)

const (
	portZerovalue = 5000
)

type peer struct {
	proto.UnimplementedP2PServer
	id         int
	port       int
	lamport    int
	wantToCS   bool
	replyCount int
	inCS       bool 
	clients    map[int]proto.P2PClient
	quorumSize int
	ctx        context.Context
	mu         sync.Mutex 
}

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := &peer{
		id:         *clientID,
		port:       *clientID + portZerovalue,
		lamport:    0,
		wantToCS:   false,
		replyCount: 0,
		clients:    make(map[int]proto.P2PClient),
		quorumSize: 0,
		ctx:        ctx,
		mu: sync.Mutex{},
	}

	
	list, err := net.Listen("tcp", "localhost:"+strconv.Itoa(p.port))
	if err != nil {
		log.Fatalf("Failed to listen on port: %v", err)
	}
	grpcServer := grpc.NewServer()
	proto.RegisterP2PServer(grpcServer, p)

	go func() {
		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to server %v", err)
		}
	}()

	for i := 1; i <= 3; i++ {
		currentPort := i + portZerovalue

		if currentPort != p.port {
			log.Printf("Waiting for port %v to come online...\n", currentPort)
			conn, err := grpc.Dial("localhost:"+strconv.Itoa(currentPort), grpc.WithInsecure(), grpc.WithBlock())
			if err != nil {
				log.Fatalf("Could not connect: %s", err)
			}
			log.Printf("Connected to port %v.\n", currentPort)

			c := proto.NewP2PClient(conn)
			p.clients[currentPort] = c
			p.quorumSize++
			defer conn.Close()
		}
	}

	
	for {
		randomInt := rand.Intn(2)
		time.Sleep(time.Duration(randomInt) * time.Second)
		p.RequestAccessToCS()
	}
}

func (p *peer) Request(ctx context.Context, req *proto.RequestAccess) (*proto.ReplyToRequest, error) {
	p.lamport++
	
	reply := &proto.ReplyToRequest{
		Id:        int32(p.id),
		TimeStamp: int64(p.lamport),
	}

	
	responseCh := make(chan bool)

	go func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		if !p.inCS || !p.wantToCS || p.canIncommingEnterCS(req.Timestamp, req.Id) {
			responseCh <- true
		} else {
			responseCh <- false
		}
	}()

	
	select {
	case granted := <-responseCh:
		if granted {
		
			return reply, nil
		} else {
			
			return nil, nil
		}
	case <-time.After(5 * time.Second):
		
		return nil, nil
	}
}


func (p *peer) RequestAccessToCS() {
	p.wantToCS = true
	request := &proto.RequestAccess{
		Id:        int32(p.id),
		Timestamp: int64(p.lamport),
	}
	p.lamport++
	p.replyCount = 0

	for id, client := range p.clients {
		if client == nil {
			log.Printf("Client with ID %d is nil, skipping...", id)
			continue
		}
		go func(id int, client proto.P2PClient) {
			reply, err := client.Request(p.ctx, request)
			if reply == nil {
				log.Println("Received nil reply from id:", id)
				return
			}
			if err != nil {
				log.Println("Error during Request call:", err)
			}
			
			p.mu.Lock()
			defer p.mu.Unlock()
			p.replyCount++
		}(id, client)
	}

	
	for p.replyCount < p.quorumSize {
		time.Sleep(100 * time.Millisecond)
		log.Println("Waiting for replies...")
	}

	
	if p.replyCount == p.quorumSize {
		p.lamport++
		p.gototCS()
		p.wantToCS = false
	}

	
}




func (p *peer) canIncommingEnterCS(incomingTimestamp int64, incommingID int32) bool {
    if incomingTimestamp < int64(p.lamport) || (incomingTimestamp == int64(p.lamport) && incommingID < int32(p.id)) {
        return true
    }
    return false
}

func (p *peer) gototCS() {
	p.mu.Lock() 
	defer p.mu.Unlock()

	p.inCS = true
	log.Printf("%d Entered Critical section\n", p.id)
	time.Sleep(2 * time.Second)
	log.Printf("%d Exiting Critical section \n", p.id)
	p.inCS = false
}
