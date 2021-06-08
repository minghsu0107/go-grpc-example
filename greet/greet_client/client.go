package main

import (
	"context"
	"fmt"
	"greet/greetpb"
	"io"
	"log"
	"time"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/ratelimit"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

// UnaryService is the interface of the unary methods
type UnaryService interface {
	Greet(firstName, lastName string) (string, error)
}

// UnaryServiceImpl is the of the unary endpoints that compose the service
type UnaryServiceImpl struct {
	Ctx           context.Context
	UnaryEndpoint endpoint.Endpoint
}

// Greet method implements the service interface, so Set may be used as a service.
func (svc *UnaryServiceImpl) Greet(firstName, lastName string) (string, error) {
	res, err := svc.UnaryEndpoint(svc.Ctx, &greetpb.GreetRequest{
		Greeting: &greetpb.Greeting{
			FirstName: firstName,
			LastName:  lastName,
		},
	})
	if err != nil {
		return "", err
	}
	response := res.(*greetpb.GreetResponse)
	return response.Result, nil
}

func newUnaryService(conn *grpc.ClientConn) UnaryService {
	limiter := ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 100))

	// global client middlewares
	// var options []grpctransport.ClientOption

	// Each individual endpoint is an grpc/transport.Client (which implements
	// endpoint.Endpoint) that gets wrapped with various middlewares. If you
	// made your own client library, you'd do this work there, so your server
	// could rely on a consistent set of client behavior.
	var unaryEndpoint endpoint.Endpoint
	{
		unaryEndpoint = grpctransport.NewClient(
			conn,
			"greet.GreetService",
			"Greet",
			encodeGRPCunaryRequest,
			decodeGRPCunaryResponse,
			&greetpb.GreetResponse{},
		).Endpoint()
		unaryEndpoint = limiter(unaryEndpoint)
		unaryEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "unary",
			Timeout: 30 * time.Second,
		}))(unaryEndpoint)
	}

	return &UnaryServiceImpl{
		Ctx:           context.Background(),
		UnaryEndpoint: unaryEndpoint,
	}
}

func encodeGRPCunaryRequest(_ context.Context, request interface{}) (interface{}, error) {
	/*
		req := request.(*greetpb.GreetRequest)
		return &greetpb.GreetRequest{
			Greeting: &greetpb.Greeting{
				FirstName: req.Greeting.FirstName,
				LastName:  req.Greeting.LastName,
			},
		}, nil
	*/
	return request, nil
}

func decodeGRPCunaryResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	/*
		reply := grpcReply.(*greetpb.GreetResponse)
		return &greetpb.GreetResponse{
			Result: reply.Result,
		}, nil
	*/
	return grpcReply, nil
}

func main() {

	fmt.Println("Hello I'm a client")

	tls := true
	opt := grpc.WithInsecure()
	if tls {
		//certFile := "ssl/ca.crt" // deprecated, should be compiled with GODEBUG=x509ignoreCN=0 if we want to use it
		certFile := "ssl/cert.pem"
		creds, sslErr := credentials.NewClientTLSFromFile(certFile, "")
		if sslErr != nil {
			log.Fatalf("Error while loading CA trust certificate: %v", sslErr)
			return
		}
		opt = grpc.WithTransportCredentials(creds)
	}
	retryOpts := []grpc_retry.CallOption{
		// generate waits between 900ms to 1100ms
		grpc_retry.WithBackoff(grpc_retry.BackoffLinearWithJitter(1*time.Second, 0.1)),
		// retry only on NotFound and Unavailable
		grpc_retry.WithCodes(codes.NotFound, codes.Aborted),
	}

	// connection timeout after  3 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// control client side max receive msg size and connection timeout
	cc, err := grpc.DialContext(
		ctx,
		"localhost:10051",
		opt,
		grpc.WithStreamInterceptor(grpc_retry.StreamClientInterceptor(retryOpts...)),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(retryOpts...)),
		grpc.WithMaxMsgSize(1024*1024*8),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer cc.Close()

	unarySvc := newUnaryService(cc)
	var res string
	res, err = unarySvc.Greet("ming", "hsu")
	if err != nil {
		log.Fatalf("Error in unary grpc client, %v", err)
	} else {
		log.Printf("Unary grpc client returns %v\n", res)
	}

	c := greetpb.NewGreetServiceClient(cc)

	doUnary(c)
	doServerStreaming(c)
	doClientStreaming(c)
	doBiDiStreaming(c)

	doUnaryWithDeadline(c, 5*time.Second) // should complete
	doUnaryWithDeadline(c, 1*time.Second) // should timeout
}

func doUnary(c greetpb.GreetServiceClient) {
	fmt.Println("Starting to do a Unary RPC...")
	req := &greetpb.GreetRequest{
		Greeting: &greetpb.Greeting{
			FirstName: "Stephane",
			LastName:  "Maarek",
		},
	}
	res, err := c.Greet(context.Background(), req)
	if err != nil {
		log.Fatalf("error while calling Greet RPC: %v", err)
	}
	log.Printf("Response from Greet: %v", res.Result)
}

