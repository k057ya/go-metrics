package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/k057ya/go-metrics/internal/repository"
)

func ListAllMetrics(resp http.ResponseWriter, req *http.Request, storage repository.MetricsStorage) {

	resp.Header().Set("Content-Type", "text/html; charset=utf-8")
	resp.WriteHeader(http.StatusOK)

	var html strings.Builder
	html.WriteString("<h1>Metrics List</h1><ul>\n")

	if len(storage.List()) == 0 {
		html.WriteString("<li><em>Storage is empty.</em></li>")
	}

	for _, metric := range storage.List() {

		html.WriteString(fmt.Sprintf(
			`<li><a href="%s"><b>%s</b></a>: <pre>%s</pre></li>`,
			metric.URL(),
			metric.ID,
			metric.StringValue(),
		))

	}
	html.WriteString("</ul>")

	resp.Write([]byte(html.String()))
}
