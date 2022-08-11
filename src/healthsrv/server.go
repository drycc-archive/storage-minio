package healthsrv

import (
	"fmt"
	"net/http"

	"github.com/drycc/storage/src/storage"
)

const (
	DefaultHost = "0.0.0.0"
	DefaultPort = 8082
)

// Start starts the healthcheck server on $host:$port and blocks. It only returns if the server fails, with the indicative error
func Start(host string, port int, healthChecker storage.HealthChecker) error {
	mux := http.NewServeMux()
	mux.Handle("/healthz", healthZHandler(healthChecker))

	hostStr := fmt.Sprintf("%s:%d", host, port)
	return http.ListenAndServe(hostStr, mux)
}
