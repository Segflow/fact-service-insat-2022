package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var requestTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "http_request_total",
	Help: "Total number of requests",
}, []string{"status"})

func (a *Service) fact(ctx context.Context, n int) int {
	_, span := a.Tracer.Start(ctx, "fact")
	defer span.End()

	s := 1
	for i := 1; i <= n; i++ {
		s = s * i
		time.Sleep(10 * time.Millisecond)
	}

	return s
}

type Service struct {
	Logger *zap.SugaredLogger
	Tracer trace.Tracer
}

func getRequestID(r *http.Request) string {
	// No request id found.
	hv := r.Header.Get("X-Request-ID")
	if hv != "" {
		return hv
	}

	return uuid.NewString()
}

func (s *Service) handler(w http.ResponseWriter, r *http.Request) {
	rid := getRequestID(r)
	ctx, span := s.Tracer.Start(r.Context(), "handle", trace.WithAttributes(
		attribute.String("request_id", rid),
	))
	defer span.End()

	requestLogger := s.Logger.With("client_ip", r.RemoteAddr, "user_agent", r.UserAgent(), "request_id", rid)

	if r.URL.Query().Get("n") == "" {
		w.WriteHeader(http.StatusBadRequest)
		requestTotal.With(prometheus.Labels{
			"status": "400",
		}).Inc()
		return
	}

	n, err := strconv.Atoi(r.URL.Query().Get("n"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		requestTotal.With(prometheus.Labels{
			"status": "400",
		}).Inc()
		return
	}

	requestLogger.Infow("calculating fact", "n", n)

	res := s.fact(ctx, n)

	json.NewEncoder(w).Encode(map[string]int{
		"response": res,
	})

	requestLogger.Infow("done calculating fact", "n", n)

	requestTotal.With(prometheus.Labels{
		"status": "200",
	}).Inc()
}

func main() {
	logger := buildLogger()

	tracer := buildTracer()

	s := Service{Logger: logger, Tracer: tracer}

	http.HandleFunc("/fact", s.handler)
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Listening on port 8080")
	http.ListenAndServe(":8080", nil)
}

func buildTracer() trace.Tracer {
	// Environment variables

	traceExporter, err := otlptracegrpc.New(context.Background())
	if err != err {
		os.Exit(1)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // 100% of the traces
		sdktrace.WithBatcher(traceExporter),           //
	)

	otel.SetTracerProvider(provider)

	return otel.Tracer("fact-service")
}

func buildLogger() *zap.SugaredLogger {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	l, _ := config.Build()
	return l.Sugar()
}
