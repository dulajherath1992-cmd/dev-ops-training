package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Task struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

var tasks = []Task{
	{
		ID:        1,
		Title:     "Learn Go",
		Completed: false,
	},
	{
		ID:        2,
		Title:     "Learn Docker",
		Completed: false,
	},
}

func main() {

	databaseURL := os.Getenv("DATABASE_URL")

	db, err := pgxpool.New(context.Background(), databaseURL)

	if err != nil {
		log.Fatal("Unable to connect to database:", err)
	}

	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Fatal("Database ping failed:", err)
	}

	log.Println("Connected to PostgreSQL")

	app := fiber.New()

	app.Use(cors.New())

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("DevOps Training API is running")
	})

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
		})
	})

	// Get all tasks
	app.Get("/tasks", func(c *fiber.Ctx) error {

		rows, err := db.Query(
			context.Background(),
			"SELECT id, title, completed FROM tasks ORDER BY id",
		)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "failed to get tasks",
			})
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
				return c.Status(500).JSON(fiber.Map{
					"error": "failed to read task",
				})
			}

			tasks = append(tasks, task)
		}

		return c.JSON(tasks)
	})

	// Create task
	app.Post("/tasks", func(c *fiber.Ctx) error {
		var task Task

		if err := c.BodyParser(&task); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid request",
			})
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
			return c.Status(500).JSON(fiber.Map{
				"error": "failed to create task",
			})
		}

		return c.Status(201).JSON(task)
	})

	// Update task
	app.Put("/tasks/:id", func(c *fiber.Ctx) error {
		id, err := strconv.Atoi(c.Params("id"))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid task id",
			})
		}

		var task Task

		if err := c.BodyParser(&task); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid request",
			})
		}

		err = db.QueryRow(
			context.Background(),
			`
		UPDATE tasks
		SET title = $1, completed = $2
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
			return c.Status(404).JSON(fiber.Map{
				"error": "task not found",
			})
		}

		return c.JSON(task)
	})

	// Delete task
	app.Delete("/tasks/:id", func(c *fiber.Ctx) error {
		id, err := strconv.Atoi(c.Params("id"))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid task id",
			})
		}

		result, err := db.Exec(
			context.Background(),
			"DELETE FROM tasks WHERE id = $1",
			id,
		)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "failed to delete task",
			})
		}

		if result.RowsAffected() == 0 {
			return c.Status(404).JSON(fiber.Map{
				"error": "task not found",
			})
		}

		return c.SendStatus(204)
	})

	app.Listen(":8080")
}
