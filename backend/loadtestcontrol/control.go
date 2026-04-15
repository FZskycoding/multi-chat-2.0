package loadtestcontrol

import (
	"encoding/json"
	"net/http"
	"sync"
)

type barrierState struct {
	mu               sync.Mutex
	ExpectedClients  int  `json:"expectedClients"`
	ConnectedClients int  `json:"connectedClients"`
	Open             bool `json:"open"`
	AutoOpen         bool `json:"autoOpen"`
}

type resetRequest struct {
	ExpectedClients int  `json:"expectedClients"`
	AutoOpen        bool `json:"autoOpen"`
}

var barrier = &barrierState{}

func Reset(expectedClients int, autoOpen bool) {
	barrier.mu.Lock()
	defer barrier.mu.Unlock()

	barrier.ExpectedClients = expectedClients
	barrier.ConnectedClients = 0
	barrier.Open = false
	barrier.AutoOpen = autoOpen
}

func OpenBarrier() {
	barrier.mu.Lock()
	defer barrier.mu.Unlock()
	barrier.Open = true
}

func SetConnectedClients(count int) bool {
	barrier.mu.Lock()
	defer barrier.mu.Unlock()

	barrier.ConnectedClients = count
	if barrier.AutoOpen && !barrier.Open && barrier.ExpectedClients > 0 && count >= barrier.ExpectedClients {
		barrier.Open = true
		return true
	}

	return false
}

func Snapshot() barrierState {
	barrier.mu.Lock()
	defer barrier.mu.Unlock()

	return barrierState{
		ExpectedClients:  barrier.ExpectedClients,
		ConnectedClients: barrier.ConnectedClients,
		Open:             barrier.Open,
		AutoOpen:         barrier.AutoOpen,
	}
}

func HandleBarrierStatus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(Snapshot())
}

func HandleBarrierReset(w http.ResponseWriter, r *http.Request) {
	var req resetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ExpectedClients < 0 {
		http.Error(w, "expectedClients must be >= 0", http.StatusBadRequest)
		return
	}

	Reset(req.ExpectedClients, req.AutoOpen)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(Snapshot())
}

func HandleBarrierOpen(w http.ResponseWriter, _ *http.Request) {
	OpenBarrier()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(Snapshot())
}
