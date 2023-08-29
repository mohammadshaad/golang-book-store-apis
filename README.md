

# Balkan ID Internship Project Documentation

## Introduction
Welcome to the documentation for the Bookstore Application, submitted as part of my application for the Balkan ID internship. This document provides a comprehensive overview of the application, its features, and how to set it up and use it.

### Project Overview
The Bookstore Application is a web-based platform that allows users to browse, purchase, and manage books. It offers user registration and authentication, book management, and shopping cart functionality.

### Technologies Used
- Go (Golang)
- Fiber (Web framework)
- PostgreSQL (Database)
- GORM (Object-Relational Mapping)
- JSON Web Tokens (JWT) for authentication
- React.js (Frontend)

## Getting Started
To run and test the application, please follow these steps:

### Prerequisites
Before you begin, ensure you have the following installed:

- Go programming language
- PostgreSQL database
- Required Go packages (dependencies managed via Go modules)

### Installation
1. Clone the repository: `git clone https://github.com/mohammadshaad/golang-book-store-backend.git`
2. Navigate to the project directory: `cd bookstore`
3. Create a `.env` file and configure the necessary environment variables (see [Configuration](#configuration) section).
4. Run database migrations: `go run main.go migrate`
5. Start the application: `go run main.go`

## Application Structure
The project is organized as follows:

- `main.go`: Entry point of the application.
- `tests/`: Unit and integration tests.

## Configuration
The application reads configuration settings from environment variables. Here are the key variables to configure:

- `DB_HOST`: PostgreSQL database host address.
- `DB_PORT`: PostgreSQL database port.
- `DB_NAME`: PostgreSQL database name.
- `DB_USER`: PostgreSQL database username.
- `DB_PASSWORD`: PostgreSQL database password.
- `JWT_SECRET`: Secret key for JWT token generation.

Example `.env` file:
```env
DB_HOST=localhost
DB_PORT=5432
DB_NAME=bookstore_db
DB_USER=myuser
DB_PASSWORD=mypassword
JWT_SECRET=mysecretkey
```

## Features

### User Authentication
- **User Registration:** Users can create new accounts by providing their email and password.
- **User Login:** Registered users can log in to access their account.

### Book Management
- **Book Listing:** Users can view a list of available books.
- **Book Details:** Users can view detailed information about a specific book.
- **Book Search:** Users can search for books by title or author.
- **Book Addition:** Admin users can add new books to the catalog.
- **Book Modification:** Admin users can update book details.
- **Book Deletion:** Admin users can remove books from the catalog.

### Shopping Cart
- **Cart Management:** Users can add books to their shopping cart, view the cart, remove items, and update quantities.
- **Checkout:** Users can proceed to checkout, where they can review their order and complete the purchase.

### Admin Features
- **Admin Access:** Certain routes and features are accessible only to admin users.
- **User Management:** Admin users can manage user accounts, including user activation, deactivation, and deletion.
- **Book Management:** Admin users can manage the catalog of books, including adding, modifying, and deleting entries.

## Testing
To run tests, use the following command:

```shell
go test ./...
```

## Deployment
For production deployment, follow these steps:

1. Set up a production-ready PostgreSQL database.
2. Configure the environment variables for the production environment.
3. Build the application: `go build -o bookstore-app main.go`
4. Deploy the binary to your production server.
5. Set up a reverse proxy (e.g., Nginx) to serve the application.

## Troubleshooting
If you encounter any issues or have questions, please contact Mohammad Shaad at callshaad@gmail.com.

## Conclusion
Thank you for considering my application. I hope you find this Bookstore Application and its documentation useful. Please feel free to reach out if you have any further questions or require additional information.

Sincerely,

Mohammad Shaad
