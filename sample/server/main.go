package main

import (
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
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
	tracer, closer := initJaeger("服务端")
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	gine := gin.Default()
	gine.GET("", func(c *gin.Context) {
		spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
		span := tracer.StartSpan("服务端例子", ext.RPCServerOption(spanCtx))
		span.SetTag("type", "server")
		defer span.Finish()

		span.LogKV("TimeSleep", "1s")
		time.Sleep(time.Second)
		span.LogKV("sleep", "done")

		c.String(200, "success")
		return
	})

	gine.Run(":9090")
}
