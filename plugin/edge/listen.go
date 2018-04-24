package edge

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// Start listening for table updates on port 8053.
func (e *Edge) startListeningForTableUpdates() {
	e.server = &http.Server{Addr: ":8053"}
	http.HandleFunc("/", e.parseTableUpdate)
	go func() {
		if err := e.server.ListenAndServe(); err != nil {
			log.Errorf("ListenAndServe error: %s\n", err)
		}
	}()
}

// Parse incoming requests from edge sites.
func (e *Edge) parseTableUpdate(w http.ResponseWriter, r *http.Request) {
	jsn, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorln("Error while reading table update:", err)
	}
	update := TableUpdate{}
	if err = json.Unmarshal(jsn, &update); err != nil {
		log.Errorln("Error while unmarshalling JSON into table update struct:", err)
	}
	e.table.Update(update.Meta.IP, update.Meta.Lon, update.Meta.Lat, update.Services)
}

// Stop listening for updates.
func (e *Edge) stopListeningForTableUpdates() {
	e.server.Shutdown(nil)
}
