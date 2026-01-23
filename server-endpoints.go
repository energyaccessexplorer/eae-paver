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

func sw(r *http.Request, k *websocket.Conn) reporter {
	return func(s string, x ...any) string {
		return socket_write(k, fmt.Sprintf(s+"\n", x...), r)
	}
}

type server_routine func(*http.Request, *websocket.Conn) (string, error)

var server_routines = map[string]server_routine{
	"admin-boundaries": server_admin_boundaries,
	"clip-proximity":   server_clip_proximity,
	"crop-raster":      server_crop_raster,
	"csv-points":       server_csv_points,
	"csv-raster":       server_csv_raster,
	"simplify":         server_simplify,
	"subgeographies":   server_subgeographies,
}

func _routines(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("routine")
	if q == "" {
		http.Error(w, "Routine (q)uery parameter is not optional", 405)
		return
	}

	rtn := server_routines[q]
	if rtn == nil {
		http.Error(w, "Unknown routine: "+q, 405)
		return
	}

	sid := r.URL.Query().Get("socket_id")
	s := socket_table[sid]

	if jsonstr, err := rtn(r, s); err == nil {
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

func server_prepare(f *formdata, r *http.Request) (ok bool, s3 s3config, datasetfile string, referencefile string, resolution int, longlat [2]string, err error) {
	if err = form_parse(f, r); err != nil {
		return
	}

	s3, err = s3config_get(string((*f)["s3bucket"]))
	if err != nil {
		return
	}

	if s3.Key == "" {
		err = errors.New("No such bucket: " + string((*f)["s3bucket"]))
		return
	}

	datasetfile, err = snatch(string((*f)["dataseturl"]))
	if err != nil {
		err = errors.New(fmt.Sprintf("Dataset URL: %s", err.Error()))
		return
	}

	if ref, has := (*f)["referenceurl"]; has {
		referencefile, err = snatch(string(ref))
		if err != nil {
			err = errors.New(fmt.Sprintf("Reference URL: %s", err.Error()))
			return
		}
	}

	if res, has := (*f)["resolution"]; has {
		resolution, err = strconv.Atoi(string(res))
		if err != nil {
			err = errors.New(fmt.Sprintf("Could not parse resolution: %s", err.Error()))
			return
		}
	}

	if ll, has := (*f)["lnglat"]; has {
		s := strings.Split(string(ll), ",")
		if len(s) != 2 {
			err = errors.New("lnglat: should have length 2.")
			return
		}

		copy(longlat[:], s[:2])
	}

	return true, s3, datasetfile, referencefile, resolution, longlat, nil
}

func server_admin_boundaries(r *http.Request, s *websocket.Conn) (string, error) {
	f := formdata{
		"s3bucket":   nil,
		"dataseturl": nil,
		"attr":       nil,
		"resolution": nil,
	}

	ok, s3, datasetfile, _, resolution, _, err := server_prepare(&f, r)
	if !ok {
		return "", err
	}

	jsonstr, err := routine_admin_boundaries(
		sw(r, s),
		s3,
		datasetfile,
		string(f["attr"]),
		resolution,
	)
	if err != nil {
		return "", err
	}

	return jsonstr, nil
}

func server_simplify(r *http.Request, s *websocket.Conn) (string, error) {
	f := formdata{
		"s3bucket":   nil,
		"dataseturl": nil,
		"simplify":   nil,
		"resolution": nil,
		"attr":       nil,
	}

	ok, s3, datasetfile, _, resolution, _, err := server_prepare(&f, r)
	if !ok {
		return "", err
	}

	factor, err := strconv.ParseFloat(string(f["simplify"]), 32)
	if err != nil {
		return "", err
	}

	jsonstr, err := routine_simplify(
		sw(r, s),
		s3,
		datasetfile,
		float32(factor),
		string(f["attr"]),
		resolution,
	)

	if err != nil {
		return "", err
	}

	return jsonstr, nil
}

func server_clip_proximity(r *http.Request, s *websocket.Conn) (string, error) {
	f := formdata{
		"s3bucket":     nil,
		"dataseturl":   nil,
		"referenceurl": nil,
		"fields":       nil,
		"resolution":   nil,
		"simplify":     nil,
	}

	ok, s3, datasetfile, referencefile, resolution, _, err := server_prepare(&f, r)
	if !ok {
		return "", err
	}

	_simp, _ := strconv.ParseFloat(string(f["simplify"]), 32)
	simp := float32(_simp)

	jsonstr, err := routine_clip_proximity(
		sw(r, s),
		s3,
		datasetfile,
		referencefile,
		strings.Split(string(f["fields"]), ","),
		resolution,
		simp,
	)

	if err != nil {
		return "", err
	}

	return jsonstr, nil
}

func server_csv_points(r *http.Request, s *websocket.Conn) (string, error) {
	f := formdata{
		"s3bucket":     nil,
		"dataseturl":   nil,
		"referenceurl": nil,
		"fields":       nil,
		"lnglat":       nil,
		"resolution":   nil,
	}

	ok, s3, datasetfile, referencefile, resolution, lnglat, err := server_prepare(&f, r)
	if !ok {
		return "", err
	}

	jsonstr, err := routine_csv_points(
		sw(r, s),
		s3,
		datasetfile,
		referencefile,
		lnglat,
		strings.Split(string(f["fields"]), ","),
		resolution,
	)

	if err != nil {
		return "", err
	}

	return jsonstr, nil
}

func server_crop_raster(r *http.Request, s *websocket.Conn) (string, error) {
	f := formdata{
		"s3bucket":     nil,
		"dataseturl":   nil,
		"baseurl":      nil,
		"referenceurl": nil,
		"config":       nil,
		"resolution":   nil,
	}

	ok, s3, datasetfile, referencefile, resolution, _, err := server_prepare(&f, r)
	if !ok {
		return "", err
	}

	basefile, err := snatch(string(f["baseurl"]))
	if err != nil {
		return "", err
	}

	configjson := string(f["config"])

	jsonstr, err := routine_crop_raster(
		sw(r, s),
		s3,
		datasetfile,
		basefile,
		referencefile,
		configjson,
		resolution,
	)

	if err != nil {
		return "", err
	}

	return jsonstr, nil
}

func server_subgeographies(r *http.Request, s *websocket.Conn) (string, error) {
	f := formdata{
		"s3bucket":   nil,
		"dataseturl": nil,
		"attr":       nil,
	}

	ok, s3, datasetfile, _, _, _, err := server_prepare(&f, r)
	if !ok {
		return "", err
	}

	jsonstr, err := routine_subgeographies(
		sw(r, s),
		s3,
		datasetfile,
		string(f["attr"]),
	)

	return jsonstr, nil
}

func server_csv_raster(r *http.Request, s *websocket.Conn) (string, error) {
	f := formdata{
		"s3bucket":     nil,
		"dataseturl":   nil,
		"referenceurl": nil,
		"attr":         nil,
		"lnglat":       nil,
		"resolution":   nil,
	}

	ok, s3, datasetfile, referencefile, resolution, lnglat, err := server_prepare(&f, r)
	if !ok {
		return "", err
	}

	jsonstr, err := routine_csv_raster(
		sw(r, s),
		s3,
		datasetfile,
		referencefile,
		lnglat,
		string(f["attr"]),
		resolution,
	)

	if err != nil {
		return "", err
	}

	return jsonstr, nil
}
