/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hazzardr/spacetraders/cmd"
	restApi "github.com/hazzardr/spacetraders/generated/api"
	"net/http"
	"time"
)

func main() {
	cmd.Execute()
}

func StartServer() {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	restApi.HandlerFromMux(&restHandler{}, r)

	http.ListenAndServe(":8080", r)
}

type restHandler struct {
}

func (r restHandler) GetUsersUserId(w http.ResponseWriter, req *http.Request, shipId int) {
	//TODO implement me
	panic("implement me")
}
