// Package handlers provides HTTP handlers for the ingestion API.
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"

	"github.com/drfirst/go-oec/internal/api/middleware"
	"github.com/drfirst/go-oec/internal/domain/prescription"
	fhir "github.com/drfirst/go-oec/internal/fhir/r5"
	"github.com/drfirst/go-oec/internal/ncpdp/mapper"
)

// PrescriptionHandler handles prescription endpoints
type PrescriptionHandler struct {
	repo   *prescription.Repository
	mapper *mapper.FHIRToScriptMapper
	logger *zap.Logger
	tracer interface {
		Start(interface{}, string, ...interface{}) (interface{}, interface{})
	}
}

// NewPrescriptionHandler creates a new handler
func NewPrescriptionHandler(repo *prescription.Repository, logger *zap.Logger) *PrescriptionHandler {
	return &PrescriptionHandler{
		repo:   repo,
		mapper: mapper.NewFHIRToScriptMapper(),
		logger: logger,
	}
}

// Routes returns the handler routes
func (h *PrescriptionHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Get("/{id}/events", h.GetEvents)
	r.Post("/{id}/route", h.Route)
	return r
}

// CreateRequest is the request body for creating a prescription
type CreateRequest struct {
	MedicationRequest *fhir.MedicationRequest `json:"medicationRequest"`
	Patient           *fhir.Patient           `json:"patient"`
	Practitioner      *fhir.Practitioner      `json:"practitioner"`
	Pharmacy          *fhir.Organization      `json:"pharmacy,omitempty"`
}

// CreateResponse is the response for creating a prescription
type CreateResponse struct {
	ID             string    `json:"id"`
	Status         string    `json:"status"`
	IdempotencyKey string    `json:"idempotency_key"`
	IsControlled   bool      `json:"is_controlled"`
	CreatedAt      time.Time `json:"created_at"`
}

// Create handles POST /prescriptions
func (h *PrescriptionHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tracer := otel.Tracer("prescription-handler")
	ctx, span := tracer.Start(ctx, "create_prescription")
	defer span.End()

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.MedicationRequest == nil {
		h.jsonError(w, "medicationRequest is required", http.StatusBadRequest)
		return
	}

	// Generate prescription ID
	prescriptionID := uuid.New().String()
	span.SetAttributes(attribute.String("prescription_id", prescriptionID))

	// Map to SCRIPT for validation and extract metadata
	mapResult, err := h.mapper.MapToNewRx(req.MedicationRequest, req.Patient, req.Practitioner, req.Pharmacy)
	if err != nil {
		h.logger.Error("mapping failed", zap.Error(err))
		h.jsonError(w, "failed to process prescription: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Create aggregate
	agg := prescription.NewAggregate(prescriptionID)

	// Build creation data
	fhirPayload, _ := json.Marshal(req.MedicationRequest)
	qty, _ := req.MedicationRequest.GetQuantity()

	createData := &prescription.PrescriptionCreatedData{
		PrescriptionID:      prescriptionID,
		PatientHash:         mapResult.PatientHash,
		PrescriberNPI:       req.MedicationRequest.GetPrescriberNPI(),
		PrescriberDEA:       req.MedicationRequest.GetPrescriberDEA(),
		MedicationNDC:       req.MedicationRequest.GetNDC(),
		MedicationRxNorm:    req.MedicationRequest.GetRxNorm(),
		MedicationName:      req.MedicationRequest.GetMedicationDisplay(),
		IsControlled:        mapResult.IsControlled,
		DEASchedule:         mapResult.DEASchedule,
		Quantity:            qty,
		DaysSupply:          req.MedicationRequest.GetDaysSupply(),
		RefillsAllowed:      req.MedicationRequest.GetRefillsAllowed(),
		SigText:             req.MedicationRequest.GetSigText(),
		SubstitutionAllowed: req.MedicationRequest.IsSubstitutionAllowed(),
		FHIRPayload:         fhirPayload,
		WrittenDate:         req.MedicationRequest.AuthoredOn,
	}

	if err := agg.Create(createData); err != nil {
		h.logger.Error("aggregate create failed", zap.Error(err))
		h.jsonError(w, "failed to create prescription", http.StatusInternalServerError)
		return
	}

	// Save to event store
	if err := h.repo.Save(ctx, agg); err != nil {
		h.logger.Error("save failed", zap.Error(err))
		h.jsonError(w, "failed to save prescription", http.StatusInternalServerError)
		return
	}

	h.logger.Info("prescription created",
		zap.String("id", prescriptionID),
		zap.String("request_id", middleware.GetRequestID(ctx)),
		zap.Bool("controlled", mapResult.IsControlled),
	)

	resp := CreateResponse{
		ID:             prescriptionID,
		Status:         string(agg.Status()),
		IdempotencyKey: mapResult.IdempotencyKey,
		IsControlled:   mapResult.IsControlled,
		CreatedAt:      time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// Get handles GET /prescriptions/{id}
func (h *PrescriptionHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	agg, err := h.repo.Load(ctx, id)
	if err != nil {
		h.jsonError(w, "prescription not found", http.StatusNotFound)
		return
	}

	resp := map[string]interface{}{
		"id":      agg.ID(),
		"status":  agg.Status(),
		"version": agg.Version(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetEvents handles GET /prescriptions/{id}/events
func (h *PrescriptionHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	events, err := h.repo.GetEvents(ctx, id)
	if err != nil {
		h.jsonError(w, "failed to get events", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// RouteRequest is the request for routing a prescription
type RouteRequest struct {
	PharmacyNCPDPID string `json:"pharmacy_ncpdp_id"`
	PharmacyName    string `json:"pharmacy_name"`
}

// Route handles POST /prescriptions/{id}/route
func (h *PrescriptionHandler) Route(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	var req RouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	agg, err := h.repo.Load(ctx, id)
	if err != nil {
		h.jsonError(w, "prescription not found", http.StatusNotFound)
		return
	}

	if err := agg.Route(req.PharmacyNCPDPID, req.PharmacyName); err != nil {
		h.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.Save(ctx, agg); err != nil {
		h.jsonError(w, "failed to save", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"id":       agg.ID(),
		"status":   agg.Status(),
		"pharmacy": req.PharmacyNCPDPID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *PrescriptionHandler) jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
