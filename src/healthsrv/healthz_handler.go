package healthsrv

import (
    "context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/drycc/minio/src/storage"
	"github.com/minio/minio-go/v7"
)

type healthZResp struct {
	Buckets []minio.BucketInfo `json:"buckets"`
}

func healthZHandler(bucketLister storage.BucketLister) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buckets, err := bucketLister.ListBuckets(context.Background())
		if err != nil {
			str := fmt.Sprintf("Probe error: listing buckets (%s)", err)
			log.Println(str)
			http.Error(w, str, http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(healthZResp{Buckets: buckets}); err != nil {
			str := fmt.Sprintf("Probe error: encoding buckets json (%s)", err)
			log.Println(str)
			http.Error(w, str, http.StatusInternalServerError)
			return
		}
	})
}
