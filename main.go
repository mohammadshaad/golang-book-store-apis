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
	Image       string  `json:"image"`
	Path        string  `json:"path"`
}

// Define a struct to represent a cart item
type CartItem struct {
	gorm.Model
	UserID   uint `json:"user_id"`
	BookID   uint `json:"book_id"`
	Quantity uint `json:"quantity"`
}

type Review struct {
	gorm.Model
	BookID  uint   `json:"book_id"`
	UserID  uint   `json:"user_id"`
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
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

	// // Define the database connection string using environment variables
	// dbHost := os.Getenv("DB_HOST")
	// dbPort := os.Getenv("DB_PORT")
	// dbUser := os.Getenv("DB_USER")
	// dbPassword := os.Getenv("DB_PASSWORD")
	// dbName := os.Getenv("DB_NAME")

	// DB_HOST := os.Getenv("DB_HOST")
	// DB_PORT := os.Getenv("DB_PORT")
	// DB_USER := os.Getenv("DB_USER")
	// DB_PASSWORD := os.Getenv("DB_PASSWORD")
	// DB_NAME := os.Getenv("DB_NAME")

	// // Define the database connection string
	// connStr := fmt.Sprintf(
	// 	"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
	// 	dbHost, dbPort, dbUser, dbPassword, dbName,
	// )

	connStr := "postgresql://postgres:Z6mMaUDtLKJyaoE1f3kg@containers-us-west-191.railway.app:7080/railway"

	// Open the database connection
	var err error
	db, err = gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		panic("failed to connect to the database")
	}

	// Auto-migrate the User, Book, Cart Items & Review model to create the users table
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Book{})
	db.AutoMigrate(&CartItem{})
	db.AutoMigrate(&Review{})

	// Create a Fiber app
	app := fiber.New()

	// Define a route for the home page
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome to the book store!")
	})

	// Define routes
	setupRoutes(app)

	// Start the Fiber app
	port := 8080 // You can change this to your desired port
	fmt.Printf("Server is listening on port %d...\n", port)
	app.Listen("0.0.0.0:"  + strconv.Itoa(port))
}

// setupRoutes defines all the routes and their handlers
func setupRoutes(app *fiber.App) {
	// Public routes
	app.Post("/register", registerHandler)
	app.Post("/login", loginHandler)

	// User routes (protected by JWT)
	user := app.Group("/user")
	user.Use(jwtware.New(jwtware.Config{
		SigningKey: []byte(os.Getenv("JWT_SECRET")),
	}))
	user.Use(checkJWTValidity)

	user.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome user!")
	})
	user.Get("/profile/:id", profile)
	user.Put("/profile/:id", updateProfile)
	user.Put("/deactivate/:id", deactivateAccountHandler)
	user.Put("/activate/:id", activateAccountHandler)
	user.Delete("/delete/:id", deleteAccountHandler)
	user.Post("/logout", logoutHandler)
	user.Get("/books", getAllBooksHandler)
	user.Get("/book/:id", getBookByIDHandler)
	user.Post("/cart", addToCartHandler)
	user.Get("/cart", getCartHandler)
	user.Delete("/cart/:book_id", removeFromCartHandler)
	user.Put("/cart/:book_id", updateCartItemQuantityHandler)
	user.Post("/book/:book_id/reviews", addReviewHandler)
	user.Get("/book/:book_id/reviews", getBookReviewsHandler)
	user.Get("/book/:id/download", downloadBookHandler)

	// Admin routes (protected by JWT and requires admin role)
	admin := app.Group("/admin")
	admin.Use(jwtware.New(jwtware.Config{
		SigningKey: []byte(os.Getenv("JWT_SECRET")),
	}))
	admin.Use(checkJWTValidity)
	admin.Use(checkAdminRole)

	admin.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome admin!")
	})
	admin.Get("/books", getAllBooksHandler)
	admin.Get("/book/:id", getBookByIDHandler)
	admin.Post("/book", createBookHandler)
	admin.Put("/book/:id", updateBookHandler)
	admin.Delete("/book/:id", deleteBookHandler)
	admin.Get("/users", getAllUsersHandler)
	admin.Get("/user/:id", getUserByIDHandler)
	admin.Get("/book/:id/download", downloadBookHandler)
	admin.Get("/book/:id/reviews", getBookReviewsHandler)
	admin.Get("/cart", getAllCartItemsHandler)
	admin.Get("/cart/:user_id", getUserCartHandler)
	admin.Delete("/cart/:user_id/:book_id", deleteCartItemHandler)
	admin.Post("/logout", logoutHandler)
}

