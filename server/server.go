package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	pb "github.com/hitheshthummala/chat-grpc/chat"
	"google.golang.org/grpc"
)

type chatServer struct {
	pb.UnimplementedChatServiceServer
	mu        sync.Mutex
	clients   map[pb.ChatService_ChatStreamServer]string
	broadcast chan *pb.Message
}

func (s *chatServer) JoinChat(ctx context.Context, req *pb.JoinRequest) (*pb.JoinResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// In a real app, you'd want to check for duplicate usernames
	return &pb.JoinResponse{
		Message: fmt.Sprintf("Welcome %s! You've joined the chat.", req.User),
	}, nil
}

func (s *chatServer) ChatStream(stream pb.ChatService_ChatStreamServer) error {
	firstMsg, err := stream.Recv()
	if err != nil {
		return err
	}
	user := firstMsg.User

	s.mu.Lock()
	s.clients[stream] = user
	s.mu.Unlock()

	log.Printf("%s has connected", user)

	// Send welcome message
	s.broadcast <- &pb.Message{
		User:      "Server",
		Text:      fmt.Sprintf("%s has joined the chat!", user),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	defer func() {
		s.mu.Lock()
		delete(s.clients, stream)
		s.mu.Unlock()

		s.broadcast <- &pb.Message{
			User:      "Server",
			Text:      fmt.Sprintf("%s has left the chat", user),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		log.Printf("%s has disconnected", user)
	}()

	// Receive messages from this client and send to broadcast
	for {
		msg, err := stream.Recv()
		if err != nil {
			return nil // client disconnected or error
		}
		msg.Timestamp = time.Now().Format(time.RFC3339)
		s.broadcast <- msg
	}
}

func newServer() *chatServer {
	return &chatServer{
		clients:   make(map[pb.ChatService_ChatStreamServer]string),
		broadcast: make(chan *pb.Message, 1000),
	}
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server := newServer()

	// Start broadcasting messages to all clients
	go func() {
		for msg := range server.broadcast {
			server.mu.Lock()
			for client := range server.clients {
				if err := client.Send(msg); err != nil {
					log.Printf("Failed to send message to %s: %v", server.clients[client], err)
				}
			}
			server.mu.Unlock()
		}
	}()

	grpcServer := grpc.NewServer()
	pb.RegisterChatServiceServer(grpcServer, server)
	log.Printf("Server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
