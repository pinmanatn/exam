package main

import (
	"strings"
	"context"
	"net/http"
  	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"os"
)

type Question struct {
	ID       int    `json:"id"`
	Question string `json:"question"`
	Choice1  string `json:"choice1"`
	Choice2  string `json:"choice2"`
	Choice3  string `json:"choice3"`
	Choice4  string `json:"choice4"`
}

type Answer struct {
	QuestionID int `json:"question_id"`
	Answer     int `json:"answer"`
}

type SubmitRequest struct {
	StudentName string   `json:"student_name"`
	Answers     []Answer `json:"answers"`
}

func main() {
	godotenv.Load()
	conn, err := pgx.Connect(
		context.Background(),
		os.Getenv("DATABASE_URL"),
	)

	if err != nil {
		panic(err)
	}

	defer conn.Close(context.Background())

  r := gin.Default()
  r.Use(cors.Default())

	r.GET("/api/questions", func(c *gin.Context) {

		rows, err := conn.Query(
			context.Background(),
			`SELECT id,question,choice1,choice2,choice3,choice4
			 FROM questions
			 ORDER BY id`,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		defer rows.Close()

		var questions []Question

		for rows.Next() {

			var q Question

			rows.Scan(
				&q.ID,
				&q.Question,
				&q.Choice1,
				&q.Choice2,
				&q.Choice3,
				&q.Choice4,
			)

			questions = append(questions, q)
		}

		c.JSON(http.StatusOK, questions)
	})
	r.POST("/api/submit", func(c *gin.Context) {

		var req SubmitRequest
	
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"error": err.Error(),
			})
			return
		}
		if strings.TrimSpace(req.StudentName) == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "**โปรดระบุชื่อผู้สอบ",
			})
			return
		}

		var totalQuestions int
		err = conn.QueryRow(
			context.Background(),
			`SELECT COUNT(*) FROM questions`,
		).Scan(&totalQuestions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		if len(req.Answers) != totalQuestions {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "**กรุณาตอบคำถามให้ครบทุกข้อ",
			})
			return
		}
		score := 0
	
		for _, ans := range req.Answers {
	
			var correctAnswer int
	
			err := conn.QueryRow(
				context.Background(),
				`
				SELECT correct_answer
				FROM questions
				WHERE id = $1
				`,
				ans.QuestionID,
			).Scan(&correctAnswer)
	
			if err != nil {
				continue
			}
	
			if ans.Answer == correctAnswer {
				score++
			}
		}
		
		_, err = conn.Exec(
			context.Background(),
			`
			INSERT INTO exam_results
			(student_name, score)
			VALUES ($1,$2)
			`,
			req.StudentName,
			score,
		)

		c.JSON(200, gin.H{
			"score": score,
		})
	})
	// React Static Files
	r.Static("/assets", "./public/assets")

	r.GET("/", func(c *gin.Context) {
		c.File("./public/index.html")
	})

	r.Run(":8080")
}