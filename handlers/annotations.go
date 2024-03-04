package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rramiachraf/dumb/data"
	"github.com/rramiachraf/dumb/views"
	"github.com/sirupsen/logrus"
)

func Annotations(l *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		if a, err := getCache[data.Annotation]("annotation:" + id); err == nil {
			encoder := json.NewEncoder(w)

			w.Header().Set("content-type", "application/json")
			if err = encoder.Encode(&a); err != nil {
				l.Errorln(err)
			}

			return
		}

		url := fmt.Sprintf("https://genius.com/api/referents/%s?text_format=html", id)
		resp, err := sendRequest(url)

		if err != nil {
			l.Errorln(err)
			w.WriteHeader(http.StatusInternalServerError)
			views.ErrorPage(500, "cannot reach genius servers").Render(context.Background(), w)
			return
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			w.WriteHeader(http.StatusNotFound)
			views.ErrorPage(404, "page not found").Render(context.Background(), w)
			return
		}

		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(resp.Body)
		if err != nil {
			l.Errorln("Error paring genius api response", err)
			w.WriteHeader(http.StatusInternalServerError)
			views.ErrorPage(500, "something went wrong").Render(context.Background(), w)
			return
		}

		var data data.AnnotationsResponse
		err = json.Unmarshal(buf.Bytes(), &data)
		if err != nil {
			l.Errorf("could not unmarshal json: %s\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			views.ErrorPage(500, "something went wrong").Render(context.Background(), w)
			return
		}

		body := data.Response.Referent.Annotations[0].Body
		body.HTML = cleanBody(body.HTML)

		w.Header().Set("content-type", "application/json")
		encoder := json.NewEncoder(w)

		if err = encoder.Encode(&body); err != nil {
			l.Errorln("Error sending response: ", err)
			return
		}

		if err = setCache("annotation:"+id, body); err != nil {
			l.Errorln(err)
		}
	}
}

func cleanBody(body string) string {
	var withCleanedImageLinks = strings.ReplaceAll(body, "https://images.rapgenius.com/", "/images/")
	withCleanedImageLinks = strings.ReplaceAll(body, "https://images.genius.com/", "/images/")

	var re = regexp.MustCompile(`https?:\/\/[a-z]*.?genius.com`)
	var withCleanedLinks = re.ReplaceAllString(withCleanedImageLinks, "")

	return withCleanedLinks
}