package main

import (
	"log"

	"net/http"
	"os"
	"strconv"

	"golang.org/x/time/rate"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/ochttp/propagation/tracecontext"
	"go.opencensus.io/trace"
)

var (
	requestDurationsHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "request_durations_histogram_secs",
		Buckets: prometheus.DefBuckets,
		Help:    "Requests Durations, in Seconds",
	})
)

func init() {
	prometheus.MustRegister(requestDurationsHistogram)
}

func instrumentHandler(
	handler http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			t := prometheus.NewTimer(requestDurationsHistogram)
			defer t.ObserveDuration()
			handler.ServeHTTP(w, r)
		},
	)
}

func main() {
	rpsLimitStr := os.Getenv("RPS_THRESHOLD")
	rpsLimit, err := strconv.ParseFloat(rpsLimitStr, 64)
	if err != nil {
		log.Fatalf("bad value for rps limit: %s", rpsLimitStr)
	}

	throttledHandler := throttler(
		rpsLimit,
		http.FileServer(http.Dir("/app/content")),
	)
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/", instrumentHandler(throttledHandler))

	appInsightEnabledStr := os.Getenv("APP_INSIGHT_ENABLED")
	var handler http.Handler
	if appInsightEnabledStr == "true" {
		serviceName := os.Getenv("SERVICE_NAME")
		if len(serviceName) == 0 {
			serviceName = "go-app"
		}
		log.Printf("new ocagent named %s", serviceName)
		exporter, err := ocagent.NewExporter(
			ocagent.WithInsecure(),
			ocagent.WithServiceName(serviceName),
		)
		if err != nil {
			log.Fatal("Failed to create the agent exporter: %v", err)
		}

		trace.RegisterExporter(exporter)
		// Always trace for this demo. In a production application, you should
		// configure this to a trace.ProbabilitySampler set at the desired
		// probability.
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
		handler = &ochttp.Handler{
			Propagation:      &tracecontext.HTTPFormat{},
			IsPublicEndpoint: true,
		}

	}
	log.Fatal(http.ListenAndServe(":8080", handler))

}

func throttler(
	limit float64,
	handler http.Handler,
) http.Handler {
	limiter := rate.NewLimiter(rate.Limit(limit), 10)
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			limiter.Wait(r.Context())
			handler.ServeHTTP(w, r)
		},
	)
}
