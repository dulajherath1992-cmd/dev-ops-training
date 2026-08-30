package main

import (
	"context"
	"fmt"
	"log"
	"os"

	fiberprometheus "github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Task struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

var db *pgxpool.Pool

func main() {

	// --------------------------------------------------
	// PostgreSQL
	// --------------------------------------------------

	connectDatabase()

	defer db.Close()

	// --------------------------------------------------
	// Fiber
	// --------------------------------------------------

	app := fiber.New()

	// --------------------------------------------------
	// CORS
	// --------------------------------------------------

	app.Use(cors.New())

	// --------------------------------------------------
	// Prometheus
	// --------------------------------------------------

	prometheus := fiberprometheus.New("devops-training-backend")

	prometheus.RegisterAt(app, "/metrics")

	app.Use(prometheus.Middleware)

	// --------------------------------------------------
	// Routes
	// --------------------------------------------------

	app.Get("/", home)

	app.Get("/health", health)

	app.Get("/tasks", getTasks)

	app.Post("/tasks", createTask)

	app.Put("/tasks/:id", updateTask)

	app.Delete("/tasks/:id", deleteTask)

	// --------------------------------------------------
	// Start Server
	// --------------------------------------------------

	log.Println("Backend running on port 8080")

	log.Fatal(app.Listen(":8080"))
}

// ======================================================
// Database Connection
// ======================================================

func connectDatabase() {

	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {

		host := getEnv("DB_HOST", "postgres")
		port := getEnv("DB_PORT", "5432")
		user := getEnv("DB_USER", "postgres")
		password := getEnv("DB_PASSWORD", "postgres")
		dbName := getEnv("DB_NAME", "devops")

		databaseURL = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=disable",
			user,
			password,
			host,
			port,
			dbName,
		)
	}

	var err error

	db, err = pgxpool.New(
		context.Background(),
		databaseURL,
	)

	if err != nil {
		log.Fatal("Unable to create database connection pool:", err)
	}

	// Check that PostgreSQL is actually reachable
	if err := db.Ping(context.Background()); err != nil {
		log.Fatal("Unable to connect to PostgreSQL:", err)
	}

	log.Println("Connected to PostgreSQL")
}

// ======================================================
// Home
// ======================================================

func home(c *fiber.Ctx) error {

	return c.JSON(fiber.Map{
		"message": "DevOps Training API is running",
	})
}

// ======================================================
// Health
// ======================================================

func health(c *fiber.Ctx) error {

	if err := db.Ping(context.Background()); err != nil {

		return c.Status(fiber.StatusServiceUnavailable).JSON(
			fiber.Map{
				"status":   "unhealthy",
				"database": "disconnected",
			},
		)
	}

	return c.JSON(
		fiber.Map{
			"status":   "healthy",
			"database": "connected",
		},
	)
}

// ======================================================
// GET /tasks
// ======================================================

func getTasks(c *fiber.Ctx) error {

	rows, err := db.Query(
		context.Background(),
		`
		SELECT id, title, completed
		FROM tasks
		ORDER BY id
		`,
	)

	if err != nil {

		log.Println("Error getting tasks:", err)

		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{
				"error": "Failed to get tasks",
			},
		)
	}

	defer rows.Close()

	tasks := []Task{}

	for rows.Next() {

		var task Task

		err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Completed,
		)

		if err != nil {

			log.Println("Error reading task:", err)

			return c.Status(fiber.StatusInternalServerError).JSON(
				fiber.Map{
					"error": "Failed to read task",
				},
			)
		}

		tasks = append(tasks, task)
	}

	return c.JSON(tasks)
}

// ======================================================
// POST /tasks
// ======================================================

func createTask(c *fiber.Ctx) error {

	var task Task

	if err := c.BodyParser(&task); err != nil {

		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": "Invalid request body",
			},
		)
	}

	if task.Title == "" {

		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": "Task title is required",
			},
		)
	}

	err := db.QueryRow(
		context.Background(),
		`
		INSERT INTO tasks (title, completed)
		VALUES ($1, $2)
		RETURNING id
		`,
		task.Title,
		task.Completed,
	).Scan(&task.ID)

	if err != nil {

		log.Println("Error creating task:", err)

		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{
				"error": "Failed to create task",
			},
		)
	}

	return c.Status(fiber.StatusCreated).JSON(task)
}

// ======================================================
// PUT /tasks/:id
// ======================================================

func updateTask(c *fiber.Ctx) error {

	id := c.Params("id")

	var task Task

	if err := c.BodyParser(&task); err != nil {

		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": "Invalid request body",
			},
		)
	}

	if task.Title == "" {

		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"error": "Task title is required",
			},
		)
	}

	err := db.QueryRow(
		context.Background(),
		`
		UPDATE tasks
		SET title = $1,
		    completed = $2
		WHERE id = $3
		RETURNING id, title, completed
		`,
		task.Title,
		task.Completed,
		id,
	).Scan(
		&task.ID,
		&task.Title,
		&task.Completed,
	)

	if err != nil {

		log.Println("Error updating task:", err)

		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{
				"error": "Failed to update task",
			},
		)
	}

	return c.JSON(task)
}

// ======================================================
// DELETE /tasks/:id
// ======================================================

func deleteTask(c *fiber.Ctx) error {

	id := c.Params("id")

	result, err := db.Exec(
		context.Background(),
		`
		DELETE FROM tasks
		WHERE id = $1
		`,
		id,
	)

	if err != nil {

		log.Println("Error deleting task:", err)

		return c.Status(fiber.StatusInternalServerError).JSON(
			fiber.Map{
				"error": "Failed to delete task",
			},
		)
	}

	if result.RowsAffected() == 0 {

		return c.Status(fiber.StatusNotFound).JSON(
			fiber.Map{
				"error": "Task not found",
			},
		)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ======================================================
// Environment Helper
// ======================================================

func getEnv(key string, fallback string) string {

	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
