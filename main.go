package main

import (
	r "job-connect/delivery/routers"

	"github.com/gorilla/mux"

	_ "job-connect/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Job Connect API
// @version 1.0
// @description Backend API for Job Connect platform
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in cookie
// @name token
func main() {
	router := mux.NewRouter()

	// Register the Swagger UI route
	// Swagger docs route
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)
	newRouter := r.NewRouter(router)
	newRouter.RegisterRoute()

	// Start the server
	newRouter.Run(":8080", router)
}
