package main

import (
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"gitlab.com/noop.nu/srv"
	"io"
	"net/http"
	"os"
)

type H map[string]srv.Handler

type reporter func(string, ...any) string

type routine func(reporter, routine_params) (string, error)

type routine_params struct {
	S3         s3config      `json:"s3"`
	Dataset    string        `json:"dataseturl"`
	Reference  string        `json:"referenceurl"`
	Base       string        `json:"baseurl"`
	Resolution int           `json:"resolution"`
	Simplify   float32       `json:"simplify"`
	LngLat     [2]string     `json:"lnglat"`
	Attr       string        `json:"attr"`
	Fields     []string      `json:"fields"`
	Config     raster_config `json:"config"`
	Dissolve   bool          `json:"dissolve"`
}

type server_routine struct {
	fn       routine
	required []string
}

func serve() {
	server_setup()

	srv.Run(
		socket,
		[]srv.Route{
			{"/check", nil, H{"GET": _check}},
			{"/socket", nil, H{"GET": _socket}},
			{"/routines", []string{"*"}, H{"POST": _routines}},
			{"/s3-presigned", []string{"*"}, H{"GET": s3presigned_handler}},
		},
		pubkeyfile,
	)
}

func server_setup() {
	_, err := os.Stat(tmpdir)
	if os.IsNotExist(err) {
		logger.Println("Specified temporary directory does not exist. Creating...")
		os.Mkdir(tmpdir, 0755)
	}

	t, err := os.Open(tmpdir)
	if err != nil {
		logger.Fatal("Specified temporary directory (still) does not exist!")
	}
	t.Close()

	fmt.Printf("Temporary directory is '%s'\n", tmpdir)
	fmt.Printf("Public key is: %s\n", pubkeyfile)
}

func sw(r *http.Request, k *websocket.Conn) reporter {
	return func(s string, x ...any) string {
		return socket_write(k, fmt.Sprintf(s+"\n", x...), r)
	}
}

var server_routines = map[string]server_routine{
	"admin-boundaries": {
		routine_admin_boundaries,
		[]string{
			"s3bucket",
			"dataseturl",
			"attr",
			"resolution",
		},
	},
	"simplify": {
		routine_simplify,
		[]string{
			"s3bucket",
			"dataseturl",
			"simplify",
			"resolution",
			"attr",
		}},
	"clip-proximity": {
		routine_clip_proximity,
		[]string{
			"s3bucket",
			"dataseturl",
			"referenceurl",
			"fields",
			"resolution",
			"simplify",
		},
	},
	"csv-points": {
		routine_csv_points,
		[]string{
			"s3bucket",
			"dataseturl",
			"referenceurl",
			"fields",
			"lnglat",
			"resolution",
		}},
	"csv-raster": {
		routine_csv_raster,
		[]string{
			"s3bucket",
			"dataseturl",
			"referenceurl",
			"attr",
			"lnglat",
			"resolution",
		}},
	"crop-raster": {
		routine_crop_raster,
		[]string{
			"s3bucket",
			"dataseturl",
			"baseurl",
			"referenceurl",
			"config",
			"resolution",
		},
	},
	"subgeographies": {
		routine_subgeographies,
		[]string{
			"s3bucket",
			"dataseturl",
			"attr",
		}},
}

func _routines(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("routine")
	if q == "" {
		http.Error(w, "Routine (q)uery parameter is not optional", 405)
		return
	}

	var rtn server_routine
	var has bool
	if rtn, has = server_routines[q]; !has {
		http.Error(w, "Unknown routine: "+q, 405)
		return
	}

	sid := r.URL.Query().Get("socket_id")
	s := socket_table[sid]

	var jb map[string]interface{}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err = json.Unmarshal(body, &jb); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	for _, x := range rtn.required {
		if _, ok := jb[x]; !ok {
			http.Error(w, fmt.Sprintf("Incomplete payload. Missing '%s'", x), 400)
			return
		}
	}

	p := routine_params{}

	s3bucket := jb["s3bucket"].(string)
	p.S3, _ = s3config_get(s3bucket)
	if err != nil || p.S3.Key == "" {
		http.Error(w, fmt.Sprintf("No such bucket: '%s'", s3bucket), 400)
		return
	}

	defer r.Body.Close()
	if err := json.Unmarshal(body, &p); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if jsonstr, err := rtn.fn(sw(r, s), p); err == nil {
		fmt.Fprintf(w, jsonstr)
	} else {
		j, _ := json.Marshal(map[string]string{"error": err.Error()})
		http.Error(w, string(j), 400)
	}

	defer socket_destroy(sid, s, "routine finished")
}

func _socket(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	socket_create(id, w, r)
}

func _check(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "TJA!")
}
