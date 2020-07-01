package main

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"

	api "github.com/weak-head/proglog/api/v1"
)

func main() {
	addr := ":8400"
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}

	client := api.NewLogClient(conn)
	ctx := context.Background()

	res, err := client.GetServers(ctx, &api.GetServersRequest{})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("servers:")
	for _, server := range res.Servers {
		fmt.Printf("\t- %v\n", server)
	}
}
