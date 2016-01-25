package api

import (
	"encoding/json"
	"fmt"
	"github.com/CiscoCloud/mantl-api/install"
	"github.com/CiscoCloud/mantl-api/mesos"
	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

type Api struct {
	listen  string
	install *install.Install
	mesos   *mesos.Mesos
}

func NewApi(listen string, install *install.Install, mesos *mesos.Mesos) *Api {
	return &Api{listen, install, mesos}
}

func logHandler(handler http.Handler) http.Handler {
	hfunc := func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("%s %s", r.Method, r.RequestURI)
		handler.ServeHTTP(w, r)
	}
	return http.HandlerFunc(hfunc)
}

func (api *Api) Start() {
	router := httprouter.New()
	router.GET("/health", api.health)

	router.GET("/1/packages", api.packages)
	router.GET("/1/packages/:name", api.describePackage)

	router.GET("/1/frameworks", api.frameworks)
	router.DELETE("/1/frameworks/:id", api.shutdownFramework)

	router.POST("/1/install", api.installPackage)
	router.DELETE("/1/install", api.uninstallPackage)

	log.WithField("port", api.listen).Info("Starting listener")
	log.Fatal(http.ListenAndServe(api.listen, logHandler(router)))
}

func (api *Api) health(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	w.WriteHeader(200)
	fmt.Fprintf(w, "OK")
}

func (api *Api) packages(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	packages, err := api.install.Packages()

	if err != nil {
		writeError(w, "Could not retrieve package list", 500, err)
		return
	}

	if err = json.NewEncoder(w).Encode(packages); err != nil {
		writeError(w, "Could not retrieve package list", 500, err)
	}
}

func (api *Api) describePackage(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	name := ps.ByName("name")
	pkg, err := api.install.Package(name)
	if err != nil {
		writeError(w, fmt.Sprintf("Package %s not found.", name), 404, err)
		return
	}

	if err = json.NewEncoder(w).Encode(pkg); err != nil {
		writeError(w, fmt.Sprintf("Could not encode package %s", name), 500, err)
	}
}

func (api *Api) installPackage(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	req.Header.Add("Accept", "application/json")

	pkgRequest, err := parsePackageRequest(req.Body)

	if err != nil || pkgRequest == nil {
		writeError(w, "Could not parse package request", 400, err)
		return
	}

	marathonResponse, err := api.install.InstallPackage(pkgRequest)
	if err != nil {
		writeError(w, fmt.Sprintf("Could not install %s package", pkgRequest.Name), 500, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, marathonResponse)
}

func (api *Api) uninstallPackage(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	req.Header.Add("Accept", "application/json")

	pkgRequest, err := parsePackageRequest(req.Body)

	if err != nil || pkgRequest == nil {
		writeError(w, "Could not parse package request", 400, err)
		return
	}

	apps, err := api.install.FindInstalled(pkgRequest)

	if err != nil {
		writeError(w, "Could not retrieve installed packages", 500, err)
		return
	}

	if len(apps) == 0 {
		w.WriteHeader(404)
		if pkgRequest.AppID != "" {
			fmt.Fprintf(w, "Package %s (%s) not found.\n", pkgRequest.Name, pkgRequest.AppID)
		} else {
			fmt.Fprintf(w, "Package %s not found.\n", pkgRequest.Name)
		}
		return
	} else if len(apps) > 1 {
		w.WriteHeader(409)
		fmt.Fprintf(w, "There is more than 1 instance of the %s package running. Please include the application id in the request.\n", pkgRequest.Name)
		return
	}

	err = api.install.UninstallPackage(apps[0])
	if err != nil {
		writeError(w, fmt.Sprintf("Could not uninstall %s package", pkgRequest.Name), 500, err)
		return
	}

	w.WriteHeader(204)
}

type frameworkResponse struct {
	Name             string    `json:"name"`
	ID               string    `json:"id"`
	Active           bool      `json:"active"`
	Hostname         string    `json:"hostname"`
	User             string    `json:"user"`
	RegisteredTime   time.Time `json:"registeredTime"`
	ReregisteredTime time.Time `json:"reregisteredTime"`
	ActiveTasks      int       `json:"activeTasks"`
}

func (api *Api) frameworks(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	var frameworks []*mesos.Framework
	var err error
	if _, ok := req.URL.Query()["completed"]; ok {
		frameworks, err = api.mesos.CompletedFrameworks()
	} else {
		frameworks, err = api.mesos.Frameworks()
	}

	if err != nil {
		writeError(w, "Could not retrieve frameworks", 500, err)
		return
	}

	response := make([]*frameworkResponse, len(frameworks))
	for i, fw := range frameworks {
		var rrtime time.Time
		rtime := time.Unix(int64(fw.RegisteredTime), 0)
		if fw.ReregisteredTime != 0 {
			rrtime = time.Unix(int64(fw.ReregisteredTime), 0)
		}
		response[i] = &frameworkResponse{fw.Name, fw.ID, fw.Active, fw.Hostname, fw.User, rtime, rrtime, len(fw.Tasks)}
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		writeError(w, "Could not encode frameworks %s", 500, err)
	}
}

func (api *Api) shutdownFramework(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	w.WriteHeader(202)
	frameworkId := ps.ByName("id")

	err := api.mesos.Shutdown(frameworkId)
	if err != nil {
		writeError(w, fmt.Sprintf("Could not shutdown framework %s", frameworkId), 500, err)
		return
	}
}

func writeError(w http.ResponseWriter, msg string, status int, err error) {
	w.WriteHeader(status)
	m := fmt.Sprintf("%s: %v", msg, err)
	log.Error(m)
	fmt.Fprintln(w, m)
}

func parsePackageRequest(r io.Reader) (*install.PackageRequest, error) {
	body, err := ioutil.ReadAll(r)
	if err != nil {
		log.Errorf("Unable to read request body: %v", err)
		return nil, err
	}

	pkgRequest, err := install.NewPackageRequest(body)
	if err != nil {
		return nil, err
	}

	return pkgRequest, nil
}
