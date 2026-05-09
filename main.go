package main

import (
	"os"

	r "job-connect/delivery/routers"

	"github.com/gorilla/mux"

	"job-connect/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Job Connect API
// @version 1.0
// @description Backend API for Job Connect platform
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in cookie
// @name token

func main() {
	router := mux.NewRouter()

	// Swagger dynamic config
	docs.SwaggerInfo.Host = os.Getenv("HOST")
	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Schemes = []string{"https"}

	// Swagger route
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	newRouter := r.NewRouter(router)
	newRouter.RegisterRoute()

	newRouter.Run(":8080", router)
}
