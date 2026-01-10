package tracing

import (
	"fmt"
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

// InitTracer 初始化Jaeger Tracer
func InitTracer(serviceName, endpoint string, sampler float64) (opentracing.Tracer, io.Closer, error) {
	cfg := &jaegercfg.Configuration{
		ServiceName: serviceName,
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: sampler,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: endpoint,
		},
	}

	tracer, closer, err := cfg.NewTracer(jaegercfg.Logger(jaeger.StdLogger))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init jaeger tracer: %w", err)
	}

	opentracing.SetGlobalTracer(tracer)
	return tracer, closer, nil
}
