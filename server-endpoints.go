package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coder/websocket"
	"net/http"
	"strconv"
	"strings"
)

type reporter func(string, ...any) string

type routine func(reporter, routine_params) (string, error)

type routine_params struct {
	ok         bool
	s3         s3config
	dataset    string
	reference  string
	base       string
	resolution int
	simplify   float32
	lnglat     [2]string
	attr       string
	fields     []string
	config     string // json
}

func sw(r *http.Request, k *websocket.Conn) reporter {
	return func(s string, x ...any) string {
		return socket_write(k, fmt.Sprintf(s+"\n", x...), r)
	}
}

type server_routine struct {
	fn       routine
	required []string
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

	p, err := server_prepare(r, rtn.required)
	if !p.ok {
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

func server_prepare(r *http.Request, n []string) (p routine_params, err error) {
	f, err := form_parse(r)
	if err != nil {
		return
	}

	for i := range n {
		err = errors.New(fmt.Sprintf("Missing parameter: %s", i))
		return
	}

	s3, err := s3config_get(f["s3bucket"])
	if err != nil {
		return p, err
	}

	if s3.Key == "" {
		err = errors.New("No such bucket: " + f["s3bucket"])
		return
	}

	p.dataset, err = snatch(f["dataseturl"])
	if err != nil {
		err = errors.New(fmt.Sprintf("Dataset URL: %s", err.Error()))
		return
	}

	if ref, has := f["referenceurl"]; has {
		p.reference, err = snatch(ref)
		if err != nil {
			err = errors.New(fmt.Sprintf("Reference URL: %s", err.Error()))
			return
		}
	}

	if bas, has := f["baseurl"]; has {
		p.base, err = snatch(bas)
		if err != nil {
			err = errors.New(fmt.Sprintf("Base URL: %s", err.Error()))
			return
		}
	}

	if res, has := f["resolution"]; has {
		p.resolution, err = strconv.Atoi(res)
		if err != nil {
			err = errors.New(fmt.Sprintf("Could not parse resolution: %s", err.Error()))
			return
		}
	}

	if ll, has := f["lnglat"]; has {
		s := strings.Split(ll, ",")
		if len(s) != 2 {
			err = errors.New("lnglat: should have length 2.")
			return
		}

		copy(p.lnglat[:], s[:2])
	}

	if si, has := f["simplify"]; has {
		f64, err := strconv.ParseFloat(si, 32)
		if err != nil {
			return p, err
		}

		p.simplify = float32(f64)
	}

	if ff, has := f["fields"]; has {
		p.fields = strings.Split(ff, ",")
	}

	if cfg, has := f["config"]; has {
		p.config = cfg // json...
	}

	if atr, has := f["attr"]; has {
		p.attr = atr
	}

	if err != nil {
		return p, err
	}

	return p, nil
}
