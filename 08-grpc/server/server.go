package main

import (
	"context"
	"fmt"
	"grpc-app/proto"
	"io"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AppServiceImpl struct {
	proto.UnimplementedAppServiceServer
}

// proto.AppServiceServer interface implementation
func (appService *AppServiceImpl) Add(ctx context.Context, r *proto.AddRequest) (*proto.AddResponse, error) {
	x := r.GetX()
	y := r.GetY()

	log.Printf("Processing [Add], x = %d and y = %d\n", x, y)
	result := x + y

	response := &proto.AddResponse{
		Result: result,
	}

	return response, nil
}

func (appService *AppServiceImpl) GeneratePrimes(r *proto.PrimesRequest, serverStream proto.AppService_GeneratePrimesServer) error {
	start := r.GetStart()
	end := r.GetEnd()
	for no := start; no <= end; no++ {
		if isPrime(no) {
			fmt.Println("[GeneratePrimes] sending prime no :", no)
			res := &proto.PrimesResponse{
				PrimeNo: no,
			}
			if err := serverStream.Send(res); err != nil {
				log.Println(err)
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
	return nil
}

func isPrime(no int64) bool {
	for i := int64(2); i <= (no / 2); i++ {
		if no%i == 0 {
			return false
		}
	}
	return true
}

func (asi *AppServiceImpl) Aggregate(serverStream proto.AppService_AggregateServer) error {
	var sum, min, max int64 = 0, 9223372036854775807, -9223372036854775808
LOOP:
	for {
		req, err := serverStream.Recv()
		if err == io.EOF {
			log.Println("[AppService - Aggregate] All the data have been received")
			res := &proto.AggregateResponse{
				Sum: sum,
				Min: min,
				Max: max,
			}
			if err := serverStream.SendAndClose(res); err != io.EOF && err != nil {
				log.Fatalln(err)
			}
			break LOOP
		}
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(req)
		no := req.GetNo()
		sum += no
		if no < min {
			min = no
		}
		if no > max {
			max = no
		}
	}
	return nil
}

func (asi *AppServiceImpl) Greet(serverStream proto.AppService_GreetServer) error {
	for {
		greetReq, err := serverStream.Recv()
		if code := status.Code(err); code == codes.Unavailable {
			fmt.Println("Client connection closed")
			break
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalln(err)
		}
		person := greetReq.GetPerson()
		firstName := person.GetFirstName()
		lastName := person.GetLastName()
		log.Printf("Received greet request for %q and %q\n", firstName, lastName)
		message := fmt.Sprintf("Hi %s %s, Have a nice day!", firstName, lastName)
		time.Sleep(2 * time.Second)
		log.Printf("Sending response : %q\n", message)
		greetResp := &proto.GreetResponse{
			Message: message,
		}
		if err := serverStream.Send(greetResp); err != nil {
			if code := status.Code(err); code == codes.Unavailable {
				fmt.Println("Client connection closed")
				break
			}
		}
	}
	return nil
}

func main() {
	asi := &AppServiceImpl{}

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalln(err)
	}

	grpcServer := grpc.NewServer()

	proto.RegisterAppServiceServer(grpcServer, asi)

	grpcServer.Serve(listener)
}
