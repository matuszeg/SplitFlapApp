package restfulApi

import (
	"github.com/gorilla/mux"
	"net/http"
)

type RestfulRouter struct {
	Router *mux.Router
}

func NewRestfulRouter() *RestfulRouter {
	router := new(RestfulRouter)

	muxRouter := mux.NewRouter()
	muxRouter.Headers("Access-Control-Allow-Origin", "*")

	muxRouter.HandleFunc("/status", StatusCheck).Methods(http.MethodGet)

	router.Router = muxRouter

	return router
}

func StatusCheck(w http.ResponseWriter, r *http.Request) {
	response := JsonResponse[string]{Data: "Success"}
	response.SendResponse(w)
}
