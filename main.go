package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"github.com/mosgizy/rssagg/internal/database"

	_ "github.com/lib/pq"
)

type apiConfig struct{
	DB *database.Queries
}

func main() {
	godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT is not found in the environment")
	}

	dbUrl := os.Getenv("DB_URL")
	if dbUrl == "" {
		log.Fatal("DB_URL is not found in the environment")
	}

	conn, err := sql.Open("postgres",dbUrl)
	if err != nil {
		log.Fatal("Can't connect to database")
	}

	db := database.New(conn)

	apiCfg := apiConfig{
		DB: db,
	}
	
	go startScrapping(db,10, time.Minute)

	router := chi.NewRouter()

	router.Use((cors.Handler((cors.Options{
		AllowedOrigins: []string{"https://*","http://*"},
		AllowedMethods: []string{"GET","POST","DELETE","OPTIONS"},
		AllowedHeaders: []string{"*"},
		ExposedHeaders: []string{"Link"},
		AllowCredentials: false,
		MaxAge: 300,
	}))))

	v1Router := chi.NewRouter()
	v1Router.Get("/healthz", handlerReadiness)
	v1Router.Get("/err",handlerErr)
	v1Router.Post("/users", apiCfg.handlerCreateUser)
	v1Router.Get("/users", apiCfg.middlewareAuth(apiCfg.handlerGetUser))

	v1Router.Post("/feeds",apiCfg.middlewareAuth(apiCfg.handlerCreateFeed))
	v1Router.Get("/feeds",apiCfg.handlerGetFeeds)

	v1Router.Get("/posts",apiCfg.middlewareAuth(apiCfg.handlerGetPostsForUser))

	v1Router.Post("/feed_follows",apiCfg.middlewareAuth(apiCfg.handlerCreateFeedFollows))
	v1Router.Get("/feed_follows",apiCfg.middlewareAuth(apiCfg.handlerGetFeedFollows))
	v1Router.Delete("/feed_follows/{feedFollowID}",apiCfg.middlewareAuth(apiCfg.handlerDeleteFeedFollows))

	router.Mount("/v1",v1Router)

	srv := &http.Server{
		Handler: router,
		Addr: ":" + port,
	}

	log.Printf("Server starting on port %v", port)

	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Port:", port)
}     