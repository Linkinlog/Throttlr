package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/linkinlog/throttlr/internal"
	"github.com/linkinlog/throttlr/internal/db"
	"github.com/linkinlog/throttlr/internal/models"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title						Throttlr API
// @version					0.0.1
// @description				This is the API for Throttlr, a rate limiting service.
// @BasePath					/v1
// @securityDefinitions.apikey	ApiKeyAuth
// @in							query
// @name						key
var (
	MissingAPIKey         = errors.New("missing API key")
	InvalidAPIKey         = errors.New("invalid API key")
	InvalidEndpointValues = errors.New("invalid endpoint values")
	EndpointExists        = errors.New("endpoint already exists")
	EndpointMissing       = errors.New("endpoint doesnt exist")
	BucketExists          = errors.New("bucket already exists")
)

func apiLogHandler(l *slog.Logger, h HandlerErrorFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpErr := h(w, r)
		if httpErr != nil {
			l.Error("api handler error", "error", httpErr.Error())
			http.Error(w, httpErr.display, http.StatusInternalServerError)
		}
	}
}

func HandleServer(l *slog.Logger, pool *pgx.Conn) *http.ServeMux {
	m := http.NewServeMux()

	m.Handle("/v1/", http.StripPrefix("/v1", serveV1(l, pool)))
	m.Handle("/swagger/*", httpSwagger.WrapHandler)

	return m
}

func serveV1(l *slog.Logger, pool *pgx.Conn) *http.ServeMux {
	m := http.NewServeMux()

	m.Handle("POST /register", apiLogHandler(l, registerEndpoint(pool)))
	m.Handle("POST /update/{throttlrPath}", apiLogHandler(l, updateEndpoint(pool)))
	m.Handle("POST /delete/{throttlrPath}", apiLogHandler(l, deleteEndpoint(pool)))
	m.Handle("/endpoints/{throttlrPath}", apiLogHandler(l, proxyEndpoint(pool)))

	return m
}

// @Summary		Proxy endpoint
// @Description	Users will hit this endpoint to access the proxied endpoint
// @Tags			Proxy
// @Accept			x-www-form-urlencoded
// @Accept			json
// @Produce		plain
// @Produce		json
// @Produce		html
// @Param			throttlrPath	path	string	true	"Throttlr path"
// @Security		ApiKeyAuth
// @Router			/endpoints/{throttlrPath} [get]
// @Router			/endpoints/{throttlrPath} [post]
func proxyEndpoint(pool *pgx.Conn) HandlerErrorFunc {
	ks := db.NewKeyStore(pool)
	es := db.NewEndpointStore(pool)

	return func(w http.ResponseWriter, r *http.Request) *httpError {
		key := r.URL.Query().Get("key")
		exists, apiKeyId := ks.Exists(key)
		if !exists {
			return &httpError{MissingAPIKey, "No API key"}
		}
		if !ks.Valid(apiKeyId) {
			return &httpError{InvalidAPIKey, "Invalid API key"}
		}

		userId, err := ks.UserIdFromKey(key)
		if err != nil {
			return &httpError{err, "failed to get user from key"}
		}

		throttlrPath := r.PathValue("throttlrPath")

		e := &models.Endpoint{ThrottlrPath: throttlrPath, Bucket: &models.Bucket{}}
		if exists, err := es.ExistsByThrottlr(r.Context(), e, userId); !exists {
			if err != nil {
				return &httpError{err, "failed to check if endpoint exists"}
			} else {
				return &httpError{EndpointMissing, "Endpoint doesnt exist"}
			}
		}

		if err := es.Fill(r.Context(), e, userId); err != nil {
			return &httpError{err, "failed to fill endpoint"}
		}

		proxy := httputil.NewSingleHostReverseProxy(e.OriginalUrl)
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			modifyRequest(req, e.OriginalUrl)
		}

		proxy.ServeHTTP(w, r)
		return nil
	}
}

func modifyRequest(r *http.Request, originalUrl *url.URL) {
	r.URL = originalUrl
	r.Host = originalUrl.Host
}

