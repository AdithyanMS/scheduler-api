package router

import (
	"ss/middleware"

	"github.com/gorilla/mux"
)

// Router is exported and used in main.go
func Router() *mux.Router {

	router := mux.NewRouter()

	router.HandleFunc("/schedule/dbUpdate", middleware.DbUpdate).Methods("POST", "OPTIONS")
	router.HandleFunc("/schedule/inputCSV", middleware.Input).Methods("POST", "OPTIONS")
	router.HandleFunc("/schedule/generate/{id}", middleware.Generate).Methods("GET", "OPTIONS")
	router.HandleFunc("/schedule/receiveCSV/{name}", middleware.Receive).Methods("GET", "OPTIONS")

	return router
}
