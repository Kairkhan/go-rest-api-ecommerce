package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type App struct {
	Router *mux.Router
	DB     *sql.DB
}

func (a *App) Initialize(user, password, dbname string) {
	connectionString :=
		fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, password, dbname)

	var err error
	a.DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	a.Router = mux.NewRouter()

	a.initializeRoutes()
}

func respondWithJSON(writer http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)
	writer.Write(response)
}

func respondWithError(writer http.ResponseWriter, code int, message string) {
	respondWithJSON(writer, code, map[string]string{"error" : message})
}

func (a *App) getProduct(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id, err := strconv.Atoi(vars["id"])

	if err != nil {
		respondWithError(writer, http.StatusBadRequest, "Invalid product ID")
		return
	}

	p := product{ID: id}
	if err := p.getProduct(a.DB); err != nil {

		switch err {
		case sql.ErrNoRows: 
			respondWithError(writer, http.StatusNotFound, "Product not found")
		default:
			respondWithError(writer, http.StatusInternalServerError, err.Error())
		}

		return
	}

	respondWithJSON(writer, http.StatusOK, p)

}

func (a *App) getProducts(writer http.ResponseWriter, request *http.Request) {
	count, _ := strconv.Atoi(request.FormValue("count"))
	start, _ := strconv.Atoi(request.FormValue("start"))

	if count > 10 || count < 1 {
		count = 10
	}

	if start < 0 {
		start = 0
	}

	products, err := getProducts(a.DB, start, count)
	
	if err != nil {
		respondWithError(writer, http.StatusInternalServerError, err.Error())
		return 
	}

	respondWithJSON(writer, http.StatusOK, products)
}

func (a *App) createProduct(writer http.ResponseWriter, request *http.Request) {
	var p product
	decoder := json.NewDecoder(request.Body)

	if err := decoder.Decode(&p); err != nil {
		respondWithError(writer, http.StatusBadRequest, "Invalid request payload")
		return
	}

	defer request.Body.Close()

	if err := p.createProduct(a.DB); err != nil {
		respondWithError(writer, http.StatusInternalServerError, err.Error())
		return 
	}

	respondWithJSON(writer, http.StatusCreated, p)
}

func (a *App) updateProduct(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id, err := strconv.Atoi(vars["id"])

	if err != nil {
		respondWithError(writer, http.StatusBadRequest, "Invalid product ID")
		return
	}

	var p product
	decoder := json.NewDecoder(request.Body)

	if err := decoder.Decode(&p); err != nil {
		respondWithError(writer, http.StatusBadRequest, "Invalid request payload")
		return
	}

	defer request.Body.Close()
	p.ID = id

	if err := p.updateProduct(a.DB); err != nil {
		respondWithError(writer, http.StatusInternalServerError, err.Error())
	}

	respondWithJSON(writer, http.StatusOK, p)
}

func (a *App) deleteProduct(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id, err := strconv.Atoi(vars["id"])

	if err != nil {
		respondWithError(writer, http.StatusBadRequest, "Invalid product ID")
		return 
	}

	p := product{ID: id}

	if err := p.deleteProduct(a.DB); err != nil {
		respondWithError(writer, http.StatusInternalServerError, err.Error())
		return 
	}

	respondWithJSON(writer, http.StatusOK, map[string]string{"result" : "success"})
}

func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/products", a.getProducts).Methods("GET")
	a.Router.HandleFunc("/products", a.createProduct).Methods("POST")
	a.Router.HandleFunc("/products/{id:[0-9]+}", a.getProduct).Methods("GET")
	a.Router.HandleFunc("/products/{id:[0-9]+}", a.updateProduct).Methods("PUT")
	a.Router.HandleFunc("/products/{id:[0-9]+}", a.deleteProduct).Methods("DELETE")


}

func (a *App) Run(addr string) {
	log.Fatal(http.ListenAndServe(":8010", a.Router))
}
