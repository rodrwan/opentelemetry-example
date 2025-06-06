package tracing

import (
	"context"
	"fmt"
	"log"

	"github.com/99designs/gqlgen/graphql"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func TracerMiddleware(tracer trace.Tracer) graphql.HandlerExtension {
	return &interceptor{tracer: tracer}
}

type interceptor struct {
	tracer trace.Tracer
}

func (i *interceptor) ExtensionName() string {
	return "MyInterceptor"
}

func (i *interceptor) Validate(schema graphql.ExecutableSchema) error {
	return nil
}

// InterceptResponse captura errores de toda la operación GraphQL
func (i *interceptor) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	span := trace.SpanFromContext(ctx)

	// Ejecutar la operación
	response := next(ctx)

	// Si hay errores en la respuesta, marcar el span raíz como error
	if response != nil && len(response.Errors) > 0 && span != nil && span.SpanContext().IsValid() {
		// Marcar el span como error
		errorMsg := fmt.Sprintf("GraphQL operation completed with %d error(s)", len(response.Errors))
		span.SetStatus(codes.Error, errorMsg)
		span.SetAttributes(attribute.Bool("graphql.has_errors", true))
		span.SetAttributes(attribute.Int("graphql.error_count", len(response.Errors)))

		// Registrar todos los errores
		for idx, err := range response.Errors {
			span.RecordError(err)
			span.SetAttributes(attribute.String(fmt.Sprintf("graphql.error.%d.message", idx), err.Message))
			if err.Path != nil {
				span.SetAttributes(attribute.String(fmt.Sprintf("graphql.error.%d.path", idx), err.Path.String()))
			}
		}

		log.Printf("GraphQL operation completed with errors: %v", response.Errors)
	} else if span != nil && span.SpanContext().IsValid() {
		// No hay errores
		span.SetStatus(codes.Ok, "GraphQL operation completed successfully")
		span.SetAttributes(attribute.Bool("graphql.has_errors", false))
		span.SetAttributes(attribute.Int("graphql.error_count", 0))
	}

	return response
}

type GraphQLArgsCarrier map[string]any

func (c GraphQLArgsCarrier) Get(key string) string {
	if val, ok := c[key]; ok {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

func (c GraphQLArgsCarrier) Set(key, value string) {
	c[key] = value
}

func (c GraphQLArgsCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

func (i *interceptor) InterceptField(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
	fieldCtx := graphql.GetFieldContext(ctx)
	// get query or mutation name
	query := fieldCtx.Field.Name
	log.Printf("Intercepting Query >> %s", query)
	spanName := fmt.Sprintf("graphql.%s", query)
	ctx, span := i.tracer.Start(ctx, spanName)
	defer span.End()

	// Configurar atributos del span si es válido
	if span != nil && span.SpanContext().IsValid() {
		log.Printf("%s Span is valid", spanName)
		span.SetAttributes(attribute.String("graphql.query", query))
		span.SetAttributes(attribute.String("graphql.args", fmt.Sprintf("%v", fieldCtx.Args)))
		span.SetAttributes(attribute.String("graphql.object", fieldCtx.Object))
		span.SetAttributes(attribute.String("graphql.field", fieldCtx.Field.Name))
		span.SetAttributes(attribute.String("graphql.path", fieldCtx.Path().String()))
		span.SetAttributes(attribute.String("graphql.operation_type", fieldCtx.Field.ObjectDefinition.Name))
	}

	// Ejecutar el resolver y manejar errores
	defer func() {
		if span != nil && span.SpanContext().IsValid() {
			if err != nil {
				// Marcar el span como error
				span.SetStatus(codes.Error, err.Error())
				span.RecordError(err)
				span.SetAttributes(attribute.Bool("graphql.error", true))

				log.Printf("GraphQL resolver error: %v", err)
			} else {
				// Marcar como exitoso
				span.SetStatus(codes.Ok, "GraphQL resolver completed successfully")
				span.SetAttributes(attribute.Bool("graphql.error", false))
			}
		}
		span.End()
	}()

	// Continuar con el resolver original
	res, err = next(ctx)
	return res, err
}
