package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"

	jaegercfg "github.com/uber/jaeger-client-go/config"
)

func initJaeger(service string) (opentracing.Tracer, io.Closer) {
	cfg := jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:          true,
			CollectorEndpoint: "http://localhost:14268/api/traces",
			// LocalAgentHostPort: "localhost:6831",
			// 这个是说发送到缓冲区的数据经过多长时间后会真正发送给jaeger
			BufferFlushInterval: time.Second,
		},
		ServiceName: service,
	}

	tracer, closer, err := cfg.NewTracer(jaegercfg.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}

func main() {
	tracer, closer := initJaeger("hello-world")
	defer closer.Close()
	// 不设置setGloba的话，ContextWithSpan和StartSpanFromContext创建的跨度Finish也是不会被记录的
	opentracing.SetGlobalTracer(tracer)

	rootSpan := tracer.StartSpan("say-hello")
	rootSpan.SetTag("hello-to", "anshan")

	rootSpan.LogFields(
		log.String("event", "string-format"),
		log.String("value", "anshan"),
	)
	time.Sleep(2 * time.Second)
	rootSpan.LogKV("event", "println")

	rootSpan.LogEvent("test123")
	rootSpan.LogEventWithPayload("key", map[string]string{"hello": "world"})
	defer rootSpan.Finish()

	// 两种方式创建span都可以
	// span := rootSpan.Tracer().StartSpan("hello-2",
	// 	opentracing.ChildOf(rootSpan.Context()),
	// )

	ctx := opentracing.ContextWithSpan(context.Background(), rootSpan)
	span, _ := opentracing.StartSpanFromContext(ctx, "hello-3")
	defer span.Finish()
	time.Sleep(2 * time.Second)
}
