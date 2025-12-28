package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	"github.com/CDavidSV/GopherStore/internal/resp"
	"github.com/go-playground/validator/v10"
)

var (
	cacheServerHost = "localhost:5001"
	validate        = validator.New()
)

type Response struct {
	Data resp.RespValue `json:"data"`
}

type SetCommandRequest struct {
	Key           string `json:"key"`
	Value         string `json:"value"`
	ExpireSeconds int    `json:"expiration,omitempty" validate:"omitempty,min=1"`
	Condition     string `json:"condition,omitempty" validate:"omitempty,oneof=NX XX"` // Ensures only NX or XX is used
}

type DeleteCommandRequest struct {
	Keys []string `json:"keys"`
}

type PushCommandRequest struct {
	Key       string   `json:"key"`
	Values    []string `json:"values"`
	Direction string   `json:"direction,omitempty" validate:"omitempty,oneof=left right"`
}

type PopCommandRequest struct {
	Key       string `json:"key"`
	Direction string `json:"direction,omitempty" validate:"omitempty,oneof=left right"`
}

type ExpiresCommandRequest struct {
	Key           string `json:"key"`
	ExpireSeconds int    `json:"expiration" validate:"min=1"`
}

// Makes a request to the cache server and disconnects after receiving a response.
func makeRequest(respString string) (resp.RespValue, error) {
	conn, err := net.Dial("tcp", cacheServerHost)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(respString))
	if err != nil {
		return nil, err
	}

	// Wait for the reply before closing the connection
	reader := bufio.NewReader(conn)
	val, err := resp.ReadRESP(reader)
	if err != nil {
		return nil, err
	}

	if respErr, ok := val.(resp.RespErrorValue); ok {
		return nil, &resp.RESPError{Msg: respErr.Message}
	}

	return val, nil
}

// Route handlers
func handleRoot(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("./ui/html/index.tmpl.html"))
	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleSetCommand(w http.ResponseWriter, r *http.Request) {
	var req SetCommandRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reqArr := [][]byte{
		[]byte("SET"),
		[]byte(req.Key),
		[]byte(req.Value),
	}

	if req.Condition != "" {
		reqArr = append(reqArr, []byte(req.Condition))
	}

	if req.ExpireSeconds > 0 {
		reqArr = append(reqArr, []byte("EX"), []byte(strconv.Itoa(req.ExpireSeconds)))
	}

	cashRes, err := makeRequest(string(resp.EncodeBulkStringArray(reqArr)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stringRes, ok := cashRes.(resp.RespSimpleString)
	if !ok {
		http.Error(w, "Invalid response format", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Data: stringRes.Value})
}

func handleGetCommand(w http.ResponseWriter, r *http.Request) {
	// Get the ket from query params
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Missing 'key' query parameter", http.StatusBadRequest)
		return
	}

	cashRes, err := makeRequest(string(resp.EncodeBulkStringArray([][]byte{
		[]byte("GET"),
		[]byte(key),
	})))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stringRes, ok := cashRes.(resp.RespBulkString)
	if ok && stringRes.Value == nil {
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Data: string(stringRes.Value)})
}

