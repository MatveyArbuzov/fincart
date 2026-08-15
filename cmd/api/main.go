package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/MatveyArbuzov/fincart/internal/database"
	"github.com/MatveyArbuzov/fincart/internal/httpserver"
	"github.com/MatveyArbuzov/fincart/internal/product"
)

func main() {
	ctx := context.Background()
	dsn := os.Getenv("DATABASE_URL")
	db, err := database.NewPostgres(ctx, dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	productRepository := product.NewPostgresRepository(db)

	productService := product.NewService(productRepository)

	productHandler := product.NewHandler(productService)

	router := httpserver.NewRouter(productHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	log.Println("server started on :8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
