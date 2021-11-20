package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	jaeger "github.com/uber/jaeger-client-go"
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

	// 因为这里使用的是istio，传递的header前缀是x-b3，所以需要使用 NewZipkinB3HTTPHeaderPropagator 去初始化
	zipkinPropagator := zipkin.NewZipkinB3HTTPHeaderPropagator()
	tracer, closer, err := cfg.New(service, config.Logger(jaeger.StdLogger), config.ZipkinSharedRPCSpan(false),
		config.Injector(opentracing.HTTPHeaders, zipkinPropagator),
		config.Extractor(opentracing.HTTPHeaders, zipkinPropagator),
	)

	// 下面的这条语句在istio中不适用
	// tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}

func driverInit(ctx context.Context) []byte {
	span, _ := opentracing.StartSpanFromContext(ctx, "远端数据库查询驱动")
	defer span.Finish()

	span.LogFields(
		otlog.String("FalconName", "此驱动通过查库来返回余额"),
		otlog.String("step1", "驱动初始化启动！"),
	)
	// time.Sleep(time.Second * 2)
	span.LogKV("step2", "驱动初始化完成！")

	url := "http://10.243.44.106:9091/balance"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err.Error())
	}

	ext.SpanKindRPCClient.Set(span)
	ext.HTTPUrl.Set(span, url)
	ext.HTTPMethod.Set(span, "GET")
	span.Tracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header),
	)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return body
}

func Welcome(ctx context.Context) {
	span, _ := opentracing.StartSpanFromContext(ctx, "随机初始化一个用户欢迎界面")
	defer span.Finish()

	span.LogFields(
		otlog.String("FalconName", "随机生成用户界面"),
		otlog.String("step1", "欢迎语挑选"),
	)
	// time.Sleep(time.Second * 2)
	span.LogFields(
		otlog.String("step2", "欢迎语挑选结束"),
		otlog.String("work", "Hello World!"),
	)

}

func main() {
	tracer, closer := initJaeger("银行自助查询工具")
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	gine := gin.Default()
	gine.GET("", func(c *gin.Context) {
		for key, value := range c.Request.Header {
			fmt.Printf("%s: %s\n", key, value)
		}

		testSpan := opentracing.SpanFromContext(c.Request.Context())
		if testSpan == nil {
			fmt.Printf("找到了span！\n")
		}

		// 因为开始时调用了opentracing.SetGlobalTracer(tracer)， 所以这里globaTracer就是tracer，这两条语句都是可行的
		// spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
		spanCtx, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
		if spanCtx == nil {
			log.Printf("未获取到spanCtx\n")
			c.JSON(500, map[string]string{"status": "spanCtx is null"})
			return
		} else if err != nil {
			log.Printf("获取spanCtx失败：%v\n", err)
			c.JSON(500, map[string]string{"status": err.Error()})
			return
		}

		rootSpan := tracer.StartSpan("服务进程初始化", opentracing.ChildOf(spanCtx))
		// rootSpan := tracer.StartSpan("服务进程初始化", ext.RPCServerOption(spanCtx))
		defer rootSpan.Finish()

		rootSpan.LogFields(
			otlog.String("FalconName", "此服务为余额查询入口"),
			otlog.String("step1", "查询服务初始化！"),
		)
		// time.Sleep(time.Second * 1)
		rootSpan.LogKV("step2", "服务初始化完成！")

		rootCtx := opentracing.ContextWithSpan(context.Background(), rootSpan)
		Welcome(rootCtx)
		balance := driverInit(rootCtx)

		c.String(200, string(balance))
	})

	log.Fatal(gine.Run(":9090"))
}
