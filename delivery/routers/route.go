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
	portfolioRepo := repository.NewPortfolioRepository(db)
	portfolioUsecase := usecase.NewPortfolioUsecase(portfolioRepo)
	portfolioHandler := handler.NewPortfolioHandler(portfolioUsecase, userUsecase)
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
	userRoutes.HandleFunc("/{id}/portfolio", portfolioHandler.GetPortfolioByUserID).Methods("GET")

	// Protected routes
	userRoutes.HandleFunc("/me", userHandler.GetUserByID).Methods("GET")
	userRoutes.HandleFunc("/me", userHandler.UpdateUser).Methods("PATCH")
	userRoutes.HandleFunc("/me", userHandler.DeleteUser).Methods("DELETE")

	// =========================
	// JOB MODULE
	// =========================
	jobRoutes := api.PathPrefix("/jobs").Subrouter()

	jobRepo := repository.NewJobRepository(db)
	jobUsecase := usecase.NewJobUsecase(jobRepo)
	jobHandler := handler.NewJobHandler(jobUsecase)

	// =========================
	// JOB ENDPOINTS
	// =========================

	// Create job
	jobRoutes.HandleFunc("", jobHandler.CreateJob).Methods("POST")
	jobRoutes.HandleFunc("", jobHandler.ListJobs).Methods("GET")
	jobRoutes.HandleFunc("/mine", jobHandler.ListMyJobs).Methods("GET")
	jobRoutes.HandleFunc("/{id}", jobHandler.GetJobByID).Methods("GET")
	jobRoutes.HandleFunc("/{id}", jobHandler.UpdateJob).Methods("PATCH")
	jobRoutes.HandleFunc("/{id}", jobHandler.DeleteJob).Methods("DELETE")

	// =========================
	// PROPOSAL MODULE
	// =========================

	proposalRepo := repository.NewProposalRepository(db)
	proposalUsecase := usecase.NewProposalUsecase(proposalRepo)
	proposalHandler := handler.NewProposalHandler(proposalUsecase)

	proposalRoutes := api.PathPrefix("/proposals").Subrouter()

	// normal proposal routes
	proposalRoutes.HandleFunc("", proposalHandler.CreateProposal).Methods("POST")
	proposalRoutes.HandleFunc("/jobs", proposalHandler.ListProposalsByJobID).Methods("POST")
	proposalRoutes.HandleFunc("/mine", proposalHandler.ListMyProposals).Methods("GET")
	proposalRoutes.HandleFunc("/{id}", proposalHandler.GetProposalByID).Methods("GET")
	proposalRoutes.HandleFunc("/{id}", proposalHandler.UpdateProposal).Methods("PATCH")
	proposalRoutes.HandleFunc("/{id}", proposalHandler.DeleteProposal).Methods("DELETE")

	// =========================
	// PORTFOLIO MODULE
	// =========================
	portfolioRoutes := api.PathPrefix("/portfolio").Subrouter()

	portfolioRoutes.HandleFunc("", portfolioHandler.CreatePortfolioItem).Methods("POST")
	portfolioRoutes.HandleFunc("/{id}", portfolioHandler.UpdatePortfolioItem).Methods("PATCH")
	portfolioRoutes.HandleFunc("/{id}", portfolioHandler.DeletePortfolioItem).Methods("DELETE")

	// =========================
	// CONTRACT MODULE
	// =========================
	contractRepo := repository.NewContractRepository(db)
	contractUsecase := usecase.NewContractUsecase(contractRepo)
	contractHandler := handler.NewContractHandler(contractUsecase)

	contractRoutes := api.PathPrefix("/contracts").Subrouter()

	contractRoutes.HandleFunc("", contractHandler.CreateContract).Methods("POST")
	contractRoutes.HandleFunc("/mine", contractHandler.GetMyContracts).Methods("GET")
	contractRoutes.HandleFunc("/{id}", contractHandler.GetContractByID).Methods("GET")
	contractRoutes.HandleFunc("/milestone/submit", contractHandler.SubmitMilestone).Methods("POST")
	contractRoutes.HandleFunc("/milestone/{milestone_id}/status", contractHandler.ModifyMilestoneStatus).Methods("PATCH")
	contractRoutes.HandleFunc("/{contract_id}/status", contractHandler.ModifyContractStatus).Methods("PATCH")
	contractRoutes.HandleFunc("/work-session/start", contractHandler.StartWorkSession).Methods("POST")
	contractRoutes.HandleFunc("/work-session/end", contractHandler.EndWorkSession).Methods("POST")
	contractRoutes.HandleFunc("/work-session/time-logs", contractHandler.FetchTimeLogs).Methods("POST")
	contractRoutes.HandleFunc("/work-session/time-elapsed", contractHandler.FetchTimeElapsed).Methods("POST")
	contractRoutes.HandleFunc("/work-session/weekly-hours", contractHandler.FetchWeeklyHours).Methods("POST")
}

func (r *Router) Run(addr string, router *mux.Router) error {
	log.Println("Server running on port:", addr)
	return http.ListenAndServe(addr, router)
}