// checkJWTValidity middleware checks if the JWT is valid
func checkJWTValidity(c *fiber.Ctx) error {
	token := c.Locals("user").(*jwt.Token)
	if token == nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Login first",
		})
	}
	return c.Next()
}

// checkAdminRole middleware checks if the user has the "admin" role
func checkAdminRole(c *fiber.Ctx) error {
	// Get the user ID from the JWT payload
	userID := c.Locals("user").(*jwt.Token).Claims.(jwt.MapClaims)["user_id"].(float64)

	// Find the user in the database
	var user User
	if err := db.First(&user, uint(userID)).Error; err != nil {
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

	return c.Next()
}

// Handlers for the routes

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
	token, err := createToken(user.ID)
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

	// Return a success response
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
	book.Image = updatedBook.Image
	book.Path = updatedBook.Path

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

// Create a new cart item and add it to the user's cart
func addToCartHandler(c *fiber.Ctx) error {
	// Parse the user ID from the JWT token
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["user_id"].(float64))

	// Parse the book ID and quantity from the request body
	var cartItem struct {
		BookID   uint `json:"book_id" validate:"required"`
		Quantity uint `json:"quantity" validate:"required"`
	}

	if err := c.BodyParser(&cartItem); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input data",
		})
	}

	// Validate the input
	if err := validate.Struct(cartItem); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid input data",
			"errors": err.(validator.ValidationErrors),
		})
	}

	// Check if the book is already in the user's cart
	var existingCartItem CartItem
	if err := db.Where("user_id = ? AND book_id = ?", userID, cartItem.BookID).First(&existingCartItem).Error; err == nil {
		// Book is already in the cart, update the quantity
		existingCartItem.Quantity += cartItem.Quantity
		if err := db.Save(&existingCartItem).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update cart",
			})
		}
		return c.JSON(existingCartItem)
	}

	// Book is not in the cart, create a new cart item
	newCartItem := CartItem{
		UserID:   userID,
		BookID:   cartItem.BookID,
		Quantity: cartItem.Quantity,
	}

	if err := db.Create(&newCartItem).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add to cart",
		})
	}

	return c.JSON(newCartItem)
}

// Get the user's cart items
func getCartHandler(c *fiber.Ctx) error {
	// Parse the user ID from the JWT token
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["user_id"].(float64))

	// Find all cart items for the user
	var cartItems []CartItem
	if err := db.Where("user_id = ?", userID).Find(&cartItems).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch cart items",
		})
	}

	if len(cartItems) == 0 {
		return c.JSON(fiber.Map{
			"message": "Cart is empty",
		})
	}

	// Return the cart items
	return c.JSON(cartItems)
}

// Remove an item from the user's cart
func removeFromCartHandler(c *fiber.Ctx) error {
	// Parse the user ID from the JWT token
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["user_id"].(float64))

	// Parse the book ID from the URL parameter
	bookID := c.Params("book_id")

	// Find the cart item to remove
	var cartItem CartItem
	if err := db.Where("user_id = ? AND book_id = ?", userID, bookID).First(&cartItem).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Cart item not found",
		})
	}

	// Delete the cart item
	if err := db.Delete(&cartItem).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to remove item from cart",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Item removed from cart",
	})
}

