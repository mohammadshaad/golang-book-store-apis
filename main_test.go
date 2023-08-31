package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/mohammadshaad/golang-book-store-backend/routes"
)

func TestLoginHandler(t *testing.T) {
	// Create a new Fiber app for testing
	app := setupTestApp()

	// Define a test case with sample request data
	reqData := `{"email": "anam@user.com", "password": "anam"}`

	// Use Fiber's testing utilities to simulate the request
	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(reqData)))

	// Check for errors
	if err != nil {
		t.Fatalf("Failed to perform test request: %v", err)
	}

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Check the response body or other assertions as needed
	expectedToken := `{"token": "your_expected_token_value"}`
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	if string(bodyBytes) != expectedToken {
		t.Errorf("Expected response body:\n%s\nGot:\n%s", expectedToken, string(bodyBytes))
	}

	// Log the request data
	t.Logf("Request Data: %s", reqData)

	// Log the response body
	t.Logf("Response Body: %s", string(bodyBytes))

}

// Helper function to set up a Fiber app for testing
func setupTestApp() *fiber.App {
	app := fiber.New()

	// Set up your routes here, similar to your main function
	app.Post("/login", routes.LoginHandler)

	return app
}
