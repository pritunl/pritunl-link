package interlink

import (
	"net/http"
	"time"

	"github.com/pritunl/pritunl-link/utils"
	"github.com/sirupsen/logrus"
)

type server struct {
	server *http.Server
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/check" {
		utils.WriteText(w, 200, "ok")
		return
	}

	utils.WriteStatus(w, 404)
	return
}

func (s *server) Run() {
	for {
		err := s.server.ListenAndServe()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Error("interlink: Server failure")
		}

		time.Sleep(3 * time.Second)
	}
}

func (s *server) Init() (err error) {
	s.server = &http.Server{
		Addr:           ":9790",
		Handler:        s,
		ReadTimeout:    3 * time.Second,
		WriteTimeout:   3 * time.Second,
		MaxHeaderBytes: 4096,
	}

	go s.Run()

	return
}
