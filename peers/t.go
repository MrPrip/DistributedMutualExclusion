package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"net"
	"os"
	"strconv"

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
	id                  int
	port                int
	lamport             int
	isInCriticalSection bool
	replyCount          int
	//requestQueue      []proto.RequestMessage
	amountOfPings map[int]int
	clients       map[int]proto.P2PClient
	quorumSize    int
	ctx           context.Context
}

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := &peer{
		id:                  *clientID,
		port:                *clientID + portZerovalue,
		lamport:             0,
		isInCriticalSection: false,
		replyCount:          0,
		amountOfPings:       make(map[int]int),
		clients:             make(map[int]proto.P2PClient),
		quorumSize:          0,
		ctx:                 ctx,
	}

	// Create listener tcp on port ownPort
	list, err := net.Listen("tcp", "localhost:" + strconv.Itoa(p.port))
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

	p.createConnection()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		log.Println("Enter a message")
		p.RequestAccessToCS()
	}
}

func (p *peer) Request(ctx context.Context, req *proto.RequestAccess) (*proto.ReplyToRequest, error) {
	
	senderLamportTime := req.Timestamp
	canSenderEnter := false
	if senderLamportTime < int64(p.lamport) {
		canSenderEnter = true
	} else if senderLamportTime == int64(p.lamport) {
		senderId := req.Id
		if senderId < int32(p.id){
			canSenderEnter = true
		}
	}

	rep := &proto.ReplyToRequest{
		Id: int32(p.id),
		TimeStamp: int64(p.lamport),
		CanEnter: canSenderEnter,
	}
	return rep, nil
}

func (p *peer) RequestAccessToCS() {
	request := &proto.RequestAccess{
		Id: int32(p.id),
		Timestamp: int64(p.lamport),
	}
	
	for id, client := range p.clients {
		if client == nil {
			log.Printf("Client with ID %d is nil, skipping...", id)
			continue
		}
		reply, err := client.Request(p.ctx, request)
		if err != nil {
			log.Println(err)
			log.Println("something went wrong")
		}
		log.Println("Got reply from id: " + strconv.Itoa(id) + " value " + strconv.FormatBool(reply.CanEnter))
	}
}

func (p *peer) createConnection() {
	for i := 1; i <= 3; i++ {
		currentPort := i + portZerovalue

		if currentPort != p.port {
			log.Printf("Waiting for port %v to come online...\n", currentPort)
			conn, err := grpc.Dial("localhost:" + strconv.Itoa(currentPort), grpc.WithInsecure(), grpc.WithBlock())
			if err != nil {
				log.Fatalf("Could not connect: %s", err)
			}
			log.Printf("Connected to port %v.\n", currentPort)
			
			c := proto.NewP2PClient(conn)
			p.clients[currentPort] = c
			p.quorumSize++
			conn.Close()
		}
		
	}
}