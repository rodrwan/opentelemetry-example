package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "proto"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
)

var tracer trace.Tracer

func main() {
	tp := initTracer()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("error shutting down tracer provider: %v", err)
		}
	}()

	tracer = otel.Tracer("service-a")

	mux := http.NewServeMux()
	mux.Handle("/start", otelhttp.NewHandler(http.HandlerFunc(handler), "StartHandler"))

	log.Println("Listening on :8088")
	log.Fatal(http.ListenAndServe(":8088", mux))
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ctx, span := tracer.Start(ctx, "call-service-b")
	defer span.End()

	conn, err := grpc.NewClient(
		"service-b:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		http.Error(w, "could not connect to service-b", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	client := pb.NewServiceBClient(conn)

	res, err := client.DoSomething(ctx, &pb.Request{Message: "Hello from A"})
	if err != nil {
		http.Error(w, fmt.Sprintf("error calling B: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Response from B: %s", res.Result)
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
			semconv.ServiceNameKey.String("service-a"),
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
