package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	pb "github.com/hitheshthummala/grpc-chat/chat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewChatServiceClient(conn)

	// Get username
	fmt.Print("Enter your username: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	username := scanner.Text()

	// Join chat
	joinResp, err := client.JoinChat(context.Background(), &pb.JoinRequest{User: username})
	if err != nil {
		log.Fatalf("could not join chat: %v", err)
	}
	fmt.Println(joinResp.Message)

	// Start chat stream
	stream, err := client.ChatStream(context.Background())
	if err != nil {
		log.Fatalf("could not open chat stream: %v", err)
	}

	// Goroutine to receive messages
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				log.Fatalf("Error receiving message: %v", err)
			}
			fmt.Printf("\r[%s] %s: %s\n> ", msg.Timestamp, msg.User, msg.Text)
		}
	}()

	// Send initial connection message
	stream.Send(&pb.Message{
		User: username,
		Text: "has connected",
	})

	// Read input and send messages
	fmt.Print("> ")
	for scanner.Scan() {
		text := scanner.Text()
		if text == "/quit" {
			break
		}

		err := stream.Send(&pb.Message{
			User:      username,
			Text:      text,
			Timestamp: time.Now().Format(time.RFC3339),
		})
		if err != nil {
			log.Printf("Error sending message: %v", err)
			break
		}
		fmt.Print("> ")
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
	}

	fmt.Println("Goodbye!")
}
