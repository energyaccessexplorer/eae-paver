package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"nhooyr.io/websocket"
	"strconv"
	"strings"
)

type reporter func(string, ...any)

func sw(r *http.Request, k *websocket.Conn) reporter {
	return func(s string, x ...any) {
		socket_write(k, fmt.Sprintf(s+"\n", x...), r)
	}
}

type server_routine func(*http.Request, *websocket.Conn) (string, error)

var server_routines = map[string]server_routine{
	"admin-boundaries": server_admin_boundaries,
	"clip-proximity":   server_clip_proximity,
	"crop-raster":      server_crop_raster,
	"csv-points":       server_csv_points,
	"simplify":         server_simplify,
	"subgeographies":   server_subgeographies,
}

func _routines(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("routine")
	if q == "" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		io.WriteString(w, "routine query parameter is not optional")
		return
	}

	if rtn := server_routines[q]; rtn == nil {
		w.WriteHeader(http.StatusMethodNotAllowed)
		io.WriteString(w, "don't know what you mean by: "+q)
	} else {
		sid := r.URL.Query().Get("socket_id")
		s := socket_table[sid]

		if jsonstr, err := rtn(r, s); err == nil {
			io.WriteString(w, jsonstr)
		} else {
			j, _ := json.Marshal(map[string]string{"error": err.Error()})
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, string(j))
		}

		defer socket_destroy(sid, s, "routine finished")
	}
}

func _socket(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	socket_create(id, w, r)
}

func _check(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "TJA!")
}

func server_admin_boundaries(r *http.Request, s *websocket.Conn) (string, error) {
	f := formdata{
		"dataseturl": nil,
		"field":      nil,
		"resolution": nil,
	}

	if err := form_parse(&f, r); err != nil {
		return "", err
	}
	inputfile, err := snatch(string(f["dataseturl"]))
	if err != nil {
		return "", err
	}

	res, _ := strconv.Atoi(string(f["resolution"]))

	jsonstr, err := routine_admin_boundaries(
		sw(r, s),
		inputfile,
		string(f["field"]),
		res,
	)

	if err != nil {
		return "", err
	}

	return jsonstr, nil
}

func server_simplify(r *http.Request, s *websocket.Conn) (string, error) {
	f := formdata{
		"dataseturl": nil,
		"simplify":   nil,
		"resolution": nil,
		"field":      nil,
	}

	if err := form_parse(&f, r); err != nil {
		return "", err
	}
	if err != nil {
		return "", err
	}

	inputfile, err := snatch(string(f["dataseturl"]))
	if err != nil {
		return "", err
	}

	factor, err := strconv.ParseFloat(string(f["simplify"]), 32)
	if err != nil {
		return "", err
	}

	resolution, err := strconv.Atoi(string(f["resolution"]))
	if err != nil {
		return "", err
	}

	jsonstr, err := routine_simplify(
		sw(r, s),
		inputfile,
		float32(factor),
		string(f["field"]),
		int(resolution),
	)

	if err != nil {
		return "", err
	}

	return jsonstr, nil
}

func server_clip_proximity(r *http.Request, s *websocket.Conn) (string, error) {
	f := formdata{
		"dataseturl":   nil,
		"referenceurl": nil,
		"fields":       nil,
		"resolution":   nil,
		"simplify":     nil,
	}

	if err := form_parse(&f, r); err != nil {
		return "", err
	}
	if err != nil {
		return "", err
	}

	inputfile, err := snatch(string(f["dataseturl"]))
	if err != nil {
		return "", err
	}

	referencefile, err := snatch(string(f["referenceurl"]))
	if err != nil {
		return "", err
	}

	res, _ := strconv.Atoi(string(f["resolution"]))

	_simp, _ := strconv.ParseFloat(string(f["simplify"]), 32)
	simp := float32(_simp)

	jsonstr, err := routine_clip_proximity(
		sw(r, s),
		inputfile,
		referencefile,
		strings.Split(string(f["fields"]), ","),
		res,
		simp,
	)

	if err != nil {
		return "", err
	}

	return jsonstr, nil
}

func server_csv_points(r *http.Request, s *websocket.Conn) (string, error) {
	f := formdata{
		"dataseturl":   nil,
		"referenceurl": nil,
		"fields":       nil,
		"lnglat":       nil,
		"resolution":   nil,
	}

	if err := form_parse(&f, r); err != nil {
		return "", err
	}
	if err != nil {
		return "", err
	}

	inputfile, err := snatch(string(f["dataseturl"]))
	if err != nil {
		return "", err
	}

	referencefile, err := snatch(string(f["referenceurl"]))
	if err != nil {
		return "", err
	}

	res, _ := strconv.Atoi(string(f["resolution"]))
	ll := strings.Split(string(f["lnglat"]), ",")

	if len(ll) != 2 {
		return "", errors.New("Argument Error: lnglat length should be 2")
	}

	jsonstr, err := routine_csv_points(
		sw(r, s),
		inputfile,
		referencefile,
		[2]string{ll[0], ll[1]},
		strings.Split(string(f["fields"]), ","),
		res,
	)

	if err != nil {
		return "", err
	}

	return jsonstr, nil
}

func server_crop_raster(r *http.Request, s *websocket.Conn) (string, error) {
	f := formdata{
		"dataseturl":   nil,
		"baseurl":      nil,
		"referenceurl": nil,
		"config":       nil,
		"resolution":   nil,
	}

	if err := form_parse(&f, r); err != nil {
		return "", err
	}
	if err != nil {
		return "", err
	}

	inputfile, err := snatch(string(f["dataseturl"]))
	if err != nil {
		return "", err
	}

	basefile, err := snatch(string(f["baseurl"]))
	if err != nil {
		return "", err
	}

	referencefile, err := snatch(string(f["referenceurl"]))
	if err != nil {
		return "", err
	}

	res, _ := strconv.Atoi(string(f["resolution"]))

	configjson := string(f["config"])

	jsonstr, err := routine_crop_raster(
		sw(r, s),
		inputfile,
		basefile,
		referencefile,
		configjson,
		res,
	)

	if err != nil {
		return "", err
	}

	return jsonstr, nil
}

func server_subgeographies(r *http.Request, s *websocket.Conn) (string, error) {
	f := formdata{
		"dataseturl": nil,
		"idcolumn":   nil,
	}

	if err := form_parse(&f, r); err != nil {
		return "", err
	}
	if err != nil {
		return "", err
	}

	dataseturl, err := snatch(string(f["dataseturl"]))
	if err != nil {
		return "", err
	}

	idcolumn := string(f["idcolumn"])

	jsonstr, err := routine_subgeographies(
		sw(r, s),
		dataseturl,
		idcolumn,
	)

	return jsonstr, nil
}
