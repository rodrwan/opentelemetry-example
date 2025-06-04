package main

import (
	"context"
	"log"
	"net"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	pb "proto"
)

var tracer trace.Tracer

type serverC struct {
	pb.UnimplementedServiceCServer
}

func main() {
	tp := initTracer()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("error shutting down tracer provider: %v", err)
		}
	}()

	tracer = otel.Tracer("service-c")

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	pb.RegisterServiceCServer(grpcServer, &serverC{})

	log.Println("Service C listening on :50052")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (s *serverC) DoSomethingElse(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	ctx, span := tracer.Start(ctx, "DoSomethingElse in C")
	defer span.End()

	result := req.Message + " -> C"
	return &pb.Response{Result: result}, nil
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
			semconv.ServiceNameKey.String("service-c"),
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
