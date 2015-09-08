package api

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
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
	router := httprouter.New()
	router.GET("/1/packages", api.packages)
	router.GET("/1/packages/:name", api.describePackage)

	log.WithField("port", api.listen).Info("Starting listener")
	log.Fatal(http.ListenAndServe(api.listen, router))
}

func (api *Api) packages(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	packages, err := api.install.Packages()

	if err != nil {
		api.writeError(w, "Could not retrieve package list", err)
		return
	}

	if err = json.NewEncoder(w).Encode(packages); err != nil {
		api.writeError(w, "Could not retrieve package list", err)
	}
}

func (api *Api) describePackage(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	name := ps.ByName("name")
	pkg, err := api.install.Package(name)
	if err != nil {
		api.writeError(w, fmt.Sprintf("Could not find package %s", name), err)
		return
	}

	if err = json.NewEncoder(w).Encode(pkg); err != nil {
		api.writeError(w, fmt.Sprintf("Could not encode package %s", name), err)
	}
}

func (api *Api) writeError(w http.ResponseWriter, msg string, err error) {
	m := fmt.Sprintf("%s: %v", msg, err)
	log.Warn(m)
	fmt.Fprintln(w, m)
	w.WriteHeader(500)
}
