# OpenTelemetry in Go - Distributed Tracing Example

This repository contains a practical example of implementing distributed tracing using OpenTelemetry in Go. The project demonstrates how to instrument multiple microservices and visualize traces using Jaeger.

## Architecture

The project consists of 5 services that communicate with each other:

- **Service A**: HTTP service that initiates the call chain
- **Service B**: gRPC service that receives calls from A and calls C
- **Service C**: gRPC service that receives calls from B and calls D
- **Service D**: HTTP service that ends the chain
- **Service E**: GraphQL service with tracing instrumentation

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- Make (optional, for using Makefile commands)

## Setup

1. Clone the repository:
```bash
git clone https://github.com/rodrwan/opentelemetry-example.git
cd opentelemetry-example
```

2. Start the services using Docker Compose:

There are several ways to start the services depending on your needs:

### Option 1: All services (recommended)
```bash
make up 
```

### Available Components:

- **Application Services**:
  - Service A (HTTP): http://localhost:8088
  - Service B (gRPC): localhost:50051
  - Service C (gRPC): localhost:50052
  - Service D (HTTP): http://localhost:8089
  - Service E (GraphQL): http://localhost:8090

- **Observability Tools**:
  - Jaeger UI: http://localhost:16686
  - Grafana: http://localhost:3000 (user: admin, password: admin)
  - Tempo: http://localhost:3200

### Important Environment Variables:

The services use the following environment variables that are automatically configured in docker-compose:

- `OTEL_EXPORTER_OTLP_ENDPOINT`: OpenTelemetry collector endpoint
- `OTEL_SERVICE_NAME`: Service name for identification in Jaeger

### Volumes:

The project uses volumes for:
- Go modules cache
- Go build cache
- Grafana configuration
- Tempo configuration

## OpenTelemetry Implementation

### 1. Tracer Configuration

Each service implements the tracer configuration similarly:

```go
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
            semconv.ServiceNameKey.String("service-name"),
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
```

### 2. HTTP Service Instrumentation

For HTTP services, the `otelhttp` middleware is used:

```go
mux.Handle("/endpoint", otelhttp.NewHandler(http.HandlerFunc(handler), "HandlerName"))
```

### 3. gRPC Service Instrumentation

For gRPC services, the `otelgrpc` interceptor is used:

```go
conn, err := grpc.Dial(
    "service:port",
    grpc.WithTransportCredentials(insecure.NewCredentials()),
    grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
)
```

### 4. GraphQL Service Instrumentation

For GraphQL services, we use a custom tracing middleware:

```go
srv.Use(graphqlTracer.TracerMiddleware(tracer))
```

The GraphQL service also includes:
- GraphQL Playground for testing queries
- Automatic Persisted Queries (APQ) support
- Query caching
- OpenTelemetry instrumentation for both HTTP and GraphQL operations

## Trace Visualization

1. Open Jaeger UI at http://localhost:16686
2. Select the service you want to analyze
3. Configure the time range and click "Find Traces"

## Environment Variables

The services use the following environment variables:

- `OTEL_EXPORTER_OTLP_ENDPOINT`: OpenTelemetry collector endpoint
- `OTEL_SERVICE_NAME`: Service name for identification in Jaeger

## Useful Commands

The project includes a Makefile with useful commands:

```bash
make build    # Build all services
make run      # Run services locally
make clean    # Clean generated files
```

## Contributing

Contributions are welcome. Please open an issue to discuss proposed changes.

## License

MIT
