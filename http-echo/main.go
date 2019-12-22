package main

import (
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

func main() {

	cfg := jaegercfg.Configuration{
		ServiceName: "echo service",
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
	}

	// Initialize tracer with a logger and a metrics factory
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}
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
