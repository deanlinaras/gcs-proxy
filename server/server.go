package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/afiore/gcs-proxy/gcs"
)

//ServeFromBuckets maps incoming requests to bucket objects defined in the supplied configuration
func ServeFromBuckets(bucketByAlias map[string]string, gcpSaPath string, aliasIndexHTML bool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		for alias, bucketName := range bucketByAlias {
			if !strings.HasPrefix(r.URL.Path, "/"+alias) {
				continue
			}
			objectKey := strings.Replace(r.URL.Path, fmt.Sprintf("/%s/", alias), "", 1)

			if aliasIndexHTML {
				objectKey = replaceEmptyBase(objectKey, "index.html")
			}
			log.Printf("Fetching key: %s from bucket %s", objectKey, bucketName)

			gcsObj, err := gcs.GetObject(gcpSaPath, bucketName, objectKey)

			var objNotFoundErr *gcs.ObjectNotFound
			if errors.As(err, &objNotFoundErr) {
				http.NotFound(w, r)
				return
			}
			if err != nil {
				http.Error(w, "An internal error has occured", 500)
				return
			}

			for k, v := range objectHeaders(gcsObj) {
				w.Header().Add(k, v)
			}

			_, err = gcsObj.Copy(w)
			if err != nil {
				log.Fatal(err)
			} else {
				return
			}
		}
	}

}

func base(key string) string {
	parts := strings.Split(key, "/")
	return parts[len(parts)-1]
}

func replaceEmptyBase(key string, replacement string) string {
	if key != "" && base(key) == "" {
		parts := strings.Split(key, "/")
		return strings.Join(append(parts[:len(parts)-1], replacement), "/")

	}
	return key
}

func objectHeaders(o gcs.Object) map[string]string {
	return map[string]string{
		"Content-Type":   o.ContentType,
		"Content-Length": fmt.Sprintf("%d", o.Size),
		"Last-Modified":  o.Updated.Format(http.TimeFormat),
	}
}