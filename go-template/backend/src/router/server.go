package router

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"cargonex-backend/src/auth"
	"cargonex-backend/src/controllers/deduction_service"
	"cargonex-backend/src/controllers/deduction_service/deduction_service_impl"
	"cargonex-backend/src/controllers/ledger_service"
	"cargonex-backend/src/controllers/ledger_service/ledger_service_impl"
	"cargonex-backend/src/controllers/maintenance_service"
	"cargonex-backend/src/controllers/maintenance_service/maintenance_service_impl"
	"cargonex-backend/src/controllers/resource_service"
	"cargonex-backend/src/controllers/resource_service/resource_service_impl"
	"cargonex-backend/src/controllers/tour_service"
	"cargonex-backend/src/controllers/tour_service/tour_service_impl"
	"cargonex-backend/src/controllers/user_service"
	"cargonex-backend/src/controllers/user_service/user_service_impl"
	"cargonex-backend/src/database/store"
)

type Server struct {
	db                 *sql.DB
	store              *store.Store
	userService        user_service.UserService
	ledgerService      ledger_service.LedgerService
	resourceService    resource_service.ResourceService
	maintenanceService maintenance_service.MaintenanceService
	tourService        tour_service.TourService
	deductionService   deduction_service.DeductionService
	authSecret         string
	corsOrigin         string
}

type currentUser struct {
	ID    int
	Email string
	Role  string
}

type ServerConfig struct {
	AuthSecret string
	CORSOrigin string
}

func NewServerCore(db *sql.DB, configs ...ServerConfig) *Server {
	cfg := serverConfig(configs...)
	appStore := store.New(db)
	return &Server{
		db:    db,
		store: appStore,
		userService: user_service_impl.NewUserService(user_service_impl.NewUserServiceImpl{
			Store:      appStore,
			AuthSecret: cfg.AuthSecret,
		}),
		ledgerService: ledger_service_impl.NewLedgerService(ledger_service_impl.NewLedgerServiceImpl{
			Store: appStore,
		}),
		resourceService: resource_service_impl.NewResourceService(resource_service_impl.NewResourceServiceImpl{
			Store: appStore,
		}),
		maintenanceService: maintenance_service_impl.NewMaintenanceService(maintenance_service_impl.NewMaintenanceServiceImpl{
			Store: appStore,
		}),
		tourService: tour_service_impl.NewTourService(tour_service_impl.NewTourServiceImpl{
			Store: appStore,
		}),
		deductionService: deduction_service_impl.NewDeductionService(deduction_service_impl.NewDeductionServiceImpl{
			Store: appStore,
		}),
		authSecret: cfg.AuthSecret,
		corsOrigin: cfg.CORSOrigin,
	}
}

func NewServer(db *sql.DB, configs ...ServerConfig) http.Handler {
	s := NewServerCore(db, configs...)
	return s.withMiddleware(s.routes())
}

