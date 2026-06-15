package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const RentProtocolID protocol.ID = "/meshweb/rent/1.0.0"

type RentalJob struct {
	JobID         string  `json:"job_id"`
	ProviderID    string  `json:"provider_id"`
	BuyerID       string  `json:"buyer_id"`
	CPUCores      int     `json:"cpu_cores"`
	RAMGB         int     `json:"ram_gb"`
	HasGPU        bool    `json:"has_gpu"`
	DurationHours int     `json:"duration_hours"`
	CostPerHour   float64 `json:"cost_per_hour"`
	TotalCost     float64 `json:"total_cost"`
	SpentSoFar    float64 `json:"spent_so_far"`
	StartTime     time.Time `json:"start_time"`
	IsActive      bool    `json:"is_active"`
	IsProvider    bool    `json:"-"` // false = buyer, true = provider
}

type RentRequest struct {
	JobID         string `json:"job_id"`
	CPUCores      int    `json:"cpu_cores"`
	RAMGB         int    `json:"ram_gb"`
	HasGPU        bool   `json:"has_gpu"`
	DurationHours int    `json:"duration_hours"`
	CostPerHour   float64 `json:"cost_per_hour"`
}

type RentResponse struct {
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason,omitempty"`
}

func (a *App) setupRentHandler() {
	a.node.SetStreamHandler(RentProtocolID, func(s network.Stream) {
		a.logEvent("[Rent] Received rental request stream...")
		scanner := bufio.NewScanner(s)
		if !scanner.Scan() {
			s.Close()
			return
		}

		var req RentRequest
		err := json.Unmarshal(scanner.Bytes(), &req)
		if err != nil {
			s.Close()
			return
		}

		a.mu.Lock()
		offer := a.offerResources
		a.mu.Unlock()

		res := RentResponse{Accepted: false}
		if !offer {
			res.Reason = "Node is not offering resources"
		} else {
			res.Accepted = true
			a.logEvent(fmt.Sprintf("[Rent] Accepted rental job %s from %s ✅", req.JobID[:8], s.Conn().RemotePeer().String()[:8]))
			
			// Save job
			job := &RentalJob{
				JobID:         req.JobID,
				BuyerID:       s.Conn().RemotePeer().String(),
				ProviderID:    a.myPeerID,
				CPUCores:      req.CPUCores,
				RAMGB:         req.RAMGB,
				HasGPU:        req.HasGPU,
				DurationHours: req.DurationHours,
				CostPerHour:   req.CostPerHour,
				StartTime:     time.Now(),
				IsActive:      true,
				IsProvider:    true,
			}
			a.mu.Lock()
			a.activeRentals[req.JobID] = job
			a.mu.Unlock()
		}

		b, _ := json.Marshal(res)
		b = append(b, '\n')
		s.Write(b)
		s.Close()
	})
}



func (a *App) StartRental(nodeId string, cpuCores int, ramGB int, hasGPU bool, durationHours int) map[string]interface{} {
	// 1. Narx hisoblash
	gpuCost := 0.0
	if hasGPU {
		gpuCost = 0.5 // Basic GPU simulation
	}
	costPerHour := float64(cpuCores)*0.1 + float64(ramGB)*0.05 + gpuCost

	// 2. Stream ochish
	pid, err := peer.Decode(nodeId)
	if err != nil {
		return map[string]interface{}{"success": false, "error": "Invalid Node ID"}
	}

	jobId := uuid.New().String()

	if nodeId != a.myPeerID {
		a.logEvent(fmt.Sprintf("[Rent] Sending request to %s...", nodeId[:8]))
		stream, err := a.node.NewStream(a.ctx, pid, RentProtocolID)
		if err != nil {
			return map[string]interface{}{"success": false, "error": "Failed to connect to node"}
		}
		defer stream.Close()

		req := RentRequest{
			JobID:         jobId,
			CPUCores:      cpuCores,
			RAMGB:         ramGB,
			HasGPU:        hasGPU,
			DurationHours: durationHours,
			CostPerHour:   costPerHour,
		}
		b, _ := json.Marshal(req)
		b = append(b, '\n')
		stream.Write(b)

		scanner := bufio.NewScanner(stream)
		if !scanner.Scan() {
			return map[string]interface{}{"success": false, "error": "Node did not respond"}
		}

		var res RentResponse
		json.Unmarshal(scanner.Bytes(), &res)
		if !res.Accepted {
			return map[string]interface{}{"success": false, "error": "Node rejected: " + res.Reason}
		}
	}

	a.logEvent(fmt.Sprintf("[Rent] Rental started ✅ Job: %s", jobId[:8]))

	job := &RentalJob{
		JobID:         jobId,
		ProviderID:    nodeId,
		BuyerID:       a.myPeerID,
		CPUCores:      cpuCores,
		RAMGB:         ramGB,
		HasGPU:        hasGPU,
		DurationHours: durationHours,
		CostPerHour:   costPerHour,
		TotalCost:     costPerHour * float64(durationHours),
		StartTime:     time.Now(),
		IsActive:      true,
		IsProvider:    false,
	}

	a.mu.Lock()
	a.activeRentals[jobId] = job
	a.mu.Unlock()

	return map[string]interface{}{
		"success": true,
		"jobId":   jobId,
		"cost":    costPerHour,
	}
}

func (a *App) StopRental(jobId string) bool {
	a.mu.Lock()
	job, ok := a.activeRentals[jobId]
	if ok {
		job.IsActive = false
	}
	a.mu.Unlock()
	a.logEvent(fmt.Sprintf("[Rent] Job %s stopped.", jobId[:8]))
	return ok
}

func (a *App) GetRentalStatus(jobId string) map[string]interface{} {
	a.mu.Lock()
	defer a.mu.Unlock()

	job, ok := a.activeRentals[jobId]
	if !ok {
		return map[string]interface{}{"success": false, "error": "Not found"}
	}

	elapsed := time.Since(job.StartTime).Hours()
	if elapsed > float64(job.DurationHours) {
		elapsed = float64(job.DurationHours)
		job.IsActive = false
	}
	job.SpentSoFar = elapsed * job.CostPerHour

	return map[string]interface{}{
		"success":  true,
		"job":      job,
		"elapsed":  elapsed,
	}
}

func (a *App) rentBillingLoop() {
	// Compute Coming Soon
	// Real billing keyinroq
	return
}
