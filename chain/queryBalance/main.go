package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	opentracing "github.com/opentracing/opentracing-go"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/zipkin"
)

func initJaeger(service string) (opentracing.Tracer, io.Closer) {
	cfg := config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:          true,
			CollectorEndpoint: "http://jaeger-collector.istio-system:14268/api/traces",
			// LocalAgentHostPort: "localhost:6831",
			// 这个是说发送到缓冲区的数据经过多长时间后会真正发送给jaeger
			BufferFlushInterval: time.Second,
		},
		// ServiceName: service,
	}

	zipkinPropagator := zipkin.NewZipkinB3HTTPHeaderPropagator()

	tracer, closer, err := cfg.New(service, config.Logger(jaeger.StdLogger), config.ZipkinSharedRPCSpan(false),
		config.Injector(opentracing.HTTPHeaders, zipkinPropagator),
		config.Extractor(opentracing.HTTPHeaders, zipkinPropagator),
	)

	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}

func main() {
	tracer, closer := initJaeger("queryBalance")
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	gine := gin.Default()
	gine.GET("balance", func(c *gin.Context) {
		for key, value := range c.Request.Header {
			fmt.Printf("%s: %s\n", key, value)
		}

		spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))

		// span := tracer.StartSpan("从数据库中获取用户余额", ext.RPCServerOption(spanCtx))
		rootSpan := tracer.StartSpan("从数据库中获取用户余额root", opentracing.ChildOf(spanCtx))

		defer rootSpan.Finish()

		rootSpan.LogFields(
			otlog.String("user", "jwping"),
			otlog.String("db", "mysql"),
		)

		// 模拟从数据库中获取余额的延迟
		// time.Sleep(time.Second * 3)

		// 随机种子
		rand.Seed(time.Now().Unix())
		// 用一个[0, 100]的随机数 * 10000 模拟获取的余额
		balancer := rand.Intn(100) * 10000
		rootSpan.LogKV("balance", balancer)

		c.String(200, strconv.Itoa(balancer))
	})

	log.Fatal(gine.Run(":9091"))
}
