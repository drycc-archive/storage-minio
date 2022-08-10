package healthsrv

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/drycc/storage/src/storage"
)

func healthZHandler(healthChecker storage.HealthChecker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := healthChecker.HealthCheck(9 * time.Second)
		if err != nil {
			str := fmt.Sprintf("Probe error: listing buckets (%s)", err)
			log.Println(str)
			http.Error(w, str, http.StatusInternalServerError)
			return
		}
	})
}
