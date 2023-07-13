package workflows

// This file implements a mock server that can be used to test workflows.

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"log"

	"github.com/gorilla/mux"
)

type liveHooksReq struct {
	SenderIp   string `json:"sender_ip"`
	SenderPort int    `json:"sender_port"`
}

type videoParams struct {
	VvideoWidth          int `json:"video_width",omitempty`
	VideoHeight          int `json:"video_height",omitempty`
	FrameRateNumerator   int `json:"frame_rate_numerator",omitempty`
	FrameRateDenominator int `json:"frame_rate_denominator",omitempty`
}

type mediaStreamInputParams struct {
	VideoParams videoParams `json:"video_params"`
}

type Meta struct {
	ResourceId      string `json:"resource_id",omitempty`
	ClientRequestId string `json:"client_request_id",omitempty`
	WorkflowId      string `json:"workflow_id",omitempty`
	ActivityName    string `json:"activity_name",omitempty`
	Status          string `json:"status",omitempty`
}

type liveHooksResp struct {
	Meta                   Meta                   `json:"meta"`
	MediaStreamInputParams mediaStreamInputParams `json:"media_stream_input_params"`
}

type hlsAbrSettings struct {
	Variants []videoParams `json:"variants"`
}

type mediaStreamToAbrConverterReq struct {
	MediaStreamInputParams mediaStreamInputParams `json:"media_stream_input_params"`
	HlsAbrSettings         hlsAbrSettings         `json:"hls_abr_settings"`
}

type mediaStreamToAbrConverterResp struct {
	Meta Meta `json:"meta"`
	mediaStreamToAbrConverterReq
}

var liveHooksStore = map[string]liveHooksResp{}
var mediaStreamToAbrConverterStore = map[string]mediaStreamToAbrConverterResp{}

func randomStringCreate(len int) string {
	b := make([]byte, len)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func liveHooksCreate(w http.ResponseWriter, r *http.Request) {
	// parse request body into liveHooksReq
	var req liveHooksReq

	// read r.Body into a byte array
	reqBuf := make([]byte, r.ContentLength)
	r.Body.Read(reqBuf)
	log.Println("liveHooksCreate: ", string(reqBuf))
	err := json.Unmarshal(reqBuf, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get http headers key and values from r
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = v[0]
	}

	response := liveHooksResp{}

	// initialize response with some mock data
	response.Meta.ResourceId = randomStringCreate(5)
	response.Meta.WorkflowId = headers["x-workflow-id"]
	response.Meta.ActivityName = headers["x-activity-name"]
	response.Meta.Status = "pending"
	response.MediaStreamInputParams.VideoParams.VvideoWidth = 1920
	response.MediaStreamInputParams.VideoParams.VideoHeight = 1080
	response.MediaStreamInputParams.VideoParams.FrameRateNumerator = 30
	response.MediaStreamInputParams.VideoParams.FrameRateDenominator = 1

	// store response in liveHooksStore
	liveHooksStore[response.Meta.ResourceId] = response

	// Simulate live_hook resource creation using sleep
	time.Sleep(5 * time.Second)

	response.Meta.Status = "created"
	// store response in liveHooksStore
	liveHooksStore[response.Meta.ResourceId] = response

	// write response to w
	respBuf, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(respBuf)
}

func liveHooksGet(w http.ResponseWriter, r *http.Request) {
	// Get resource id from url
	vars := mux.Vars(r)
	resourceId := vars["id"]

	liveHooksResp := liveHooksStore[resourceId]

	// write response to w
	respBuf, err := json.Marshal(liveHooksResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(respBuf)
}

func liveHooksGetWithQuery(w http.ResponseWriter, r *http.Request) {
	// Get query params from url
	queryParams := r.URL.Query()

	log.Println("RequestUri: ", r.RequestURI)

	for k, v := range queryParams {
		log.Printf("Key: %s, Value: %s", k, v)
	}

	workflowId := queryParams.Get("workflow_id")
	activityName := queryParams.Get("activity_name")

	liveHooksResp := liveHooksResp{}

	found_resource := false
	for _, v := range liveHooksStore {
		if v.Meta.WorkflowId == workflowId && v.Meta.ActivityName == activityName {
			liveHooksResp = v
			found_resource = true
			break
		}
	}

	if !found_resource {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	} else {
		// write response to w
		respBuf, err := json.Marshal(liveHooksResp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(respBuf)
	}
}

func mediaStreamToAbrConverterCreate(w http.ResponseWriter, r *http.Request) {
	// parse request body into mediaStreamToAbrConverterReq
	var req mediaStreamToAbrConverterReq

	// read r.Body into a byte array
	reqBuf := make([]byte, r.ContentLength)
	r.Body.Read(reqBuf)
	log.Println("mediaStreamToAbrConverterCreate: ", string(reqBuf))
	err := json.Unmarshal(reqBuf, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get http headers key and values from r
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = v[0]
	}

	response := mediaStreamToAbrConverterResp{}
	response.Meta.ResourceId = randomStringCreate(5)
	response.Meta.ClientRequestId = headers["x-request-id"]
	response.Meta.WorkflowId = headers["x-workflow-id"]
	response.Meta.ActivityName = headers["x-activity-name"]
	response.Meta.Status = "pending"
	response.mediaStreamToAbrConverterReq = req

	// store response in mediaStreamToAbrConverterStore
	mediaStreamToAbrConverterStore[response.Meta.ResourceId] = response

	// Simulate media_stream_to_abr_converter backend resource creation using sleep
	time.Sleep(20 * time.Second)

	response.Meta.Status = "created"

	mediaStreamToAbrConverterStore[response.Meta.ResourceId] = response

	// write response to w
	respBuf, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(respBuf)
}

func mediaStreamToAbrConverterGet(w http.ResponseWriter, r *http.Request) {
	// Get resource id from url
	vars := mux.Vars(r)
	resourceId := vars["id"]

	mediaStreamToAbrConverterResp := mediaStreamToAbrConverterStore[resourceId]

	// write response to w
	respBuf, err := json.Marshal(mediaStreamToAbrConverterResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(respBuf)
}

func mediaStreamToAbrConverterGetWithQuery(w http.ResponseWriter, r *http.Request) {
	// Get query params from url
	clientReqId := r.Header.Get("x-request-id")

	mediaStreamToAbrConverterResp := mediaStreamToAbrConverterResp{}

	found_resource := false
	for _, v := range mediaStreamToAbrConverterStore {
		if v.Meta.ClientRequestId == clientReqId {
			mediaStreamToAbrConverterResp = v
			found_resource = true
			break
		}
	}

	if !found_resource {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	} else {
		// write response to w
		respBuf, err := json.Marshal(mediaStreamToAbrConverterResp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(respBuf)
	}
}

func initMockServer() {
	router := mux.NewRouter()
	router.HandleFunc("/live_hooks", liveHooksCreate).Methods("POST")
	router.HandleFunc("/live_hooks/{id}", liveHooksGet).Methods("GET")
	router.HandleFunc("/live_hooks", liveHooksGetWithQuery).Methods("GET")
	router.HandleFunc("/media_stream_to_abr_converter", mediaStreamToAbrConverterCreate).Methods("POST")
	router.HandleFunc("/media_stream_to_abr_converter/{id}", mediaStreamToAbrConverterGet).Methods("GET")
	router.HandleFunc("/media_stream_to_abr_converter", mediaStreamToAbrConverterGetWithQuery).Methods("GET")

	log.Println("Starting mock server on port 9200")

	http.ListenAndServe(":9200", router)
}
