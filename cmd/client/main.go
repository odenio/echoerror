package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	pb "github.com/example/echoerror/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func main() {
	log.Printf("starting up!")

	host := os.Getenv("ECHOERROR_HOST")
	if host == "" {
		host = "localhost"
	}
	log.Printf("configured host: %s", host)
	port := os.Getenv("ECHOERROR_PORT")
	if port == "" {
		port = "9090"
	}
	log.Printf("configured port: %s", port)
	delayMs := 1000
	if v := os.Getenv("ECHOERROR_DELAY_MS"); v != "" {
		if d, err := strconv.Atoi(v); err == nil {
			delayMs = d
		}
	}
	log.Printf("Configured delay: %d", delayMs)
	padMessageKb := 64
	if v := os.Getenv("ECHOERROR_PAD_MESSAGE_KB"); v != "" {
		if d, err := strconv.Atoi(v); err == nil {
			padMessageKb = d
		}
	}
	log.Printf("Configured pad_message_kb: %d", padMessageKb)

	target := fmt.Sprintf("%s:%s", host, port)

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewEchoErrorClient(conn)

	log.Printf("Entering loop; will only log unexpected results")
	for {
		for code := 0; code <= 16; code++ {
			msg := fmt.Sprintf("test error code %d at %s", code, time.Now().Format(time.RFC3339Nano))

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			var trailer metadata.MD
			_, err := client.Echo(ctx, &pb.EchoRequest{
				Code:         int32(code),
				Message:      msg,
				PadMessageKb: int32(padMessageKb),
			}, grpc.Trailer(&trailer))
			cancel()

			sentCode := codes.Code(code)

			// Check the echoed message trailer regardless of status code
			if vals := trailer.Get("x-echo-message"); len(vals) == 0 {
				log.Printf("MISMATCH TRAILER: no x-echo-message trailer received for code=%d", code)
			} else if vals[0] != msg {
				log.Printf("MISMATCH TRAILER: sent=%q got=%q code=%d", msg, vals[0], code)
			}

			// Check the padding trailer
			if padMessageKb > 0 {
				if vals := trailer.Get("x-echo-pad"); len(vals) == 0 {
					log.Printf("MISMATCH PAD: no x-echo-pad trailer received for code=%d", code)
				} else {
					decoded, decErr := base64.StdEncoding.DecodeString(vals[0])
					if decErr != nil {
						log.Printf("MISMATCH PAD: failed to decode x-echo-pad: %v", decErr)
					} else if len(decoded) != padMessageKb*1024 {
						log.Printf("MISMATCH PAD: expected %d bytes, got %d bytes, code=%d",
							padMessageKb*1024, len(decoded), code)
					}
				}
			}

			if sentCode == codes.OK {
				if err != nil {
					log.Printf("MISMATCH: sent code=OK but got error: %v", err)
				}
				continue
			}

			// We expect an error for non-zero codes
			if err == nil {
				log.Printf("MISMATCH: sent code=%d but got no error", code)
				continue
			}

			st, ok := status.FromError(err)
			if !ok {
				log.Printf("MISMATCH: sent code=%d but got non-gRPC error: %v", code, err)
				continue
			}

			if st.Code() != sentCode {
				log.Printf("MISMATCH CODE: sent=%d (%s) got=%d (%s) message=%q",
					code, sentCode, st.Code(), st.Code(), msg)
			}
			if st.Message() != msg {
				log.Printf("MISMATCH MESSAGE: sent=%q got=%q code=%d",
					msg, st.Message(), code)
			}
		}
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}
}
