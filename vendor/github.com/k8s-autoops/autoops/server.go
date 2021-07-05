package autoops

import (
	"context"
	"encoding/json"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	admissionv1 "k8s.io/api/admission/v1"
)

const (
	AdmissionServerCertFile = "/autoops-data/admission-server/tls.crt"
	AdmissionServerKeyFile  = "/autoops-data/admission-server/tls.key"
)

func NewMutatingAdmissionHTTPHandler(
	fn func(ctx context.Context, request *admissionv1.AdmissionRequest, patches *[]map[string]interface{}) (deny string, err error),
) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		var err error
		defer func(err *error) {
			if *err != nil {
				log.Println("failed to handle mutating admission review:", (*err).Error())
				http.Error(rw, (*err).Error(), http.StatusServiceUnavailable)
			}
		}(&err)
		// decode request
		var review admissionv1.AdmissionReview
		if err = json.NewDecoder(req.Body).Decode(&review); err != nil {
			err = fmt.Errorf("failed to decode AdmissionReview: %s", err.Error())
			return
		}
		// logging
		log.Println("Request:")
		raw, _ := json.Marshal(&review)
		log.Println(string(raw))
		// execute fn
		var deny string
		var patches []map[string]interface{}
		if deny, err = fn(req.Context(), review.Request, &patches); err != nil {
			err = fmt.Errorf("failed to execute handler: %s", err.Error())
			return
		}
		// logging
		log.Println("Patches:")
		if len(patches) == 0 {
			log.Println("-NONE-")
		} else {
			raw, _ = json.Marshal(patches)
			log.Println(string(raw))
		}
		// build response
		var responsePatch []byte
		var responsePatchType *admissionv1.PatchType
		if len(patches) != 0 {
			if responsePatch, err = json.Marshal(patches); err != nil {
				err = fmt.Errorf("failed to marshal patches: %s", err.Error())
				return
			}
			responsePatchType = new(admissionv1.PatchType)
			*responsePatchType = admissionv1.PatchTypeJSONPatch
		}
		// send response
		var status *metav1.Status
		if deny != "" {
			status = &metav1.Status{
				Status:  metav1.StatusFailure,
				Message: deny,
				Reason:  metav1.StatusReasonBadRequest,
			}
		}

		var responseJSON []byte
		if responseJSON, err = json.Marshal(admissionv1.AdmissionReview{
			TypeMeta: review.TypeMeta,
			Response: &admissionv1.AdmissionResponse{
				UID:       review.Request.UID,
				Allowed:   deny == "",
				Result:    status,
				Patch:     responsePatch,
				PatchType: responsePatchType,
			},
		}); err != nil {
			err = fmt.Errorf("failed to marshal response json: %s", err.Error())
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.Header().Set("Content-Length", strconv.Itoa(len(responseJSON)))
		_, _ = rw.Write(responseJSON)
	}
}

func ListenAndServeAdmission(s *http.Server) (err error) {
	log.Println("listening at :443")
	return s.ListenAndServeTLS(AdmissionServerCertFile, AdmissionServerKeyFile)
}

func RunAdmissionServer(s *http.Server) (err error) {
	chErr := make(chan error, 1)
	chSig := make(chan os.Signal, 1)
	signal.Notify(chSig, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		chErr <- ListenAndServeAdmission(s)
	}()

	select {
	case err = <-chErr:
	case sig := <-chSig:
		log.Println("signal caught:", sig.String())
		_ = s.Shutdown(context.Background())
	}

	return
}
