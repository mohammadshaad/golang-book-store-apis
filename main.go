package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-playground/validator/v10"

	jwtware "github.com/gofiber/jwt/v2"
	"github.com/golang-jwt/jwt"
)

// UserRole represents the role of a user
type UserRole string

const (
	UserRoleAdmin    UserRole = "admin"
	UserRoleStandard UserRole = "standard"
)

type User struct {
	gorm.Model
	UserID    uint     `json:"id"`
	FirstName string   `json:"firstname"`
	LastName  string   `json:"lastname"`
	Email     string   `json:"email"`
	Password  []byte   `json:"-"`
	Role      UserRole `json:"role"`
}

type Book struct {
	ID          uint    `json:"id"`
	Title       string  `json:"title"`
	Author      string  `json:"author"`
	ISBN        string  `json:"isbn"`
	Genre       string  `json:"genre"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
	Description string  `json:"description"`
}

var db *gorm.DB
var validate *validator.Validate
var jwtSecret = []byte("secret")

// Function to generate a JWT token
func generateJWTToken(user *User) (string, error) {
	// Create a new token
	token := jwt.New(jwt.SigningMethodHS256)

	// Set claims
	claims := token.Claims.(jwt.MapClaims)
	claims["id"] = user.UserID
	claims["email"] = user.Email
	claims["role"] = user.Role

	// Sign the token with your JWT secret
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func main() {
	fmt.Println("Welcome to the book store")

	// Initialize the validator
	validate = validator.New()

	// Load environment variables from the .env file
	if err := godotenv.Load(); err != nil {
		panic("Error loading .env file")
	}

	// Define the database connection string using environment variables
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	// Define the database connection string
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	// connStr := "user=postgres password=oppo-098 dbname=book-store port=5432 sslmode=disable TimeZone=Asia/Shanghai"

	// Open the database connection
	var err error
	db, err = gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		panic("failed to connect to the database")
	}

	// Auto-migrate the User model to create the users table
	db.AutoMigrate(&User{})

	// Create a Fiber app
	app := fiber.New()

	// Define JWT middleware
	jwtMiddleware := jwtware.New(jwtware.Config{
		SigningKey: jwtSecret,
	})

	// Define a route for creating an admin user
	app.Post("/admin/register", createAdminUserHandler)

	// Define a route for making a user an admin
	app.Put("/admin/make-admin/:id", makeAdminHandler)

	// Apply JWT middleware to routes that require authentication
	app.Use(jwtMiddleware)

	// Define a route for user registration
	app.Post("/register", registerHandler)

	// Define a route for user login
	app.Post("/login", loginHandler)

	// Define a route for deactivating an account
	app.Put("/deactivate/:id", deactivateAccountHandler)

	// Define a route for activating an account
	app.Put("/activate/:id", activateAccountHandler)

	// Define a route for deleting an account
	app.Delete("/delete/:id", deleteAccountHandler)

	// Define a route for getting a user's profile
	app.Get("/:id", profile)

	// Define a route for updating a user's profile
	app.Put("/:id", updateProfile)

	// Create a new book
	app.Post("/books", createBookHandler)

	// Get a list of all books
	app.Get("/", getAllBooksHandler)

	// Get a single book by ID
	app.Get("/books/:id", getBookByIDHandler)

	// Update a book by ID
	app.Put("/books/:id", updateBookHandler)

	// Delete a book by ID
	app.Delete("/books/:id", deleteBookHandler)

	// Start the Fiber app
	port := 8080 // You can change this to your desired port
	fmt.Printf("Server is listening on port %d...\n", port)
	app.Listen(fmt.Sprintf(":%d", port))
}

// Create an admin user
func createAdminUserHandler(c *fiber.Ctx) error {

	// Parse the user data from the request body
	var userData User
	if err := c.BodyParser(&userData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input data",
		})
	}

	// Check if the user is an admin before creating an admin user
	userRole := c.Locals("user").(*jwt.Token).Claims.(jwt.MapClaims)["role"].(string)
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Forbidden: Only admin users can create admin users",
		})
	}

	// Validate user input
	if err := validate.Struct(userData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid input data",
			"errors": err.(validator.ValidationErrors),
		})
	}

	// Check if the user already exists (email must be unique)
	var user User
	if err := db.Where("email = ?", userData.Email).First(&user).Error; err == nil {
		// User already exists, don't register again
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "User already exists",
		})
	}

	// Generate a random numeric user ID
	rand.Seed(time.Now().UnixNano())
	userID := uint(rand.Intn(10000)) // Change the range as needed

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userData.Password), 10)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot hash password",
		})
	}

	// Create a new user with the generated ID
	newUser := User{
		UserID:    userID,
		FirstName: userData.FirstName,
		LastName:  userData.LastName,
		Email:     userData.Email,
		Password:  hashedPassword,
		Role:      "admin",
	}

	if err := db.Create(&newUser).Error; err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "User registration failed",
		})
	}

	// Return a success message

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Admin user created successfully",
	})
}

// Make a user an admin
func makeAdminHandler(c *fiber.Ctx) error {
	// Parse the user ID from the URL params
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID format",
		})
	}

	// Check if the user is an admin before making another user an admin
	userRole := c.Locals("user").(*jwt.Token).Claims.(jwt.MapClaims)["role"].(string)
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Forbidden: Only admin users can make other users admins",
		})
	}

	// Find the user in the database
	var user User
	if err := db.First(&user, uint(id)).Error; err != nil {
		// Handle database errors (e.g., no user with the given ID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Make the user an admin
	if err := db.Model(&user).Update("role", "admin").Error; err != nil {
		// Handle database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot make user an admin",
		})
	}

	// Return a success message

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User is now an admin",
	})
}

func loginHandler(c *fiber.Ctx) error {
	var userData struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}

	if err := c.BodyParser(&userData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Cannot parse JSON",
		})
	}

	// Validate user input
	if err := validate.Struct(userData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Invalid input data",
			"errors":  err.(validator.ValidationErrors),
		})
	}

	// Find the user in the database
	var user User
	if err := db.Where("email = ?", userData.Email).First(&user).Error; err != nil {
		// Handle database errors (e.g., no user with the given email)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Compare the given password with the password in the database
	if err := bcrypt.CompareHashAndPassword(user.Password, []byte(userData.Password)); err != nil {
		// Handle password incorrect error
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect password",
		})
	}

	// Generate a JWT token for the logged-in user
	token, err := generateJWTToken(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Token generation failed",
		})
	}

	// Return the JWT token along with the success message
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Logged in successfully",
		"token":   token,
	})

}

func registerHandler(c *fiber.Ctx) error {
	var userData struct {
		FirstName string `json:"firstname" validate:"required"`
		LastName  string `json:"lastname" validate:"required"`
		Email     string `json:"email" validate:"required,email"`
		Password  string `json:"password" validate:"required"`
		Role      string `json:"role" validate:"required"` // Assuming role is required during registration
	}

	if err := c.BodyParser(&userData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Cannot parse JSON",
		})
	}

	// Validate user input
	if err := validate.Struct(userData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Invalid input data",
			"errors":  err.(validator.ValidationErrors),
		})
	}

	// Check if the user already exists (email must be unique)
	var user User
	if err := db.Where("email = ?", userData.Email).First(&user).Error; err == nil {
		// User already exists, don't register again
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "User already exists",
		})
	}

	// Generate a random numeric user ID
	rand.Seed(time.Now().UnixNano())
	userID := uint(rand.Intn(10000)) // Change the range as needed

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userData.Password), 10)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Cannot hash password",
		})
	}

	// Create a new user with the generated ID
	newUser := User{
		UserID:    userID,
		FirstName: userData.FirstName,
		LastName:  userData.LastName,
		Email:     userData.Email,
		Password:  hashedPassword,
	}

	if err := db.Create(&newUser).Error; err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "User registration failed",
		})
	}

	// Generate a JWT token for the new user
	token, err := generateJWTToken(&newUser)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Token generation failed",
		})
	}

	// Return the JWT token along with the success message
	return c.JSON(fiber.Map{
		"success": true,
		"message": "User created successfully",
		"token":   token,
	})

}

func deactivateAccountHandler(c *fiber.Ctx) error {
	// Get the "id" URL parameter and convert it to a uint
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		// Handle invalid ID format
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID format",
		})
	}

	// Find the user in the database
	var user User
	if err := db.First(&user, uint(id)).Error; err != nil {
		// Handle database errors (e.g., no user with the given ID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Deactivate the user
	if err := db.Model(&user).Update("active", false).Error; err != nil {
		// Handle database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot deactivate user",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User deactivated successfully",
	})
}

func activateAccountHandler(c *fiber.Ctx) error {
	// Get the "id" URL parameter and convert it to a uint
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		// Handle invalid ID format
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID format",
		})
	}

	// Find the user in the database
	var user User
	if err := db.First(&user, uint(id)).Error; err != nil {
		// Handle database errors (e.g., no user with the given ID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Activate the user
	if err := db.Model(&user).Update("active", true).Error; err != nil {
		// Handle database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot activate user",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User activated successfully",
	})
}

func deleteAccountHandler(c *fiber.Ctx) error {
	// Get the "id" URL parameter and convert it to a uint
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		// Handle invalid ID format
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID format",
		})
	}

	// Find the user in the database
	var user User
	if err := db.First(&user, uint(id)).Error; err != nil {
		// Handle database errors (e.g., no user with the given ID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Delete the user's account from the database
	if err := db.Delete(&user).Error; err != nil {
		// Handle database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot delete user account",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User account deleted successfully",
	})
}

func profile(c *fiber.Ctx) error {
	// Get the "id" URL parameter and convert it to a uint
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		// Handle invalid ID format
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID format",
		})
	}

	// Find the user in the database
	var user User
	if err := db.First(&user, uint(id)).Error; err != nil {
		// Handle database errors (e.g., no user with the given ID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(user)
}

func updateProfile(c *fiber.Ctx) error {
	// Get the "id" URL parameter and convert it to a uint
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		// Handle invalid ID format
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID format",
		})
	}

	// Find the user in the database
	var user User
	if err := db.First(&user, uint(id)).Error; err != nil {
		// Handle database errors (e.g., no user with the given ID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	var userData User

	if err := c.BodyParser(&userData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Cannot parse JSON",
		})
	}

	// Update the user's first name if it's provided in the request
	if userData.FirstName != "" {
		user.FirstName = userData.FirstName
	}

	// Update the user's last name if it's provided in the request
	if userData.LastName != "" {
		user.LastName = userData.LastName
	}

	// Update the user's email if it's provided in the request
	if userData.Email != "" {
		user.Email = userData.Email
	}

	// Update the user's password if it's provided in the request
	if len(userData.Password) > 0 {
		// Hash the new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userData.Password), 10)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"success": false,
				"message": "Cannot hash password",
			})
		}
		user.Password = hashedPassword
	}

	if err := db.Save(&user).Error; err != nil {
		// Handle database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot update user's profile",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User profile updated successfully",
	})
}

// Create a new book
func createBookHandler(c *fiber.Ctx) error {
	var newBook Book
	if err := c.BodyParser(&newBook); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input data",
		})
	}

	// Generate a random numeric book ID
	rand.Seed(time.Now().UnixNano())
	bookID := uint(rand.Intn(10000)) // Change the range as needed

	// Set the generated ID for the new book
	newBook.ID = bookID

	// Save the new book to the database
	if err := db.Create(&newBook).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create book",
		})
	}
	return c.JSON(newBook)
}

// Get a list of all books or a single book by ID
func getAllBooksHandler(c *fiber.Ctx) error {
	id := c.Params("id")

	if id == "" {
		// No ID parameter, fetch all books
		var books []Book
		if err := db.Find(&books).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch books",
			})
		}
		return c.JSON(books)
	}

	// ID parameter is present, fetch a single book by ID
	var book Book
	if err := db.First(&book, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Book not found",
		})
	}
	return c.JSON(book)
}

// Get a single book by ID
func getBookByIDHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var book Book
	if err := db.First(&book, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Book not found",
		})
	}
	return c.JSON(book)
}

// Update a book by ID
func updateBookHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var updatedBook Book
	if err := c.BodyParser(&updatedBook); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input data",
		})
	}
	var existingBook Book
	if err := db.First(&existingBook, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Book not found",
		})
	}
	// Update the book's fields
	existingBook.Title = updatedBook.Title
	existingBook.Author = updatedBook.Author
	existingBook.Description = updatedBook.Description
	existingBook.Price = updatedBook.Price
	existingBook.Quantity = updatedBook.Quantity
	// Save the updated book to the database
	if err := db.Save(&existingBook).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update book",
		})
	}
	return c.JSON(existingBook)
}

// Delete a book by ID
func deleteBookHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var book Book
	if err := db.First(&book, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Book not found",
		})
	}
	// Delete the book from the database
	if err := db.Delete(&book).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete book",
		})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Book deleted successfully",
	})
}
