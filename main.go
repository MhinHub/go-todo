package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"todolist/database"

	"github.com/labstack/echo/v4"
)

// Todo struct tetap sama, merepresentasikan data di database.
type Todo struct {
	Id          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      bool   `json:"status"`
}

// UpdateTodoPayload digunakan khusus untuk binding data saat PATCH/update.
// Menggunakan pointer (*) memungkinkan kita membedakan antara nilai yang tidak dikirim
// (nil) dengan nilai default (misalnya string kosong "" atau boolean false).
type UpdateTodoPayload struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *bool   `json:"status"`
}

func main() {
	db := database.InitDb()
	defer db.Close()

	if err := db.Ping(); err != nil {
		// Menggunakan log.Fatal akan mencetak error dan menghentikan aplikasi
		// dengan cara yang lebih bersih daripada panic().
		log.Fatalf("Database connection failed: %v", err)
	}

	e := echo.New()

	// --- CREATE ---
	e.POST("/todos", func(c echo.Context) error {
		var newTodo Todo

		// Gunakan c.Bind() yang lebih idiomatik di Echo untuk parsing request body.
		if err := c.Bind(&newTodo); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"message": err.Error()})
		}

		// Validasi sederhana
		if newTodo.Title == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"message": "Title cannot be empty"})
		}

		err := db.QueryRow(
			"INSERT INTO todos (title, description, status) VALUES ($1, $2, $3) RETURNING id",
			newTodo.Title, newTodo.Description, newTodo.Status,
		).Scan(&newTodo.Id)

		if err != nil {
			log.Printf("Error creating todo: %v", err) // Log error di server
			// Jangan kirim detail error database ke client.
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to create todo"})
		}

		// Gunakan status 201 Created untuk resource yang baru dibuat.
		return c.JSON(http.StatusCreated, map[string]any{
			"status":  "success",
			"data":    newTodo,
			"message": "Todo created successfully",
		})
	})

	// --- READ ALL ---
	e.GET("/todos", func(c echo.Context) error {
		rows, err := db.Query("SELECT id, title, description, status FROM todos ORDER BY id ASC")
		if err != nil {
			log.Printf("Error fetching todos: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to fetch todos"})
		}
		defer rows.Close()

		var todos []Todo
		for rows.Next() {
			var todo Todo
			if err := rows.Scan(&todo.Id, &todo.Title, &todo.Description, &todo.Status); err != nil {
				log.Printf("Error scanning todo row: %v", err)
				return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to process data"})
			}
			todos = append(todos, todo)
		}

		return c.JSON(http.StatusOK, map[string]any{
			"status":  "success",
			"data":    todos,
			"message": "Todos fetched successfully",
		})
	})

	// --- UPDATE ---
	e.PATCH("/todos/:id", func(c echo.Context) error {
		id := c.Param("id")
		var payload UpdateTodoPayload

		if err := c.Bind(&payload); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"message": err.Error()})
		}

		// Membangun query secara dinamis tapi dengan cara yang AMAN dari SQL Injection.
		setParts := []string{}
		args := []any{}
		argId := 1

		if payload.Title != nil {
			setParts = append(setParts, fmt.Sprintf("title = $%d", argId))
			args = append(args, *payload.Title)
			argId++
		}
		if payload.Description != nil {
			setParts = append(setParts, fmt.Sprintf("description = $%d", argId))
			args = append(args, *payload.Description)
			argId++
		}
		if payload.Status != nil {
			setParts = append(setParts, fmt.Sprintf("status = $%d", argId))
			args = append(args, *payload.Status)
			argId++
		}

		if len(setParts) == 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"message": "No fields to update"})
		}

		// Menambahkan `id` sebagai argumen terakhir
		args = append(args, id)
		query := fmt.Sprintf("UPDATE todos SET %s WHERE id = $%d RETURNING id, title, description, status",
			strings.Join(setParts, ", "), argId)

		var updatedTodo Todo
		err := db.QueryRow(query, args...).Scan(&updatedTodo.Id, &updatedTodo.Title, &updatedTodo.Description, &updatedTodo.Status)

		if err != nil {
			// Cek jika errornya karena todo tidak ditemukan
			if err == sql.ErrNoRows {
				return c.JSON(http.StatusNotFound, map[string]string{"message": "Todo not found"})
			}
			log.Printf("Error updating todo: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to update todo"})
		}

		return c.JSON(http.StatusOK, map[string]any{
			"status":  "success",
			"data":    updatedTodo, // Mengembalikan data yang sudah terupdate
			"message": "Todo updated successfully",
		})
	})

	// --- DELETE ---
	e.DELETE("/todos/:id", func(c echo.Context) error {
		id := c.Param("id")

		result, err := db.Exec("DELETE FROM todos WHERE id = $1", id)
		if err != nil {
			log.Printf("Error deleting todo: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Failed to delete todo"})
		}

		// Cek apakah ada baris yang benar-benar terhapus.
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("Error checking rows affected: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "An error occurred"})
		}

		if rowsAffected == 0 {
			return c.JSON(http.StatusNotFound, map[string]string{"message": "Todo not found"})
		}

		// Status 200 OK dengan pesan, atau bisa juga 204 No Content tanpa body.
		return c.JSON(http.StatusOK, map[string]any{
			"status":  "success",
			"message": "Todo deleted successfully",
		})
	})

	// Menangani error yang mungkin terjadi saat server dimulai (misal: port sudah dipakai).
	log.Println("Starting server on :8080")
	if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
		e.Logger.Fatal("shutting down the server")
	}
}
