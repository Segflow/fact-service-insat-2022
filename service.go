package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var requestTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "http_request_total",
	Help: "Total number of requests",
}, []string{"status"})

func fact(n int) int {
	s := 1
	for i := 1; i <= n; i++ {
		s = s * i
		time.Sleep(10 * time.Millisecond)
	}

	return s
}

func handler(w http.ResponseWriter, r *http.Request) {
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

	res := fact(n)

	json.NewEncoder(w).Encode(map[string]int{
		"response": res,
	})

	requestTotal.With(prometheus.Labels{
		"status": "200",
	}).Inc()
}

func main() {
	http.HandleFunc("/fact", handler)
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
