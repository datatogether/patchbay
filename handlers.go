package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"text/template"

	"github.com/julienschmidt/httprouter"
)

// templates is a collection of views for rendering with the renderTemplate function
// see homeHandler for an example
var templates = template.Must(template.ParseFiles(
	"views/index.html",
	"views/accessDenied.html",
	"views/notFound.html",
	"views/urls.html",
))

func ArchiveUrlHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	done := func(err error) {}
	res, _, err := ArchiveUrl(appDB, r.FormValue("url"), done)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, fmt.Sprintf("archive url '%s' error: %s", r.FormValue("url"), err.Error()))
		return
	}

	data, err := json.MarshalIndent(res.Url, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, fmt.Sprintf("error marshalling url json: %s", err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func reqUrl(r *http.Request) (*url.URL, error) {
	return url.Parse(r.FormValue("url"))
}

func UrlMetadataHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	reqUrl, err := reqUrl(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, fmt.Sprintf("'%s' is not a valid url", r.FormValue("url")))
		return
	}

	u := &Url{Url: reqUrl.String()}
	if err := u.Read(appDB); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, fmt.Sprintf("read url '%s' err: %s", reqUrl.String(), err.Error()))
		return
	}

	meta, err := u.Metadata(appDB)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, fmt.Sprintf("read url '%s' err: %s", reqUrl.String(), err.Error()))
		return
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, fmt.Sprintf("encode json error: %s", err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)
}

func SaveUrlContextHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// uc := &UrlContext{}
	// if err := json.NewDecoder(r.Body).Decode(uc); err != nil {
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	io.WriteString(w, fmt.Sprintf("json formatting error: %s", err.Error()))
	// 	return
	// }
	// r.Body.Close()

	// if err := uc.Save(appDB); err != nil {
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	io.WriteString(w, fmt.Sprintf("error saving context: %s", err.Error()))
	// 	return
	// }

	// w.WriteHeader(200)
	// w.Header().Add("Content-Type", "application/json")
	// if err := json.NewEncoder(w).Encode(uc); err != nil {
	// 	logger.Println(err.Error())
	// }
}

// func DeleteUrlContextHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
// 	uc := &UrlContext{}
// 	if err := json.NewDecoder(r.Body).Decode(uc); err != nil {
// 		w.WriteHeader(http.StatusBadRequest)
// 		io.WriteString(w, fmt.Sprintf("json formatting error: %s", err.Error()))
// 		return
// 	}
// 	r.Body.Close()

// 	if err := uc.Delete(appDB); err != nil {
// 		w.WriteHeader(http.StatusBadRequest)
// 		io.WriteString(w, fmt.Sprintf("error saving context: %s", err.Error()))
// 		return
// 	}

// 	w.WriteHeader(200)
// 	io.WriteString(w, "url deleted")
// }

func UrlSetMetadataHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	reqUrl, err := reqUrl(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, fmt.Sprintf("'%s' is not a valid url", r.FormValue("url")))
		return
	}

	u := &Url{Url: reqUrl.String()}
	if err := u.Read(appDB); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, fmt.Sprintf("read url '%s' err: %s", reqUrl.String(), err.Error()))
		return
	}

	defer r.Body.Close()
	meta := map[string]interface{}{}
	if err := json.NewDecoder(r.Body).Decode(&meta); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, fmt.Sprintf("json parse err: %s", err.Error()))
		return
	}
	u.Meta = meta

	if err := u.Update(appDB); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, fmt.Sprintf("save url error: %s", err.Error()))
		return
	}

	m, err := u.Metadata(appDB)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, fmt.Sprintf("url metadata error: %s", err.Error()))
		return
	}
	data, err := json.Marshal(m)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, fmt.Sprintf("encode json error: %s", err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)
}

// WebappHandler renders the home page
func WebappHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	renderTemplate(w, "index.html")
}

func UrlsViewHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	urls, err := ListUrls(appDB, 200, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = templates.ExecuteTemplate(w, "urls.html", urls)
	if err != nil {
		logger.Println(err.Error())
		return
	}
}

func HandleWebsocketUpgrade(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	serveWs(room, w, r)
}

// renderTemplate renders a template with the values of cfg.TemplateData
func renderTemplate(w http.ResponseWriter, tmpl string) {
	err := templates.ExecuteTemplate(w, tmpl, cfg.TemplateData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func CertbotHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	io.WriteString(w, cfg.CertbotResponse)
}
