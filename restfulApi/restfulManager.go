package restfulApi

import (
	"fmt"
	"github.com/gorilla/handlers"
	"net/http"
)

type RestfulManager struct {
	Router *RestfulRouter
}

func NewRestfulManager() *RestfulManager {
	restfulManager := new(RestfulManager)

	router := NewRestfulRouter()

	restfulManager.Router = router

	return restfulManager
}

func (r *RestfulManager) Start(bSeparateProcess bool) {
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Origin", "Content-Type", "Accept", "Token"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"})

	fmt.Println("Starting Restful Api at localhost:8888")
	if bSeparateProcess {
		go func() {
			err := http.ListenAndServe("localhost:8888", handlers.CORS(originsOk, headersOk, methodsOk)(r.Router.Router))
			if err != nil {
				panic(err)
				return
			}
		}()
	} else {
		err := http.ListenAndServe("localhost:8888", handlers.CORS(originsOk, headersOk, methodsOk)(r.Router.Router))
		if err != nil {
			panic(err)
			return
		}
	}
}
