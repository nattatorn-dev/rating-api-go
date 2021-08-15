package handlers

import (
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func Router() *mux.Router {
	isReady := &atomic.Value{}
	isReady.Store(false)
	go func() {
		log.Printf("Readyz probe is negative by default...")
		time.Sleep(10 * time.Second)
		isReady.Store(true)
		log.Printf("Readyz probe is positive.")
	}()

	redisClient := initialize()
	service := &Service{redis: redisClient}

	r := mux.NewRouter()

	r.HandleFunc("/system/healthz", service.Healthz)
	r.HandleFunc("/system/readyz", Readyz(isReady))
	r.HandleFunc("/registers", service.Register).Methods("POST")
	r.HandleFunc("/registers/{userId}", service.GetUser).Methods("GET")

	r.HandleFunc("/ratings", service.RateUser).Methods("POST")

	r.HandleFunc("/users/ratings/{userId}", service.GetUserRatingById).Methods("GET")
	r.HandleFunc("/users/ratings", service.UserRatingLeaderboard).Methods("GET")

	r.Use(commonMiddleware)
	r.Use(loggingMiddleware)

	return r
}

func loggingMiddleware(next http.Handler) http.Handler {
	return handlers.LoggingHandler(os.Stdout, next)
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "applicationjson")
		next.ServeHTTP(w, r)
	})
}

func Readyz(isReady *atomic.Value) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if isReady == nil || !isReady.Load().(bool) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
