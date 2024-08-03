package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/linkinlog/throttlr/internal/db"
	"github.com/linkinlog/throttlr/internal/models"
)

var (
	InvalidAPIKey         = errors.New("invalid API key")
	InvalidEndpointValues = errors.New("invalid endpoint values")
	EndpointExists        = errors.New("endpoint already exists")
	BucketExists          = errors.New("bucket already exists")
)

func HandleServer(l *slog.Logger, ks *db.KeyStore, es *db.EndpointStore, bs *db.BucketStore) *http.ServeMux {
	m := http.NewServeMux()

	m.Handle("POST /register/{apiKey}", apiLogHandler(l, registerEndpoint(ks, es, bs)))

	return m
}

func apiLogHandler(l *slog.Logger, h HandlerErrorFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpErr := h(w, r)
		if httpErr != nil {
			l.Error("api handler error", "error", httpErr.Error())
			http.Error(w, httpErr.display, http.StatusInternalServerError)
		}
	}
}

func registerEndpoint(ks *db.KeyStore, es *db.EndpointStore, bs *db.BucketStore) HandlerErrorFunc {
	return func(w http.ResponseWriter, r *http.Request) *httpError {
		key := r.PathValue("apiKey")
		if !ks.Valid(key) {
			return &httpError{InvalidAPIKey, "Invalid API key"}
		}
		r.ParseForm()

		maxTokens, _ := strconv.Atoi(r.FormValue("max"))
		interval, _ := strconv.Atoi(r.FormValue("interval"))
		endpoint := r.FormValue("endpoint")

		if maxTokens == 0 || interval == 0 || endpoint == "" {
			return &httpError{InvalidEndpointValues, "Invalid endpoint values"}
		}

		e := models.NewEndpoint(key, endpoint)
		if exists, err := es.Exists(r.Context(), e); exists {
			if err != nil {
				return &httpError{err, "failed to check if endpoint exists"}
			} else {
				return &httpError{EndpointExists, "Endpoint already exists"}
			}
		}

		id, err := es.Store(r.Context(), e)
		if err != nil {
			return &httpError{err, "failed to store endpoint"}
		}

		b := models.NewBucket(e, models.Interval(interval), maxTokens)
		bDb := db.BucketModel{Bucket: b, EndpointId: id}
		if exists, err := bs.Exists(r.Context(), bDb); exists {
			if err != nil {
				return &httpError{err, "failed to check if bucket exists"}
			} else {
				return &httpError{BucketExists, "Bucket already exists"}
			}
		}

		bs.Store(r.Context(), bDb)

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(r.Host + "/endpoints/" + e.ThrottlrPath))

		return nil
	}
}
