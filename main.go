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

	jwtware "github.com/gofiber/jwt/v3"
	"github.com/golang-jwt/jwt/v4"
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

	// Open the database connection
	var err error
	db, err = gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		panic("failed to connect to the database")
	}

	// Auto-migrate the User and Book model to create the users table
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Book{})

	// Create a Fiber app
	app := fiber.New()

	// Define a route for the home page
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome to the book store!")
	})

	// Define a route for user registration
	app.Post("/register", registerHandler)

	// Define a route for user login
	app.Post("/login", loginHandler)

	// Define a middleware to protect routes that require a valid JWT
	user := app.Group("/user")
	user.Use(jwtware.New(jwtware.Config{
		SigningKey: []byte(os.Getenv("JWT_SECRET")),
	}))

	// Define a route for the user section
	user.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome user!")
	})

	// Define a route for getting a user's profile
	user.Get("/profile/:id", profile)

	// Define a route for updating a user's profile
	user.Put("/profile/:id", updateProfile)

	// Define a route for deactivating an account
	user.Put("/deactivate/:id", deactivateAccountHandler)

	// Define a route for activating an account
	user.Put("/activate/:id", activateAccountHandler)

	// Define a route for deleting an account
	user.Delete("/delete/:id", deleteAccountHandler)

	// Logout route
	user.Post("/logout", logoutHandler)

	// Get a list of all books
	user.Get("/books", getAllBooksHandler)

	// Get a single book by ID
	user.Get("/book/:id", getBookByIDHandler)

	// Define a middleware to protect routes that require a valid JWT
	admin := app.Group("/admin")
	admin.Use(jwtware.New(jwtware.Config{
		SigningKey: []byte(os.Getenv("JWT_SECRET")),
	}))

	// Add a custom middleware to check for the "admin" role
	admin.Use(func(c *fiber.Ctx) error {
		// Get the user ID from the JWT payload
		userID := c.Locals("user").(*jwt.Token).Claims.(jwt.MapClaims)["user_id"].(float64)

		// Find the user in the database
		var user User
		if err := db.First(&user, uint(userID)).Error; err != nil {
			// Handle database errors (e.g., no user with the given ID)
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}

		// Check if the user is an admin
		if user.Role != UserRoleAdmin {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		// Continue to the next middleware
		return c.Next()
	})

	// Define a route for the admin section
	admin.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome admin!")
	})

	// Admin - Get a list of all books
	admin.Get("/books", getAllBooksHandler)

	// Admin - Get a single book by ID
	admin.Get("/book/:id", getBookByIDHandler)

	// Admin - Create a new book
	admin.Post("/book", createBookHandler)

	// Admin - Update a book by ID
	admin.Put("/book/:id", updateBookHandler)

	// Admin - Delete a book by ID
	admin.Delete("/book/:id", deleteBookHandler)

	// Admin - Get all users
	admin.Get("/users", getAllUsersHandler)

	// Admin - Get a single user by ID
	admin.Get("/user/:id", getUserByIDHandler)

	// Start the Fiber app
	port := 8080 // You can change this to your desired port
	fmt.Printf("Server is listening on port %d...\n", port)
	app.Listen(fmt.Sprintf(":%d", port))
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

	// Create a JWT token
	token, err := createToken(user.UserID)
	if err != nil {
		// Handle token creation error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot log in",
		})
	}

	// Return the token
	return c.JSON(fiber.Map{
		"success": true,
		"token":   token,
	})

}

func registerHandler(c *fiber.Ctx) error {
	var userData struct {
		FirstName string   `json:"firstname" validate:"required"`
		LastName  string   `json:"lastname" validate:"required"`
		Email     string   `json:"email" validate:"required,email"`
		Password  string   `json:"password" validate:"required"`
		Role      UserRole `json:"role" validate:"required"`
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
		Role:      userData.Role,
	}

	// Save the user to the database
	if err := db.Create(&newUser).Error; err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "User registration failed",
		})
	}

	// Retrieve the auto-generated ID from the database
	autoGeneratedID := newUser.ID

	// Create a JWT token
	token, err := createToken(autoGeneratedID)
	if err != nil {
		// Handle token creation error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot log in",
		})
	}

	// Return the token
	return c.JSON(fiber.Map{
		"success": true,
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

	// Set the token's expiration time to now thereby invalidating it
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now(),
		HTTPOnly: true,
	})

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

	// Set the token's expiration time to now thereby invalidating it
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now(),
		HTTPOnly: true,
	})

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User account deleted successfully",
	})
}

func logoutHandler(c *fiber.Ctx) error {
	// Set the token's expiration time to now thereby invalidating it
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now(),
		HTTPOnly: true,
	})
	return c.JSON(fiber.Map{
		"success": true,
		"message": "User logged out successfully",
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

	// Find the book in the database
	var book Book
	if err := db.First(&book, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Book not found",
		})
	}

	// Update the book's information
	book.Title = updatedBook.Title
	book.Author = updatedBook.Author
	book.ISBN = updatedBook.ISBN
	book.Genre = updatedBook.Genre
	book.Price = updatedBook.Price
	book.Quantity = updatedBook.Quantity
	book.Description = updatedBook.Description

	// Save the updated book to the database
	if err := db.Save(&book).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update book",
		})
	}

	return c.JSON(book)
}

// Delete a book by ID
func deleteBookHandler(c *fiber.Ctx) error {
	id := c.Params("id")

	// Find the book in the database
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

// Get all users
func getAllUsersHandler(c *fiber.Ctx) error {
	var users []User
	if err := db.Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch users",
		})
	}
	return c.JSON(users)
}

// Get a single user by ID
func getUserByIDHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var user User
	if err := db.First(&user, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}
	return c.JSON(user)
}

// Create JWT token
func createToken(userID uint) (string, error) {
	// Define the payload
	payload := jwt.MapClaims{}
	payload["user_id"] = userID
	payload["exp"] = time.Now().Add(time.Hour * 24).Unix() // 24 hours

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	// Generate the encoded token
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
