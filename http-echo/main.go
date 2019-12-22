package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	// "github.com/uber/jaeger-lib/metrics"

	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	// jaegerlog "github.com/uber/jaeger-client-go/log"
)

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

	// Initialize tracer with a logger and a metrics factory
	tracer, closer := initJaeger("echo service")
	// Set the singleton opentracing.Tracer with the Jaeger tracer.
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", dump)
	mux.HandleFunc("/slow", slow)
	mux.HandleFunc("/error", errorRes)
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func dump(w http.ResponseWriter, r *http.Request) {

	dump, _ := httputil.DumpRequest(r, true)
	io.WriteString(w, "This is echo service\n")
	io.WriteString(w, "===DumpRequest===\n")
	io.WriteString(w, string(dump))

	var serverSpan opentracing.Span
	appSpecificOperationName := "receive request"
	wireContext, err := opentracing.GlobalTracer().Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header))
	if err != nil {
		// Optionally record something about err here
		log.Fatal("wireContext Error")
	}

	serverSpan = opentracing.StartSpan(
		appSpecificOperationName,
		ext.RPCServerOption(wireContext))
	serverSpan.SetTag("role", "childspan")
	serverSpan.SetBaggageItem("dump", string(dump))

	defer serverSpan.Finish()

}

func slow(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "This is echo service\n")
	time.Sleep(10 * time.Second)
	io.WriteString(w, "Waited 10 seconds \n")
}

func errorRes(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusServiceUnavailable)

	io.WriteString(w, "This is echo service\n")
	io.WriteString(w, "Error!! \n")
}
