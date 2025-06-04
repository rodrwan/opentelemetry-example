package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "proto"
)

var tracer trace.Tracer

type serverB struct {
	pb.UnimplementedServiceBServer
}

func main() {
	tp := initTracer()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("error shutting down tracer provider: %v", err)
		}
	}()

	tracer = otel.Tracer("service-b")

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	pb.RegisterServiceBServer(grpcServer, &serverB{})

	log.Println("Service B listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (s *serverB) DoSomething(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	ctx, span := tracer.Start(ctx, "DoSomething in B")
	defer span.End()

	// Call service C
	conn, err := grpc.NewClient(
		"service-c:50052",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("could not connect to service-c: %w", err)
	}
	defer conn.Close()

	client := pb.NewServiceCClient(conn)
	res, err := client.DoSomethingElse(ctx, &pb.Request{Message: req.Message + " -> B"})
	if err != nil {
		return nil, fmt.Errorf("error calling C: %w", err)
	}

	body, err := callServiceD(ctx)
	if err != nil {
		return nil, fmt.Errorf("error calling D: %w", err)
	}

	return &pb.Response{Result: res.Result + " -> B" + body}, nil
}

func initTracer() *sdktrace.TracerProvider {
	exporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("failed to create exporter: %v", err)
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String("service-b"),
		),
	)
	if err != nil {
		log.Fatalf("failed to create resource: %v", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp
}

func callServiceD(ctx context.Context) (string, error) {
	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "http://service-d:8089/hello", bytes.NewBuffer([]byte("{\"message\": \"Hello from B\"}")))
	if err != nil {
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	fmt.Println("Response from D:", string(body))
	return string(body), nil
}