// @Summary		Register endpoint
// @Description	Users will hit this endpoint to register a new endpoint
// @Tags			Register
// @Accept			x-www-form-urlencoded
// @Produce		plain
// @Produce		html
// @Security		ApiKeyAuth
// @Param			endpoint	formData	string	true	"Endpoint to register"
// @Param			interval	formData	int		true	"Interval, 1 = minute, 2 = hour, 3 = day, 4 = week, 5 = month"	Enums(1, 2, 3, 4, 5)
// @Param			max			formData	int		true	"Max requests per interval"
// @Success		201			{string}	string	"Created"
// @Router			/register [post]
func registerEndpoint(pool *pgx.Conn) HandlerErrorFunc {
	ks := db.NewKeyStore(pool)
	es := db.NewEndpointStore(pool)

	return func(w http.ResponseWriter, r *http.Request) *httpError {
		endpoint, httpErr := validateEndpointRequest(r)
		if httpErr != nil {
			return httpErr
		}

		key, httpErr := validateApiKey(r, ks)
		if httpErr != nil {
			return httpErr
		}

		userId, err := ks.UserIdFromKey(key)
		if err != nil {
			return &httpError{err, "failed to get user from key"}
		}

		if exists, err := es.ExistsByOriginal(r.Context(), endpoint, userId); exists {
			if err != nil {
				return &httpError{err, "failed to check if endpoint exists"}
			} else {
				return &httpError{EndpointExists, "Endpoint already exists"}
			}
		}

		_, err = es.Store(r.Context(), endpoint, userId)
		if err != nil {
			return &httpError{err, "failed to store endpoint"}
		}

		proxiedURL := fmt.Sprintf("%s/v1/endpoints/%s?key=%s",
			internal.ServerCallbackURL(),
			endpoint.ThrottlrPath,
			key,
		)

		if r.Header.Get("Hx-Request") == "true" {
			htmlResponse := fmt.Sprintf(
				"Success! Endpoint registered at <a href='%s' target='_blank'>Here</a>",
				proxiedURL,
			)
			w.WriteHeader(http.StatusCreated)

			_, err = w.Write([]byte(htmlResponse))
			if err != nil {
				return &httpError{err, "failed to write response"}
			}
			return nil
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)

		_, err = w.Write([]byte(proxiedURL))
		if err != nil {
			return &httpError{err, "failed to write response"}
		}

		return nil
	}
}

// @Summary		Update endpoint
// @Description	Users will hit this endpoint to update an existing endpoint
// @Tags			Update
// @Accept			x-www-form-urlencoded
// @Produce		plain
// @Produce		html
// @Security		ApiKeyAuth
// @Param			endpoint		formData	string	true	"Updated endpoint"
// @Param			interval		formData	int		true	"Interval, 1 = minute, 2 = hour, 3 = day, 4 = week, 5 = month"	Enums(1, 2, 3, 4, 5)
// @Param			max				formData	int		true	"Max requests per interval"
// @Param			throttlrPath	path		string	true	"Throttlr path"
// @Success		201				{string}	string	"Created"
// @Router			/update/{throttlrPath} [post]
func updateEndpoint(pool *pgx.Conn) HandlerErrorFunc {
	es := db.NewEndpointStore(pool)
	ks := db.NewKeyStore(pool)

	return func(w http.ResponseWriter, r *http.Request) *httpError {
		key, httpErr := validateApiKey(r, ks)
		if httpErr != nil {
			return httpErr
		}

		userId, err := ks.UserIdFromKey(key)
		if err != nil {
			return &httpError{err, "failed to get user from key"}
		}

		if err := r.ParseForm(); err != nil {
			return &httpError{err, "failed to parse form"}
		}

		newEndpoint := r.FormValue("endpoint")
		if newEndpoint == "" {
			return &httpError{InvalidEndpointValues, "Invalid endpoint value"}
		}
		newInterval := r.FormValue("interval")
		newIntervalInt, err := strconv.Atoi(newInterval)
		if newInterval == "" || err != nil {
			return &httpError{InvalidEndpointValues, "Invalid interval value"}
		}
		newMax := r.FormValue("max")
		newMaxInt, err := strconv.Atoi(newMax)
		if newMax == "" || err != nil {
			return &httpError{InvalidEndpointValues, "Invalid max value"}
		}

		throttlrPath := r.PathValue("throttlrPath")
		if throttlrPath == "" {
			return &httpError{InvalidEndpointValues, "Invalid throttlrPath value"}
		}

		endpoint, err := es.Get(r.Context(), throttlrPath, userId)
		if err != nil {
			return &httpError{err, "failed to get endpoint"}
		}

		endpoint.Bucket.Max = newMaxInt
		endpoint.Bucket.Interval = models.Interval(newIntervalInt)
		endpoint.OriginalUrl, err = url.Parse(newEndpoint)
		if err != nil {
			return &httpError{err, "failed to parse new endpoint"}
		}

		if err := es.Update(r.Context(), endpoint, userId); err != nil {
			if strings.Contains(err.Error(), "endpoint already exists") {
				return &httpError{EndpointExists, "Endpoint already exists"}
			}
			return &httpError{err, "failed to update endpoint"}
		}

		proxiedURL := fmt.Sprintf("%s/v1/endpoints/%s?key=%s",
			internal.ServerCallbackURL(),
			endpoint.ThrottlrPath,
			key,
		)

		if r.Header.Get("Hx-Request") == "true" {
			htmlResponse := fmt.Sprintf(
				"Success! Endpoint updated <a href='%s' target='_blank'>Here</a>",
				proxiedURL,
			)
			w.WriteHeader(http.StatusCreated)

			_, err = w.Write([]byte(htmlResponse))
			if err != nil {
				return &httpError{err, "failed to write response"}
			}
			return nil
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)

		_, err = w.Write([]byte(proxiedURL))
		if err != nil {
			return &httpError{err, "failed to write response"}
		}

		return nil
	}
}

