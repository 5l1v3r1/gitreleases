package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/inconshreveable/log15"

	"github.com/gorilla/mux"
)

const requestTimeout = 2 * time.Second

type apiServer struct {
	server       *http.Server
	router       *mux.Router
	githubClient *GithubClient
	logger       log15.Logger
}

func (as *apiServer) Start() error {
	return as.server.ListenAndServe()
}
func (as *apiServer) Shutdown(ctx context.Context) error {
	return as.server.Shutdown(ctx)
}

func (as *apiServer) DownloadRelease(w http.ResponseWriter, r *http.Request) {
	reqLogger := as.logger.New("method", r.Method, "url", r.RequestURI)
	reqLogger.Info("fetching release URL")

	vars := mux.Vars(r)
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	url, err := as.githubClient.FetchReleaseURL(ctx, vars["owner"], vars["repo"], vars["tag"], vars["assetName"])
	if ctx.Err() != nil {
		reqLogger.Error("error retrieving release URL", "err", err, "ctx error", ctx.Err())
		writeHTTPError(w, reqLogger, http.StatusBadGateway, "Bad Gateway")
		return
	}
	if err != nil {
		switch t := err.(type) {
		case GitHubError:
			if t.Type == TypeNotFound {
				reqLogger.Info("data not found", "err", t.WrappedError, vars)
				writeHTTPError(w, reqLogger, http.StatusNotFound, t.WrappedError.Error())
				return
			} else {
				reqLogger.Error("unhandled github error", "err", t.WrappedError, "vars", vars)
				writeHTTPError(w, reqLogger, http.StatusInternalServerError, "Internal Server Error")
				return
			}
		}
		reqLogger.Error("error retrieving release URL", "err", err)
		writeHTTPError(w, reqLogger, http.StatusInternalServerError, err.Error())
		return
	}

	reqLogger.Info("found release URL", "url", url)

	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusMovedPermanently)
}

func writeHTTPError(w http.ResponseWriter, logger log15.Logger, statusCode int, message string) {
	w.WriteHeader(statusCode)
	if _, err := fmt.Fprintln(w, message); err != nil {
		logger.Crit("error writing response", "err", err)
	}
}

func NewAPIServer(addr string, client *GithubClient, logger log15.Logger) *apiServer {
	r := mux.NewRouter()

	as := apiServer{
		server: &http.Server{
			Addr:           addr,
			Handler:        r,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		githubClient: client,
		logger:       logger,
	}

	r.HandleFunc("/gh/{owner}/{repo}/{tag}/{assetName}", as.DownloadRelease)

	return &as
}