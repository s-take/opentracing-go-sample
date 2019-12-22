package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

var (
	url string
)

var client = http.Client{
	Timeout: time.Millisecond * 30000,
}

// initJaeger returns an instance of Jaeger Tracer that samples 100% of traces and logs all spans to stdout.
func initJaeger(service string) (opentracing.Tracer, io.Closer) {
	cfg := jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans: true,
		},
	}
	tracer, closer, err := cfg.New(service, jaegercfg.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}

func main() {
	flag.StringVar(&url, "url", "http://localhost:8080", "setting get url")
	flag.Parse()

	// Initialize tracer with a logger and a metrics factory
	tracer, closer := initJaeger("echo echo service")
	// Set the singleton opentracing.Tracer with the Jaeger tracer.
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	http.DefaultTransport.(*http.Transport).MaxIdleConns = 0
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 1000

	mux := http.NewServeMux()
	mux.HandleFunc("/", dump)
	mux.HandleFunc("/slow", slow)
	mux.HandleFunc("/error", errorRes)
	log.Fatal(http.ListenAndServe(":8081", mux))

}

func dump(w http.ResponseWriter, r *http.Request) {
	span := opentracing.StartSpan("Start Span")
	defer span.Finish()

	ctx := opentracing.ContextWithSpan(context.Background(), span)
	testchildspan(ctx)
}

func testchildspan(ctx context.Context) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
	}
	span, _ := opentracing.StartSpanFromContext(ctx, "request echo service")
	defer span.Finish()
	span.SetTag("role", "childspan")
	span.SetBaggageItem("name", "take")
	ext.SpanKindRPCClient.Set(span)
	ext.HTTPUrl.Set(span, url)
	ext.HTTPMethod.Set(span, "GET")
	span.Tracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header),
	)
	resp, err := client.Do(req)
	dumpReq, _ := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.Println(err)
	}
	span.SetBaggageItem("dump", string(dumpReq))
	log.Println(resp.Status)
}

func slow(w http.ResponseWriter, r *http.Request) {

	io.WriteString(w, "This is echoecho service\n")
	// Request
	u := url + "/slow"
	req, _ := http.NewRequest("GET", u, nil)
	client := new(http.Client)

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()
	// _, err = ioutil.ReadAll(resp.Body)

	dumpResp, _ := httputil.DumpResponse(resp, true)
	io.WriteString(w, "===DumpResponse===\n")
	io.WriteString(w, string(dumpResp))
}

func errorRes(w http.ResponseWriter, r *http.Request) {

	io.WriteString(w, "This is echoecho service\n")
	// Request
	u := url + "/error"
	req, _ := http.NewRequest("GET", u, nil)

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()
	// _, err = ioutil.ReadAll(resp.Body)

	dumpResp, _ := httputil.DumpResponse(resp, true)
	io.WriteString(w, "===DumpResponse===\n")
	io.WriteString(w, string(dumpResp))
}
