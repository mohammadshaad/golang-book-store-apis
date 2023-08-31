package main

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"

	"github.com/mohammadshaad/golang-book-store-backend/database"
	"github.com/mohammadshaad/golang-book-store-backend/routes"
)

func main() {
	fmt.Println("Welcome to the book store")

	// Load environment variables from the .env file
	if err := godotenv.Load(); err != nil {
		panic("Error loading .env file")
	}

	// Define the database connection string using environment variables
	database.InitDatabase()

	// Open the database connection
	db := database.GetDB()
	defer database.CloseDB()

	// Auto-migrate the models to create the necessary tables
	database.AutoMigrateModels(db)

	// Create a Fiber app
	app := fiber.New()

	// Define routes
	routes.DefineRoutes(app)

	// Start the Fiber app
	port := 8080 // You can change this to your desired port
	routes.StartApp(app, port)
}
