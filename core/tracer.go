package core

import (
	"fmt"
	"reflect"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer = otel.Tracer("github.com/hyperledger-labs/yui-relayer/core")
)

// StartTraceWithQueryContext creates a span and a QueryContext containing the newly-created span.
func StartTraceWithQueryContext(tracer trace.Tracer, ctx QueryContext, spanName string, opts ...trace.SpanStartOption) (QueryContext, trace.Span) {
	opts = append(opts, trace.WithAttributes(AttributeGroup("query",
		// Convert revision_number and revision_height to string because the attribute package does not support uint64
		AttributeKeyHeightRevisionNumber.String(fmt.Sprint(ctx.Height().GetRevisionNumber())),
		AttributeKeyHeightRevisionHeight.String(fmt.Sprint(ctx.Height().GetRevisionHeight())),
	)...))
	spanCtx, span := tracer.Start(ctx.Context(), spanName, opts...)
	ctx = NewQueryContext(spanCtx, ctx.Height())
	return ctx, span
}

// withPackage adds the package name of the function/method `v`
func withPackage(v any) trace.SpanStartOption {
	return trace.WithAttributes(AttributeKeyPackage.String(getPackageName(v)))
}

func getPackageName(v any) string {
	if v == nil {
		return ""
	}

	rt := reflect.TypeOf(v)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	return rt.PkgPath()
}
