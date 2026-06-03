package main

import (
	"log"
	"net/http"
	"os"
	"time"

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

	host := os.Getenv("HOST")

	// Local development fallback
	if host == "" {
		host = "localhost:8080"
		docs.SwaggerInfo.Schemes = []string{"http"}
	} else {
		docs.SwaggerInfo.Schemes = []string{"https"}
	}

	// Swagger config
	docs.SwaggerInfo.Host = host
	docs.SwaggerInfo.BasePath = "/api/v1"
	// Start keep-alive job in background
	go startKeepAlive()

	// Swagger route
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	newRouter := r.NewRouter(router)
	newRouter.RegisterRoute()

	newRouter.Run(":8080", router)
}

func startKeepAlive() {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	url := "https://jobconnect-backend-4qq7.onrender.com/api/v1/users/wake-up"

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		resp, err := client.Get(url)
		if err != nil {
			log.Println("keep-alive error:", err)
		} else {
			resp.Body.Close()
			log.Println("keep-alive success:", resp.Status)
		}

		<-ticker.C
	}
}
