package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
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

func callServer(ctx context.Context) []byte {
	span, _ := opentracing.StartSpanFromContext(ctx, "请求服务端")
	defer span.Finish()

	// v := url.Values{}
	// v.Set("helloTo", helloTo)
	// url := "http://localhost:8081/format?" + v.Encode()

	url := "http://localhost:9090"
	span.SetTag("type", "CallServer")
	span.SetTag("call url", url)
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
	span.LogFields(
		log.String("url", url),
		log.String("Do", "start"),
	)
	resp, err := client.Do(req)
	if err != nil {
		ext.LogError(span, err)
		panic(err.Error())
	}
	span.LogKV("Do", "end")
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return body
}

func main() {
	tracer, closer := initJaeger("演示例子")
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	span := tracer.StartSpan("随便做点什么")
	span.SetTag("type", "client")
	// 不是rootSpan的话， 尽量不用defer
	defer span.Finish()
	ctx := opentracing.ContextWithSpan(context.Background(), span)

	childSpan1, _ := opentracing.StartSpanFromContext(ctx, "构造参数")
	childSpan1.SetTag("type", "组合请求参数")
	// 不能使用defer ，因为这样会让childSpan1在main函数运行结束后再finish，而jaeger记录span的Duration时间是根据span的finish时间-创建时间
	childSpan1.Finish()

	// 请求服务端
	callServer(ctx)

	childSpan2, _ := opentracing.StartSpanFromContext(ctx, "处理")
	childSpan2.SetTag("type", "format")

	childSpan2.LogKV("TimeSleep", "2s")
	time.Sleep(time.Second * 2)
	childSpan2.LogKV("Sleep", "end")
	childSpan2.Finish()
}
