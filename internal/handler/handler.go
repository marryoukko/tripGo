package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	api "tripgo.local/internal/generated"
	"tripgo.local/internal/repository"
	"tripgo.local/internal/service"
)

type Handler struct {
	trips *service.TripService
	pool  *pgxpool.Pool
}

func New(trips *service.TripService, pool *pgxpool.Pool) *Handler {
	return &Handler{trips: trips, pool: pool}
}

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	var body api.TripData
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid JSON")
		return
	}

	if body.UserId == (uuid.UUID{}) || body.DriverId == (uuid.UUID{}) {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "user_id and driver_id are required")
		return
	}
	if body.Price < 0 {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "price must be >= 0")
		return
	}
	if !validCoords(body.StartPoint) || !validCoords(body.EndPoint) {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid coordinates")
		return
	}

	in := service.CreateTripInput{
		UserID:   uuid.UUID(body.UserId),
		DriverID: uuid.UUID(body.DriverId),
		StartLat: body.StartPoint.Latitude,
		StartLon: body.StartPoint.Longitude,
		EndLat:   body.EndPoint.Latitude,
		EndLon:   body.EndPoint.Longitude,
		Price:    body.Price,
	}

	trip, err := h.trips.Create(r.Context(), in)
	if err != nil {
		if errors.Is(err, repository.ErrDriverBusy) {
			writeProblem(w, r, http.StatusConflict, "driver_busy", "Driver already has an active trip")
			return
		}
		log.Printf("create trip: %v", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	writeJSON(w, http.StatusCreated, toTripResponse(trip))
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := h.trips.Get(r.Context(), uuid.UUID(tripId))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeProblem(w, r, http.StatusNotFound, "trip_not_found", "Trip not found")
			return
		}
		log.Printf("get trip: %v", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toTripResponse(trip))
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := h.trips.Finish(r.Context(), uuid.UUID(tripId))
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			writeProblem(w, r, http.StatusNotFound, "trip_not_found", "Trip not found")
		case errors.Is(err, repository.ErrAlreadyDone):
			writeProblem(w, r, http.StatusConflict, "trip_completed", "Trip already completed")
		default:
			log.Printf("finish trip: %v", err)
			writeProblem(w, r, http.StatusInternalServerError, "internal_error", "internal error")
		}
		return
	}
	writeJSON(w, http.StatusOK, toTripResponse(trip))
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.pool.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handler) ListTripPositions(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) CreateTripPosition(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	w.WriteHeader(http.StatusNotImplemented)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, code, detail string) {
	instance := r.URL.Path
	detailPtr := detail
	p := api.Problem{
		Type:     "https://tripgo.example/problems/" + code,
		Title:    http.StatusText(status),
		Status:   int32(status),
		Detail:   &detailPtr,
		Code:     code,
		Instance: &instance,
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}

func toTripResponse(t *service.Trip) api.Trip {
	return api.Trip{
		Id:         t.ID,
		UserId:     t.UserID,
		DriverId:   t.DriverID,
		StartPoint: api.Coordinates{Latitude: t.StartLat, Longitude: t.StartLon},
		EndPoint:   api.Coordinates{Latitude: t.EndLat, Longitude: t.EndLon},
		Price:      t.Price,
		Status:     api.TripStatus(t.Status),
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
	}
}

func validCoords(c api.Coordinates) bool {
	return c.Latitude >= -90 && c.Latitude <= 90 &&
		c.Longitude >= -180 && c.Longitude <= 180
}