func serverConfig(configs ...ServerConfig) ServerConfig {
	cfg := ServerConfig{
		AuthSecret: env("AUTH_TOKEN_SECRET", "nextroutex-development-secret"),
		CORSOrigin: env("CORS_ORIGIN", "http://localhost:3000"),
	}
	if len(configs) == 0 {
		return cfg
	}
	if configs[0].AuthSecret != "" {
		cfg.AuthSecret = configs[0].AuthSecret
	}
	if configs[0].CORSOrigin != "" {
		cfg.CORSOrigin = configs[0].CORSOrigin
	}
	return cfg
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api", s.handleRoot)

	mux.HandleFunc("POST /api/user/login", s.handleLogin)
	mux.HandleFunc("POST /api/user/register", s.requireAuth(s.handleRegister))
	mux.HandleFunc("POST /api/user/add", s.requireAuth(s.handleRegister))
	mux.HandleFunc("GET /api/user", s.requireAuth(s.handleListUsers))
	mux.HandleFunc("GET /api/user/{id}", s.requireAuth(s.handleGetUser))
	mux.HandleFunc("PATCH /api/user/{id}", s.requireAuth(s.handleUpdateUser))
	mux.HandleFunc("DELETE /api/user/{id}", s.requireAuth(s.handleDeleteUser))
	mux.HandleFunc("GET /api/ledger", s.requireAuth(s.handleListLedgers))
	mux.HandleFunc("GET /api/ledger/{id}", s.requireAuth(s.handleGetLedger))

	registerCRUD(mux, "/api/client", s.requireAuth, crudConfig{
		CollectionKey: "clients", ItemKey: "client", Table: `"Client"`,
		CreateMessage: "Client created successfully", UpdateMessage: "Client updated successfully", DeleteMessage: "Client deleted successfully",
		NotFoundMessage: "Client not found", CreateFailedMessage: "Client creation failed", UpdateFailedMessage: "Client update failed", DeleteFailedMessage: "Client deletion failed",
		CreateColumns: []store.Column{{JSON: "fullName", DB: "fullName", Required: true}, {JSON: "phoneNumber", DB: "phoneNumber", Required: true}, {JSON: "email", DB: "email", Lowercase: true, EmptyStringWhenMissing: true}, {JSON: "address", DB: "address", EmptyStringWhenMissing: true}},
		UpdateColumns: []store.Column{{JSON: "fullName", DB: "fullName"}, {JSON: "phoneNumber", DB: "phoneNumber"}, {JSON: "email", DB: "email", Lowercase: true}, {JSON: "address", DB: "address"}},
		SelectColumns: []string{"id", "fullName", "phoneNumber", "email", "address", "createdAt", "updatedAt"},
		Store:         s.store,
	})

	registerCRUD(mux, "/api/arrange-vehicle", s.requireAuth, crudConfig{
		CollectionKey: "arrangeVehicles", ItemKey: "arrangeVehicle", Table: `"ArrangeVehicle"`,
		CreateMessage: "Arrange vehicle created successfully", UpdateMessage: "Arrange vehicle updated successfully", DeleteMessage: "Arrange vehicle deleted successfully",
		NotFoundMessage: "Arrange vehicle not found", CreateFailedMessage: "Arrange vehicle creation failed", UpdateFailedMessage: "Arrange vehicle update failed", DeleteFailedMessage: "Arrange vehicle deletion failed",
		CreateColumns: []store.Column{{JSON: "vehicleNumber", DB: "vehicleNumber", Required: true}, {JSON: "vehicleType", DB: "vehicleType", Required: true}, {JSON: "ownerName", DB: "ownerName", Required: true}, {JSON: "ownerPhoneNumber", DB: "ownerPhoneNumber", Required: true}, {JSON: "email", DB: "email", Lowercase: true, NullWhenMissing: true}},
		UpdateColumns: []store.Column{{JSON: "vehicleNumber", DB: "vehicleNumber"}, {JSON: "vehicleType", DB: "vehicleType"}, {JSON: "ownerName", DB: "ownerName"}, {JSON: "ownerPhoneNumber", DB: "ownerPhoneNumber"}, {JSON: "email", DB: "email", Lowercase: true}},
		SelectColumns: []string{"id", "vehicleNumber", "vehicleType", "ownerName", "ownerPhoneNumber", "email", "createdAt", "updatedAt"},
		Store:         s.store,
	})

	registerCRUD(mux, "/api/inventory", s.requireAuth, crudConfig{
		CollectionKey: "inventories", ItemKey: "inventory", Table: `"Inventory"`,
		CreateMessage: "Inventory item created successfully", UpdateMessage: "Inventory item updated successfully", DeleteMessage: "Inventory item deleted successfully",
		NotFoundMessage: "Inventory item not found", CreateFailedMessage: "Inventory item creation failed", UpdateFailedMessage: "Inventory item update failed", DeleteFailedMessage: "Inventory item deletion failed",
		CreateColumns: []store.Column{{JSON: "itemName", DB: "itemName", Required: true}, {JSON: "category", DB: "category", Required: true, Cast: "InventoryCategory"}, {JSON: "unitType", DB: "unitType", Required: true, Cast: "InventoryUnitType"}, {JSON: "currentStockQty", DB: "currentStockQty", Required: true}, {JSON: "purchasePrice", DB: "purchasePrice", Required: true}},
		UpdateColumns: []store.Column{{JSON: "itemName", DB: "itemName"}, {JSON: "category", DB: "category", Cast: "InventoryCategory"}, {JSON: "unitType", DB: "unitType", Cast: "InventoryUnitType"}, {JSON: "currentStockQty", DB: "currentStockQty"}, {JSON: "purchasePrice", DB: "purchasePrice"}},
		SelectColumns: []string{"id", "itemName", "category", "unitType", "currentStockQty", "purchasePrice", "createdAt", "updatedAt"},
		Store:         s.store,
	})

	mux.HandleFunc("POST /api/vehicle/add", s.requireAuth(s.handleCreateVehicle))
	mux.HandleFunc("GET /api/vehicle", s.requireAuth(s.handleListVehicles))
	mux.HandleFunc("GET /api/vehicle/{id}", s.requireAuth(s.handleGetVehicle))
	mux.HandleFunc("PATCH /api/vehicle/{id}", s.requireAuth(s.handleUpdateVehicle))
	mux.HandleFunc("DELETE /api/vehicle/{id}", s.requireAuth(s.handleDeleteVehicle))

	mux.HandleFunc("POST /api/driver/add", s.requireAuth(s.handleCreateDriver))
	mux.HandleFunc("GET /api/driver", s.requireAuth(s.handleListDrivers))
	mux.HandleFunc("PATCH /api/driver/{id}", s.requireAuth(s.handleUpdateDriver))
	mux.HandleFunc("DELETE /api/driver/{id}", s.requireAuth(s.handleDeleteDriver))

	maintenanceCRUD := crudConfig{
		CollectionKey: "maintenances", ItemKey: "maintenance", Table: `"VehicleMaintenance"`,
		IncludeData:   true,
		CreateMessage: "Vehicle maintenance created successfully", UpdateMessage: "Vehicle maintenance updated successfully", DeleteMessage: "Vehicle maintenance deleted successfully",
		NotFoundMessage: "Vehicle maintenance not found", CreateFailedMessage: "Vehicle maintenance creation failed", UpdateFailedMessage: "Vehicle maintenance update failed", DeleteFailedMessage: "Vehicle maintenance deletion failed",
		CreateColumns: []store.Column{{JSON: "vehicleId", DB: "vehicleId", Required: true}, {JSON: "vehicleName", DB: "vehicleName", NullWhenMissing: true}, {JSON: "workService", DB: "workService", Required: true}, {JSON: "category", DB: "category", Required: true, Cast: "VehicleMaintenanceCategory"}, {JSON: "date", DB: "date", Required: true}, {JSON: "amount", DB: "amount", Required: true}, {JSON: "paymentStatus", DB: "paymentStatus", Required: true, Cast: "VehicleMaintenancePaymentStatus"}, {JSON: "vendorWorkshop", DB: "vendorWorkshop", Required: true}, {JSON: "description", DB: "description", NullWhenMissing: true}},
		UpdateColumns: []store.Column{{JSON: "vehicleId", DB: "vehicleId"}, {JSON: "vehicleName", DB: "vehicleName"}, {JSON: "workService", DB: "workService"}, {JSON: "category", DB: "category", Cast: "VehicleMaintenanceCategory"}, {JSON: "date", DB: "date"}, {JSON: "amount", DB: "amount"}, {JSON: "paymentStatus", DB: "paymentStatus", Cast: "VehicleMaintenancePaymentStatus"}, {JSON: "vendorWorkshop", DB: "vendorWorkshop"}, {JSON: "description", DB: "description"}},
		SelectColumns: []string{"id", "vehicleId", "vehicleName", "workService", "category", "date", "amount", "paymentStatus", "vendorWorkshop", "description", "createdAt", "updatedAt"},
		Store:         s.store,
		AfterCreate: func(r *http.Request, user currentUser, item map[string]any) error {
			return s.maintenanceService.SyncLedger(r.Context(), idFromItem(item), user.ID)
		},
		AfterUpdate: func(r *http.Request, user currentUser, item map[string]any) error {
			return s.maintenanceService.SyncLedger(r.Context(), idFromItem(item), user.ID)
		},
		AfterDelete: func(r *http.Request, user currentUser, id int) error {
			return s.maintenanceService.DeleteLedger(r.Context(), id, user.ID)
		},
	}
	registerCRUD(mux, "/api/vehicle-maintenance", s.requireAuth, maintenanceCRUD)
	registerMaintenanceUpdateAliases(mux, s.requireAuth, maintenanceCRUD)

	registerCRUD(mux, "/api/tour-damage", s.requireAuth, crudConfig{
		CollectionKey: "damages", ItemKey: "damage", Table: `"TourDamage"`,
		CreateMessage: "Tour damage created successfully", UpdateMessage: "Tour damage updated successfully", DeleteMessage: "Tour damage deleted successfully",
		NotFoundMessage: "Tour damage not found", CreateFailedMessage: "Tour damage creation failed", UpdateFailedMessage: "Tour damage update failed", DeleteFailedMessage: "Tour damage deletion failed",
		CreateColumns: []store.Column{{JSON: "tourId", DB: "tourId", Required: true}, {JSON: "tourName", DB: "tourName", NullWhenMissing: true}, {JSON: "vehicleId", DB: "vehicleId", NullWhenMissing: true}, {JSON: "vehicleName", DB: "vehicleName", NullWhenMissing: true}, {JSON: "driverId", DB: "driverId", NullWhenMissing: true}, {JSON: "driverName", DB: "driverName", NullWhenMissing: true}, {JSON: "damageType", DB: "damageType", Required: true, Cast: "DamageType"}, {JSON: "damageDate", DB: "damageDate", Required: true}, {JSON: "reason", DB: "reason", NullWhenMissing: true}},
		UpdateColumns: []store.Column{{JSON: "tourId", DB: "tourId"}, {JSON: "tourName", DB: "tourName"}, {JSON: "vehicleId", DB: "vehicleId"}, {JSON: "vehicleName", DB: "vehicleName"}, {JSON: "driverId", DB: "driverId"}, {JSON: "driverName", DB: "driverName"}, {JSON: "damageType", DB: "damageType", Cast: "DamageType"}, {JSON: "damageDate", DB: "damageDate"}, {JSON: "reason", DB: "reason"}},
		SelectColumns: []string{"id", "tourId", "tourName", "vehicleId", "vehicleName", "driverId", "driverName", "damageType", "damageDate", "reason", "createdAt", "updatedAt"},
		Store:         s.store,
	})

	registerCRUD(mux, "/api/tour-deduction", s.requireAuth, crudConfig{
		CollectionKey: "deductions", ItemKey: "deduction", Table: `"TourDeduction"`,
		CreateMessage: "Tour deduction created successfully", UpdateMessage: "Tour deduction updated successfully", DeleteMessage: "Tour deduction deleted successfully",
		NotFoundMessage: "Tour deduction not found", CreateFailedMessage: "Tour deduction creation failed", UpdateFailedMessage: "Tour deduction update failed", DeleteFailedMessage: "Tour deduction deletion failed",
		CreateColumns: []store.Column{{JSON: "tourId", DB: "tourId", Required: true}, {JSON: "deductedAmount", DB: "deductedAmount", Required: true}, {JSON: "deductionType", DB: "deductionType", Cast: "DeductionType", DefaultFromJSON: "status"}, {JSON: "deductionDate", DB: "deductionDate", Required: true}, {JSON: "reason", DB: "reason", Required: true}, {JSON: "status", DB: "status", Required: true, Cast: "DeductionType"}},
		UpdateColumns: []store.Column{{JSON: "tourId", DB: "tourId"}, {JSON: "deductedAmount", DB: "deductedAmount"}, {JSON: "deductionType", DB: "deductionType", Cast: "DeductionType"}, {JSON: "deductionDate", DB: "deductionDate"}, {JSON: "reason", DB: "reason"}, {JSON: "status", DB: "status", Cast: "DeductionType"}},
		SelectColumns: []string{"id", "tourId", "deductedAmount", "deductionType", "deductionDate", "reason", "status", "createdAt", "updatedAt"},
		Store:         s.store,
		AfterCreate: func(r *http.Request, user currentUser, item map[string]any) error {
			deductionType := stringValue(item["deductionType"])
			if deductionType == "" {
				deductionType = stringValue(item["status"])
			}
			return s.deductionService.ApplyTourDeductionStatus(r.Context(), stringValue(item["tourId"]), deductionType, user.ID, true)
		},
		DisableDelete: true,
	})
	mux.HandleFunc("DELETE /api/tour-deduction/{id}", s.requireAuth(s.handleDeleteTourDeduction))

	registerCRUD(mux, "/api/tour", s.requireAuth, crudConfig{
		CollectionKey: "tours", ItemKey: "tour", Table: `"Tour"`,
		ListMessage:   "Tours fetched successfully",
		TransformList: s.formatTourList,
		CreateMessage: "Tour created successfully", UpdateMessage: "Tour updated successfully", DeleteMessage: "Tour deleted successfully",
		NotFoundMessage: "Tour not found", CreateFailedMessage: "Tour creation failed", UpdateFailedMessage: "Tour update failed", DeleteFailedMessage: "Tour deletion failed",
		CreateColumns: []store.Column{{JSON: "tourName", DB: "tourName", Required: true}, {JSON: "driver", DB: "driver", NullWhenMissing: true, JSONString: true}, {JSON: "vehicle", DB: "vehicle", NullWhenMissing: true, JSONString: true}, {JSON: "client", DB: "client", Required: true, JSONString: true}, {JSON: "startLocation", DB: "startLocation", Required: true, JSONB: true}, {JSON: "endLocation", DB: "endLocation", Required: true, JSONB: true}, {JSON: "startDate", DB: "startDate", Required: true}, {JSON: "time", DB: "time", Required: true}, {JSON: "expectedEndDate", DB: "expectedEndDate", DefaultFromJSON: "startDate"}, {JSON: "actualEndDate", DB: "actualEndDate", NullWhenMissing: true}, {JSON: "actualEndTime", DB: "actualEndTime", NullWhenMissing: true}, {JSON: "freightAmount", DB: "freightAmount", Required: true}, {JSON: "advanceAmount", DB: "advanceAmount", NullWhenMissing: true}, {JSON: "otherCharges", DB: "otherCharges", NullWhenMissing: true}, {JSON: "paymentStatus", DB: "paymentStatus", Cast: "TripPaymentStatus", Default: "Pending"}, {JSON: "partialReceivedPayment", DB: "partialReceivedPayment", NullWhenMissing: true}, {JSON: "expenses", DB: "expenses", JSONB: true, Default: []any{}}, {JSON: "fuelDetails", DB: "fuelDetails", JSONB: true, Default: []any{}}, {JSON: "loadType", DB: "loadType", NullWhenMissing: true}, {JSON: "cargoWeight", DB: "cargoWeight", NullWhenMissing: true}, {JSON: "vehicleType", DB: "vehicleType", NullWhenMissing: true, Cast: "VehicleType"}, {JSON: "status", DB: "status", Required: true, Cast: "TourStatus"}, {JSON: "notes", DB: "notes", NullWhenMissing: true}},
		UpdateColumns: []store.Column{{JSON: "tourName", DB: "tourName"}, {JSON: "driver", DB: "driver"}, {JSON: "vehicle", DB: "vehicle"}, {JSON: "client", DB: "client"}, {JSON: "startLocation", DB: "startLocation", JSONB: true}, {JSON: "endLocation", DB: "endLocation", JSONB: true}, {JSON: "startDate", DB: "startDate"}, {JSON: "time", DB: "time"}, {JSON: "expectedEndDate", DB: "expectedEndDate"}, {JSON: "actualEndDate", DB: "actualEndDate"}, {JSON: "actualEndTime", DB: "actualEndTime"}, {JSON: "freightAmount", DB: "freightAmount"}, {JSON: "advanceAmount", DB: "advanceAmount"}, {JSON: "otherCharges", DB: "otherCharges"}, {JSON: "paymentStatus", DB: "paymentStatus", Cast: "TripPaymentStatus"}, {JSON: "partialReceivedPayment", DB: "partialReceivedPayment"}, {JSON: "expenses", DB: "expenses", JSONB: true}, {JSON: "fuelDetails", DB: "fuelDetails", JSONB: true}, {JSON: "loadType", DB: "loadType"}, {JSON: "cargoWeight", DB: "cargoWeight"}, {JSON: "vehicleType", DB: "vehicleType", Cast: "VehicleType"}, {JSON: "status", DB: "status", Cast: "TourStatus"}, {JSON: "notes", DB: "notes"}},
		SelectColumns: []string{"id", "tourName", "driver", "vehicle", "client", "startLocation", "endLocation", "startDate", "time", "expectedEndDate", "actualEndDate", "actualEndTime", "freightAmount", "advanceAmount", "otherCharges", "paymentStatus", "partialReceivedPayment", "expenses", "fuelDetails", "loadType", "cargoWeight", "vehicleType", "status", "notes", "createdAt", "updatedAt", "createdById"},
		Store:         s.store,
		BeforeCreate: func(r *http.Request, user currentUser, body map[string]any) error {
			return s.populateTourRelationNames(r.Context(), user.ID, body)
		},
		AfterCreate: func(r *http.Request, user currentUser, item map[string]any) error {
			return s.tourService.SyncLedgers(r.Context(), idFromItem(item), user.ID)
		},
		AfterUpdate: func(r *http.Request, user currentUser, item map[string]any) error {
			if err := s.tourService.SyncLedgers(r.Context(), idFromItem(item), user.ID); err != nil {
				return err
			}
			if item["status"] == "Completed" {
				return s.tourService.ResetAssignedResources(r.Context(), item, user.ID)
			}
			return nil
		},
		DisableUpdate: true,
		DisableDelete: true,
	})
	// Override CRUD PATCH with smart dispatcher (numeric ID = normal update, slug = petrol pump bulk update)
	mux.HandleFunc("PATCH /api/tour/{id}", s.requireAuth(s.handleUpdateTour))
	mux.HandleFunc("DELETE /api/tour/{id}", s.requireAuth(s.handleDeleteTour))

	return mux
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.corsOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Token, X-Access-Token, Content-Type, Accept, Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if len(r.URL.Path) > 1 && strings.HasSuffix(r.URL.Path, "/") {
			r.URL.Path = strings.TrimRight(r.URL.Path, "/")
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAuth(next func(http.ResponseWriter, *http.Request, currentUser)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerOrToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		payload, ok := auth.VerifyToken(token, s.authSecret)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		user, err := s.store.ActiveAuthUser(r.Context(), payload.Subject)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		next(w, r, currentUser{ID: user.ID, Email: user.Email, Role: user.Role})
	}
}

func bearerOrToken(r *http.Request) string {
	if token := r.Header.Get("Token"); token != "" {
		return token
	}
	if token := r.Header.Get("X-Access-Token"); token != "" {
		return token
	}
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
		return parts[1]
	}
	return authHeader
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	writeOK(w, "Hello World!")
}

