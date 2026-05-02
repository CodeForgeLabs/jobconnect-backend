package delivery

import (
	"fmt"
	"log"
	"net/http"

	handler "job-connect/delivery/handlers"
	database "job-connect/infrastructure/database"
	"job-connect/infrastructure/repository"
	"job-connect/usecase"

	"github.com/gorilla/mux"
)

type Router struct {
	route *mux.Router
}

func NewRouter(route *mux.Router) Router {
	return Router{route: route}
}

func (r *Router) RegisterRoute() {
	db, err := database.NewDatabase()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to database")

	// FOR DEVELOPMENT ONLY
	err = database.Migrate(db)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database migrated successfully")

	// USER ROUTES
	userRepo := repository.NewUserRepository(db)
	userUsecase := usecase.NewUserUsecase(userRepo)
	userHandler := handler.NewUserHandler(userUsecase)
	// =========================
	// BASE API
	// =========================
	api := r.route.PathPrefix("/api/v1").Subrouter()
	// USER ROUTES
	userRoutes := api.PathPrefix("/users").Subrouter()

	// Public routes
	userRoutes.HandleFunc("/register", userHandler.CreateUser).Methods("POST")
	userRoutes.HandleFunc("/login", userHandler.Login).Methods("POST")
	userRoutes.HandleFunc("/email", userHandler.GetUserByEmail).Methods("GET")

	// Protected routes
	userRoutes.HandleFunc("/me", userHandler.GetUserByID).Methods("GET")
	userRoutes.HandleFunc("/me", userHandler.UpdateUser).Methods("PATCH")
	userRoutes.HandleFunc("/me", userHandler.DeleteUser).Methods("DELETE")

	// =========================
	// JOB ROUTES (future)
	// =========================
	jobRoutes := api.PathPrefix("/jobs").Subrouter()
	_ = jobRoutes

	// Example:
	// jobRoutes.HandleFunc("/", jobHandler.CreateJob).Methods("POST")
	// jobRoutes.HandleFunc("/", jobHandler.GetAllJobs).Methods("GET")
	// jobRoutes.HandleFunc("/{id}", jobHandler.GetJobByID).Methods("GET")
	// jobRoutes.HandleFunc("/{id}", jobHandler.UpdateJob).Methods("PATCH")
	// jobRoutes.HandleFunc("/{id}", jobHandler.DeleteJob).Methods("DELETE")

	// =========================
	// PROPOSAL ROUTES (future)
	// =========================
	proposalRoutes := api.PathPrefix("/proposals").Subrouter()
	_ = proposalRoutes

	// =========================
	// PAYMENT ROUTES (future)
	// =========================
	paymentRoutes := api.PathPrefix("/payments").Subrouter()
	_ = paymentRoutes

	// =========================
	// ADMIN ROUTES (future)
	// =========================
	adminRoutes := api.PathPrefix("/admin").Subrouter()
	_ = adminRoutes
}

func (r *Router) Run(addr string, router *mux.Router) error {
	log.Println("Server running on port:", addr)
	return http.ListenAndServe(addr, router)
}
