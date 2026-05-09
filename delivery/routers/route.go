package delivery

import (
	"fmt"
	"log"
	"net/http"

	"job-connect/chapa"
	handler "job-connect/delivery/handlers"
	"job-connect/delivery/ws"
	database "job-connect/infrastructure/database"
	"job-connect/infrastructure/repository"
	"job-connect/usecase"

	"github.com/gorilla/handlers"
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
	reviewRepo := repository.NewReviewRepository(db)
	reviewUsecase := usecase.NewReviewUsecase(reviewRepo)
	reviewHandler := handler.NewReviewHandler(reviewUsecase, userUsecase)
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
	userRoutes.HandleFunc("/{id}/reviews", reviewHandler.ListReviewsByFreelancerID).Methods("GET")

	// Protected routes
	userRoutes.HandleFunc("/me", userHandler.GetUserByID).Methods("GET")
	userRoutes.HandleFunc("/me", userHandler.UpdateUser).Methods("PATCH")
	userRoutes.HandleFunc("/me", userHandler.DeleteUser).Methods("DELETE")
	userRoutes.HandleFunc("/logged", userHandler.UserLoggedIn).Methods("GET")
	userRoutes.HandleFunc("/logout", userHandler.UserLogout).Methods("POST")

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
	// REVIEW MODULE
	// =========================
	reviewRoutes := api.PathPrefix("/reviews").Subrouter()

	reviewRoutes.HandleFunc("", reviewHandler.CreateReview).Methods("POST")
	reviewRoutes.HandleFunc("/{id}", reviewHandler.UpdateReview).Methods("PATCH")
	reviewRoutes.HandleFunc("/{id}/reply", reviewHandler.UpdateReviewReply).Methods("PATCH")

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

	// =========================
	// MESSAGE MODULE
	// =========================
	hub := ws.NewHub()

	r.route.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWS(hub, w, r)
	})
	messageRepo := repository.NewMessageRepository(db, hub)
	messageUsecase := usecase.NewMessageUsecase(messageRepo)
	messageHandler := handler.NewMessageHandler(messageUsecase)

	messageRoutes := api.PathPrefix("/messages").Subrouter()

	messageRoutes.HandleFunc("", messageHandler.CreateMessage).Methods("POST")
	messageRoutes.HandleFunc("", messageHandler.GetMessagesByConversationID).Methods("GET")
	messageRoutes.HandleFunc("/conversations", messageHandler.GetConversationsByUserID).Methods("GET")
	messageRoutes.HandleFunc("/seen", messageHandler.MarkMessageAsSeen).Methods("POST")

	// =========================
	// WALLET MODULE
	// =========================
	chapaClient := chapa.NewClient(
		database.GetEnv("CHAPA_SECRET_KEY", ""),
		database.GetEnv("CHAPA_BASE_URL", "https://api.chapa.co/v1"),
	)
	walletRepo := repository.NewWalletRepo(db)
	walletUsecase := usecase.NewWalletUsecase(walletRepo)
	walletHandler := handler.NewWalletHandler(walletUsecase, chapaClient)

	walletRoutes := api.PathPrefix("/wallet").Subrouter()

	walletRoutes.HandleFunc("/balance", walletHandler.GetOrCreateWallet).Methods("GET")
	walletRoutes.HandleFunc("/transaction", walletHandler.CreateTransaction).Methods("POST")
	walletRoutes.HandleFunc("/transaction/update", walletHandler.UpdateTransactionStatus).Methods("GET")
	walletRoutes.HandleFunc("/transactions", walletHandler.FetchTransactions).Methods("GET")

}

func (r *Router) Run(addr string, router *mux.Router) error {

	headers := handlers.AllowedHeaders([]string{
		"X-Requested-With",
		"Content-Type",
		"Authorization",
	})

	methods := handlers.AllowedMethods([]string{
		"GET",
		"POST",
		"PUT",
		"PATCH",
		"DELETE",
		"OPTIONS",
	})

	origins := handlers.AllowedOrigins([]string{
		"http://localhost:3000",
		"http://localhost:5173",
	})

	return http.ListenAndServe(
		addr,
		handlers.CORS(
			headers,
			methods,
			origins,
			handlers.AllowCredentials(),
		)(router),
	)
}