func parseID(r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	return id, err == nil && id > 0
}

func pagination(r *http.Request) (page int, limit int, ok bool) {
	page = queryInt(r, "page", 1)
	limit = queryInt(r, "limit", 10)
	if page < 1 || limit < 1 || limit > 100 {
		return page, limit, false
	}
	return page, limit, true
}

func queryInt(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return -1
	}
	return v
}

func decodeJSON(r *http.Request) (map[string]any, error) {
	defer r.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, nil
}

func writeOK(w http.ResponseWriter, data any) {
	if obj, ok := data.(map[string]any); ok {
		if _, exists := obj["message"]; !exists {
			obj["message"] = "Request successful"
		}
		if _, exists := obj["success"]; !exists {
			obj["success"] = true
		}
		writeJSON(w, http.StatusOK, obj)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Request successful", "success": true, "data": data})
}

func writeCreated(w http.ResponseWriter, data map[string]any) {
	if _, exists := data["success"]; !exists {
		data["success"] = true
	}
	writeJSON(w, http.StatusCreated, data)
}

func writeError(w http.ResponseWriter, status int, message any) {
	writeJSON(w, status, map[string]any{"message": message, "success": false})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func errorStatus(err error) int {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, store.ErrEmptyUpdate), errors.Is(err, store.ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, store.ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
