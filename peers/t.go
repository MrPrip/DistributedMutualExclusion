package main

import (
	//"bufio"
	"context"
	"flag"
	"log"
	"net"
	"sync"

	//"math/rand"
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
	queue      map[int]int
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
		queue:      make(map[int]int),
		quorumSize: 0,
		ctx:        ctx,
		mu:         sync.Mutex{},
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
		//randomInt := rand.Intn(3)
		//time.Sleep(time.Duration(randomInt) * time.Second)
		p.RequestAccessToCS()
	}
}

func (p *peer) RequestAccessToCS() {
	log.Println("Trying again");
	p.wantToCS = true

	request := &proto.RequestAccess{
		Id:        int32(p.id),
		Timestamp: int64(p.lamport),
	}

	for id, client := range p.clients {
		if client == nil {
			log.Printf("Client with ID %d is nil, skipping...", id)
			continue
		}
		reply, err := client.Request(p.ctx, request)

		if reply == nil {
			log.Println("Received nil reply from id:", id)
			continue
		}
		if err != nil {
			log.Println("something went wrong")
		}
		
		if reply.CanEnter {
			p.replyCount++
		}
	}
	i := 0
	for p.replyCount != p.quorumSize {
		time.Sleep(1 * time.Second)
		i++

		// wait 5 seconds
		if i == 5 {
			break
		}
	}
	

	if p.replyCount == p.quorumSize {
		p.gototCS()
		p.wantToCS = false
		// 5001 -> CS
		// 5002 -> 2=ja
		// 5003 -> 2=ja
		for _, clientID := range p.queue {
			client := p.clients[clientID]
			r := &proto.LateReplayMessage{
				CanEnter: true,
			}
			_, err := client.LateReply(p.ctx, r)
			if err != nil {
				log.Println("something went wrong")
			}
		}
		
		p.queue = make(map[int]int)
	}
	p.replyCount = 0
}

func (p *peer) Request(ctx context.Context, req *proto.RequestAccess) (*proto.ReplyToRequest, error) {
	p.lamport++
	canSenderEnter := false

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.inCS {
		canSenderEnter = false
		p.queue[int(req.Id)] = int(req.Id)+portZerovalue
	} else if p.wantToCS && p.canIncommingEnterCS(req.Timestamp, req.Id) {
		canSenderEnter = false
		p.queue[int(req.Id)] = int(req.Id)+portZerovalue
	} else {
		canSenderEnter = true
	}

	reply := &proto.ReplyToRequest{
		Id:        int32(p.id),
		TimeStamp: int64(p.lamport),
		CanEnter:  canSenderEnter,
	}
	return reply, nil
}

func (p *peer) LateReply(ctx context.Context, rep  *proto.LateReplayMessage) (*proto.Empty, error) {
	p.replyCount++
	return &proto.Empty{}, nil
}

func (p *peer) canIncommingEnterCS(incomingTimestamp int64, incommingID int32) bool {
	if incomingTimestamp < int64(p.lamport) || (incomingTimestamp == int64(p.lamport) && incommingID < int32(p.id)) {
		return true
	}
	return false
}

func (p *peer) gototCS() {
	//p.mu.Lock()
	//defer p.mu.Unlock()

	p.inCS = true
	log.Printf("%d Entered Critical section\n", p.id)
	time.Sleep(2 * time.Second)
	log.Printf("%d Exiting Critical section \n", p.id)
	p.inCS = false
}