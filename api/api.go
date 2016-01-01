package api

import (
	"encoding/json"
	"fmt"
	"github.com/CiscoCloud/mantl-api/install"
	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"io"
	"io/ioutil"
	"net/http"
)

type Api struct {
	listen  string
	install *install.Install
}

func NewApi(listen string, install *install.Install) *Api {
	return &Api{listen, install}
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