func doServerStreaming(c greetpb.GreetServiceClient) {
	fmt.Println("Starting to do a Server Streaming RPC...")

	req := &greetpb.GreetManyTimesRequest{
		Greeting: &greetpb.Greeting{
			FirstName: "Stephane",
			LastName:  "Maarek",
		},
	}

	resStream, err := c.GreetManyTimes(context.Background(), req)
	if err != nil {
		log.Fatalf("error while calling GreetManyTimes RPC: %v", err)
	}
	for {
		msg, err := resStream.Recv()
		if err == io.EOF {
			// we've reached the end of the stream
			break
		}
		if err != nil {
			log.Fatalf("error while reading stream: %v", err)
		}
		log.Printf("Response from GreetManyTimes: %v", msg.GetResult())
	}

}

func doClientStreaming(c greetpb.GreetServiceClient) {
	fmt.Println("Starting to do a Client Streaming RPC...")

	requests := []*greetpb.LongGreetRequest{
		&greetpb.LongGreetRequest{
			Greeting: &greetpb.Greeting{
				FirstName: "Stephane",
			},
		},
		&greetpb.LongGreetRequest{
			Greeting: &greetpb.Greeting{
				FirstName: "John",
			},
		},
		&greetpb.LongGreetRequest{
			Greeting: &greetpb.Greeting{
				FirstName: "Lucy",
			},
		},
		&greetpb.LongGreetRequest{
			Greeting: &greetpb.Greeting{
				FirstName: "Mark",
			},
		},
		&greetpb.LongGreetRequest{
			Greeting: &greetpb.Greeting{
				FirstName: "Piper",
			},
		},
	}

	stream, err := c.LongGreet(context.Background())
	if err != nil {
		log.Fatalf("error while calling LongGreet: %v", err)
	}

	// we iterate over our slice and send each message individually
	for _, req := range requests {
		fmt.Printf("Sending req: %v\n", req)
		stream.Send(req)
		time.Sleep(1000 * time.Millisecond)
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("error while receiving response from LongGreet: %v", err)
	}
	fmt.Printf("LongGreet Response: %v\n", res)

}

func doBiDiStreaming(c greetpb.GreetServiceClient) {
	fmt.Println("Starting to do a BiDi Streaming RPC...")

	// we create a stream by invoking the client
	stream, err := c.GreetEveryone(context.Background())
	if err != nil {
		log.Fatalf("Error while creating stream: %v", err)
		return
	}

	requests := []*greetpb.GreetEveryoneRequest{
		&greetpb.GreetEveryoneRequest{
			Greeting: &greetpb.Greeting{
				FirstName: "Stephane",
			},
		},
		&greetpb.GreetEveryoneRequest{
			Greeting: &greetpb.Greeting{
				FirstName: "John",
			},
		},
		&greetpb.GreetEveryoneRequest{
			Greeting: &greetpb.Greeting{
				FirstName: "Lucy",
			},
		},
		&greetpb.GreetEveryoneRequest{
			Greeting: &greetpb.Greeting{
				FirstName: "Mark",
			},
		},
		&greetpb.GreetEveryoneRequest{
			Greeting: &greetpb.Greeting{
				FirstName: "Piper",
			},
		},
	}

	waitc := make(chan struct{})
	// we send a bunch of messages (go routine)
	go func() {
		// function to send a bunch of messages
		for _, req := range requests {
			fmt.Printf("Sending message: %v\n", req)
			stream.Send(req)
			time.Sleep(1000 * time.Millisecond)
		}
		stream.CloseSend()
	}()
	// we receive a bunch of messages (go routine)
	go func() {
		// function to receive a bunch of messages
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("Error while receiving: %v", err)
				break
			}
			fmt.Printf("Received: %v\n", res.GetResult())
		}
		close(waitc)
	}()

	// block until everything is done
	<-waitc
}

func doUnaryWithDeadline(c greetpb.GreetServiceClient, timeout time.Duration) {
	fmt.Println("Starting to do a UnaryWithDeadline RPC...")
	req := &greetpb.GreetWithDeadlineRequest{
		Greeting: &greetpb.Greeting{
			FirstName: "Stephane",
			LastName:  "Maarek",
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	res, err := c.GreetWithDeadline(ctx, req)
	if err != nil {

		statusErr, ok := status.FromError(err)
		if ok {
			if statusErr.Code() == codes.DeadlineExceeded {
				fmt.Println("Timeout was hit! Deadline was exceeded")
			} else {
				fmt.Printf("unexpected error: %v", statusErr)
			}
		} else {
			log.Fatalf("error while calling GreetWithDeadline RPC: %v", err)
		}
		return
	}
	log.Printf("Response from GreetWithDeadline: %v", res.Result)
}