// @Summary		Delete endpoint
// @Description	Users will hit this endpoint to delete an existing endpoint
// @Tags			Delete
// @Accept			x-www-form-urlencoded
// @Produce		plain
// @Produce		html
// @Security		ApiKeyAuth
// @Param			throttlrPath	path		string	true	"Throttlr path"
// @Success		200				{string}	string	"Deleted"
// @Router			/delete/{throttlrPath} [post]
func deleteEndpoint(pool *pgx.Conn) HandlerErrorFunc {
	ks := db.NewKeyStore(pool)
	es := db.NewEndpointStore(pool)

	return func(w http.ResponseWriter, r *http.Request) *httpError {
		key, httpErr := validateApiKey(r, ks)
		if httpErr != nil {
			return httpErr
		}

		userId, err := ks.UserIdFromKey(key)
		if err != nil {
			return &httpError{err, "failed to get user from key"}
		}

		throttlrPath := r.PathValue("throttlrPath")
		if throttlrPath == "" {
			return &httpError{InvalidEndpointValues, "Invalid throttlrPath value"}
		}

		e, err := es.Get(r.Context(), throttlrPath, userId)
		if err != nil {
			return &httpError{err, "failed to get endpoint"}
		}

		if err := es.Delete(r.Context(), e, userId); err != nil {
			return &httpError{err, "failed to delete endpoint"}
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("Deleted"))
		if err != nil {
			return &httpError{err, "failed to write response"}
		}
		return nil
	}
}

func validateEndpointRequest(r *http.Request) (*models.Endpoint, *httpError) {
	if err := r.ParseForm(); err != nil {
		return nil, &httpError{err, "failed to parse form"}
	}

	maxTokens, _ := strconv.Atoi(r.FormValue("max"))
	interval, _ := strconv.Atoi(r.FormValue("interval"))
	endpoint := r.FormValue("endpoint")

	if maxTokens == 0 || interval == 0 || endpoint == "" {
		return nil, &httpError{InvalidEndpointValues, "Invalid endpoint values"}
	}
	b := models.NewBucket(models.Interval(interval), maxTokens)
	e, err := models.NewEndpoint(endpoint, b)
	if err != nil {
		return nil, &httpError{err, "failed to create endpoint"}
	}

	return e, nil
}

func validateApiKey(r *http.Request, ks *db.KeyStore) (string, *httpError) {
	key := r.URL.Query().Get("key")
	exists, apiKeyId := ks.Exists(key)
	if !exists {
		return "", &httpError{InvalidAPIKey, "No API key"}
	}
	if !ks.Valid(apiKeyId) {
		return "", &httpError{InvalidAPIKey, "Invalid API key"}
	}
	return key, nil
}
