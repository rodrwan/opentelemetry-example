package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
)

func main() {
	tp := initTracer()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("error shutting down tracer provider: %v", err)
		}
	}()

	tracer := otel.Tracer("service-d")

	mux := http.NewServeMux()
	mux.Handle("/hello", otelhttp.NewHandler(http.HandlerFunc(handler(tracer)), "HelloHandler"))

	log.Println("Listening on :8089")
	log.Fatal(http.ListenAndServe(":8089", mux))
}

type Request struct {
	Message string `json:"message"`
}

func handler(tracer trace.Tracer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		_, span := tracer.Start(ctx, "Hello in D")
		defer span.End()

		var req Request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fmt.Fprint(w, req.Message+" -> D")
	}
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
			semconv.ServiceNameKey.String("service-d"),
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