func handleDeleteCommand(w http.ResponseWriter, r *http.Request) {
	var req DeleteCommandRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reqArr := make([][]byte, len(req.Keys)+1)
	reqArr[0] = []byte("DEL")
	for i, k := range req.Keys {
		reqArr[i+1] = []byte(k)
	}
	cashRes, err := makeRequest(string(resp.EncodeBulkStringArray(reqArr)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stringRes, ok := cashRes.(resp.RespInteger)
	if !ok {
		http.Error(w, "Invalid response format", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Data: stringRes.Value})
}

func handlePushCommand(w http.ResponseWriter, r *http.Request) {
	var req PushCommandRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reqArr := make([][]byte, len(req.Values)+2)
	if req.Direction == "left" {
		reqArr[0] = []byte("LPUSH")
	} else {
		reqArr[0] = []byte("RPUSH")
	}

	reqArr[1] = []byte(req.Key)

	for i, val := range req.Values {
		reqArr[i+2] = []byte(val)
	}
	cashRes, err := makeRequest(string(resp.EncodeBulkStringArray(reqArr)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stringRes, ok := cashRes.(resp.RespInteger)
	if !ok {
		http.Error(w, "Invalid response format", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Data: stringRes.Value})
}

func handlePopCommand(w http.ResponseWriter, r *http.Request) {
	var req PopCommandRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var cmd string
	if req.Direction == "left" {
		cmd = "LPOP"
	} else {
		cmd = "RPOP"
	}

	cashRes, err := makeRequest(string(resp.EncodeBulkStringArray([][]byte{
		[]byte(cmd),
		[]byte(req.Key),
	})))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stringRes, ok := cashRes.(resp.RespBulkString)
	if !ok {
		http.Error(w, "Invalid response format", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Data: string(stringRes.Value)})
}

func handleLLenCommand(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Missing 'key' query parameter", http.StatusBadRequest)
		return
	}

	cashRes, err := makeRequest(string(resp.EncodeBulkStringArray([][]byte{
		[]byte("LLEN"),
		[]byte(key),
	})))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	intRes, ok := cashRes.(resp.RespInteger)
	if !ok {
		http.Error(w, "Invalid response format", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Data: intRes.Value})
}

func handleLRangeCommand(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Missing 'key' query parameter", http.StatusBadRequest)
		return
	}

	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	if startStr == "" || endStr == "" {
		http.Error(w, "Missing 'start' or 'end' query parameter", http.StatusBadRequest)
		return
	}

	cashRes, err := makeRequest(string(resp.EncodeBulkStringArray([][]byte{
		[]byte("LRANGE"),
		[]byte(key),
		[]byte(startStr),
		[]byte(endStr),
	})))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respArr, ok := cashRes.(resp.RespArray)
	if !ok {
		http.Error(w, "Invalid response format", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if respArr.Elements == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{Data: nil})
		return
	}

	stringRes := make([]string, len(respArr.Elements))
	for i, elem := range respArr.Elements {
		if bulkStr, ok := elem.(resp.RespBulkString); ok {
			if bulkStr.Value != nil {
				stringRes[i] = string(bulkStr.Value)
			} else {
				stringRes[i] = ""
			}
		} else {
			http.Error(w, "Invalid response element format", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Data: stringRes})
}

func handleExpiresCommand(w http.ResponseWriter, r *http.Request) {
	var req ExpiresCommandRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cashRes, err := makeRequest(string(resp.EncodeBulkStringArray([][]byte{
		[]byte("EXPIRE"),
		[]byte(req.Key),
		[]byte(strconv.Itoa(req.ExpireSeconds)),
	})))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	intRes, ok := cashRes.(resp.RespInteger)
	if !ok {
		http.Error(w, "Invalid response format", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Data: intRes.Value})
}

func main() {
	addr := flag.String("addr", "localhost:3000", "HTTP network address")
	cacheAddr := flag.String("cache-addr", "localhost:5001", "Cache server network address")
	flag.Parse()

	cacheServerHost = *cacheAddr

	mux := http.NewServeMux()

	// Static files
	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// routes
	mux.HandleFunc("GET /", handleRoot)
	mux.HandleFunc("POST /set", handleSetCommand)
	mux.HandleFunc("GET /get", handleGetCommand)
	mux.HandleFunc("POST /delete", handleDeleteCommand)
	mux.HandleFunc("POST /push", handlePushCommand)
	mux.HandleFunc("POST /pop", handlePopCommand)
	mux.HandleFunc("GET /llen", handleLLenCommand)
	mux.HandleFunc("GET /lrange", handleLRangeCommand)
	mux.HandleFunc("POST /expires", handleExpiresCommand)

	slog.Info("Starting server", "addr", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