// Update the quantity of a cart item
func updateCartItemQuantityHandler(c *fiber.Ctx) error {
	// Parse the user ID from the JWT token
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["user_id"].(float64))

	// Parse the book ID from the URL parameter
	bookID := c.Params("book_id")

	// Parse the new quantity from the request body
	var update struct {
		Quantity uint `json:"quantity" validate:"required"`
	}

	if err := c.BodyParser(&update); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input data",
		})
	}

	// Validate the input
	if err := validate.Struct(update); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid input data",
			"errors": err.(validator.ValidationErrors),
		})
	}

	// Find the cart item to update
	var cartItem CartItem
	if err := db.Where("user_id = ? AND book_id = ?", userID, bookID).First(&cartItem).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Cart item not found",
		})
	}

	// Update the quantity
	cartItem.Quantity = update.Quantity
	if err := db.Save(&cartItem).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update cart item quantity",
		})
	}

	return c.JSON(cartItem)
}

// Add a review for a book
func addReviewHandler(c *fiber.Ctx) error {
	// Parse the book ID from the URL parameter
	bookIDStr := c.Params("book_id")

	// Convert the book ID to a uint
	bookID, err := strconv.ParseUint(bookIDStr, 10, 32)
	if err != nil {
		// Handle invalid ID format
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID format",
		})
	}

	// Convert the book ID to a uint
	bookIDUint := uint(bookID)

	// Parse the user ID from the JWT token
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["user_id"].(float64))

	// Check if the user has already reviewed the book
	var existingReview Review
	if err := db.Where("user_id = ? AND book_id = ?", userID, bookIDUint).First(&existingReview).Error; err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "You have already reviewed this book",
		})
	}

	// Check if the book exists
	var book Book
	if err := db.First(&book, bookIDUint).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Book not found",
		})
	}

	// Check if the user exists
	var user User
	if err := db.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Parse the review data from the request body
	var review Review
	if err := c.BodyParser(&review); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input data",
		})
	}

	// Set the book ID and user ID
	review.BookID = bookIDUint
	review.UserID = userID

	// Save the review to the database
	if err := db.Create(&review).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add review",
		})
	}

	return c.JSON(review)
}

// Get reviews for a book
func getBookReviewsHandler(c *fiber.Ctx) error {
	// Parse the book ID from the URL parameter
	bookID := c.Params("book_id")

	// Find all reviews for the book
	var reviews []Review
	if err := db.Where("book_id = ?", bookID).Find(&reviews).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch reviews",
		})
	}

	if len(reviews) == 0 {
		return c.JSON(fiber.Map{
			"message": "No reviews for this book",
		})
	}

	// Return the reviews
	return c.JSON(reviews)
}

func downloadBookHandler(c *fiber.Ctx) error {
	// Parse the book ID from the URL parameter
	bookID := c.Params("id")

	// Find the book in the database by ID
	var book Book
	if err := db.First(&book, bookID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Book not found",
		})
	}

	// Get the file path
	filePath := book.Path

	return c.SendFile(filePath)
}

// Cart section for admin to see all the users cart items
func getAllCartItemsHandler(c *fiber.Ctx) error {
	var cartItems []CartItem
	if err := db.Find(&cartItems).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch cart items",
		})
	}

	if len(cartItems) == 0 {
		return c.JSON(fiber.Map{
			"message": "Cart is empty",
		})
	}

	// Return the cart items
	return c.JSON(cartItems)
}

// Get a user's cart items
func getUserCartHandler(c *fiber.Ctx) error {
	// Parse the user ID from the URL parameter
	userID := c.Params("user_id")

	// Find all cart items for the user
	var cartItems []CartItem
	if err := db.Where("user_id = ?", userID).Find(&cartItems).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch cart items",
		})
	}

	if len(cartItems) == 0 {
		return c.JSON(fiber.Map{
			"message": "Cart is empty",
		})
	}

	// Return the cart items
	return c.JSON(cartItems)
}

// Remove an item from the user's cart
func deleteCartItemHandler(c *fiber.Ctx) error {
	// Parse the user ID from the URL parameter
	userID := c.Params("user_id")

	// Parse the book ID from the URL parameter
	bookID := c.Params("book_id")

	// Find the cart item to remove
	var cartItem CartItem
	if err := db.Where("user_id = ? AND book_id = ?", userID, bookID).First(&cartItem).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Cart item not found",
		})
	}

	// Delete the cart item
	if err := db.Delete(&cartItem).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to remove item from cart",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Item removed from cart",
	})
}
