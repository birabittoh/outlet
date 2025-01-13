package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type Task struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
	TaskID    uint      `json:"task_id"`
}

const pageContent = `
<!DOCTYPE html>
<html>
<head>
	<title>Tasks</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		body { font-family: Arial, sans-serif; margin: 0; padding: 20px; }
		h1 { color: #333; }
		a { color: #007BFF; text-decoration: none; }
		a:hover { text-decoration: underline; }
		main { max-width: 600px; margin: 50px auto; }
	</style>
</head>
<body>
	<main>
	<h1>You did %d tasks today.</h1>
	<p><a href="/tasks">When?</a></p>

	<form method="POST" action="/new" target="_blank">
		<input type="password" name="token" placeholder="Token">
		<input type="number" name="task_id" placeholder="Task ID">
		<button type="submit">Add</button>
	</form>

	<p>Server time: %s</p>
	</main>
</body>
</html>`

var (
	db    *gorm.DB
	token string
)

func init() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: No .env file found")
	}

	token = os.Getenv("TOKEN")
	if token == "" {
		panic("TOKEN not set in .env file")
	}

	var err error
	db, err = gorm.Open(sqlite.Open("data.sqlite"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(&Task{})
	if err != nil {
		panic(err)
	}
}

func getTodayTasks() (out []Task, err error) {
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	err = db.Where("created_at >= ? AND created_at < ?", today, tomorrow).Find(&out).Error
	return
}

func newTask(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	tkn := r.FormValue("token")
	if tkn != token {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	taskIDStr := r.FormValue("task_id")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid task_id", http.StatusBadRequest)
		return
	}

	if err := db.Create(&Task{TaskID: uint(taskID)}).Error; err != nil {
		http.Error(w, "Failed to save in sqlite", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func todayTasks(w http.ResponseWriter, r *http.Request) {
	tt, err := getTodayTasks()
	if err != nil {
		http.Error(w, "Failed to query sqlite", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tt)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tt, err := getTodayTasks()
	if err != nil {
		http.Error(w, "Failed to query sqlite", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(pageContent, len(tt), time.Now().Format(time.RFC3339))))
}

func main() {
	address := ":3000"
	http.HandleFunc("POST /new", newTask)
	http.HandleFunc("GET /tasks", todayTasks)
	http.HandleFunc("GET /", indexHandler)

	fmt.Println("Listening on", address)
	http.ListenAndServe(address, nil)
}
