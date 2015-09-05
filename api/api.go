package api

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/ryane/mantl-api/install"
	"net/http"
)

type Api struct {
	listen  string
	install *install.Install
}

func NewApi(listen string, install *install.Install) *Api {
	return &Api{listen, install}
}

func (api *Api) Start() {
	http.HandleFunc("/1/packages", api.Packages)
	log.WithField("port", api.listen).Info("Starting listener")
	log.Fatal(http.ListenAndServe(api.listen, nil))
}

func (api *Api) Packages(w http.ResponseWriter, req *http.Request) {
	packages, err := api.install.Packages()

	if err != nil {
		api.writeError(w, "Could not retrieve package list", err)
		return
	}

	if err = json.NewEncoder(w).Encode(packages); err != nil {
		api.writeError(w, "Could not retrieve package list", err)
	}
}

func (api *Api) writeError(w http.ResponseWriter, msg string, err error) {
	m := fmt.Sprintf("%s: %v", msg, err)
	log.Warn(m)
	fmt.Fprintln(w, m)
	w.WriteHeader(500)
}
