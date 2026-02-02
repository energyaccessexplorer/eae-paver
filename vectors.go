package main

/*
#ifndef GO_GDAL_H_
#define GO_GDAL_H_

#include <gdal.h>
#include <gdal_alg.h>
#include <gdal_utils.h>
#include <gdalwarper.h>
#include <cpl_conv.h>
#include <ogr_srs_api.h>

// transform GDALProgressFunc to go func
GDALProgressFunc goGDALProgressFuncProxyB();

#endif // GO_GDAL_H_
#include "gdal_version.h"
*/
import "C"

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"github.com/energyaccessexplorer/gdal"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func vectors_strip(in filename, fields []string, w reporter) (filename, error) {
	out := _filename()

	opts := []string{
		"-f", "GeoJSON",
	}

	if len(fields) > 1 {
		fields = append(fields, "-select", strings.Join(fields, ","))
	}

	release := capture()
	src, err := gdal.OpenEx(in, gdal.OFReadOnly, nil, nil, nil)

	result := release()
	if err != nil {
		w(err.Error())
		w(result)
		return "", err
	}
	defer src.Close()

	release1 := capture()
	dest, err := gdal.VectorTranslate(out, []gdal.Dataset{src}, opts)

	result1 := release1()
	if err != nil {
		w(err.Error())
		w(result1)
		return "", err
	}
	defer dest.Close()

	return out, err
}

func vectors_reproject(in filename, epsg int, w reporter) (filename, error) {
	out := _filename()

	opts := []string{
		"-f", "GeoJSON",
		"-t_srs", "EPSG:" + strconv.Itoa(epsg),
	}

	src, err := gdal.OpenEx(in, gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		return "", err
	}
	defer src.Close()

	release := capture()
	dst, err := gdal.VectorTranslate(out, []gdal.Dataset{src}, opts)
	defer dst.Close()

	result := release()
	if err != nil {
		w(err.Error())
		w(result)
		return "", err
	}

	return out, nil
}

func vectors_clip(in filename, container filename, w reporter) (filename, error) {
	w("VECTORS CLIP")

	out := _filename()

	f := gdal.OpenDataSource(in, 0)
	g := gdal.OpenDataSource(container, 0)
	defer f.Destroy()
	defer g.Destroy()

	src := f.LayerByIndex(0)
	tar := g.LayerByIndex(0)

	tt, _ := src.FeatureCount(true)
	w("	source feature count: %d", tt)

	ct, _ := tar.FeatureCount(true)
	w("	container feature count: %d", ct)
	if ct > 1 {
		return "", errors.New(fmt.Sprintf(
			"	The container file has %d features. It should have 1: the contour of the geography. \n"+
				"This is a configuration error on the geography.", ct))
	}

	drv := gdal.OGRDriverByName("GeoJSON")
	ds, _ := drv.Create(out, []string{})

	s := gdal.CreateSpatialReference("")
	s.FromEPSG(4326)

	res := ds.CreateLayer("Layer0", s, src.Type(), []string{})

	w("	clipping...")
	release := capture()
	err := src.Clip(tar, res, []string{})

	result := release()
	if err != nil {
		w(err.Error())
		w(result)
		return "", err
	}

	ds.Destroy()

	h := gdal.OpenDataSource(out, 0)
	defer h.Destroy()

	l := h.LayerByIndex(0)

	rt, _ := l.FeatureCount(true)
	w("	result feature count: %d", rt)

	return out, err
}

func vectors_simplify(in filename, s float32, w reporter) (filename, error) {
	out := _filename()

	opts := []string{
		"-f", "GeoJSON",
		"-simplify", fmt.Sprintf("%f", s),
	}

	release0 := capture()
	src, err := gdal.OpenEx(in, gdal.OFReadOnly, nil, nil, nil)

	result0 := release0()
	if err != nil {
		w(err.Error())
		w(result0)
		return "", err
	}
	defer src.Close()

	release := capture()
	dst, err := gdal.VectorTranslate(out, []gdal.Dataset{src}, opts)
	defer dst.Close()

	result := release()
	if err != nil {
		w(err.Error())
		w(result)
		return "", err
	}

	return out, nil
}

func vectors_features_split(in filename, id string, w reporter) (map[int64]string, error) {
	w("VECTORS FEATURES")

	f := gdal.OpenDataSource(in, 0)
	src := f.LayerByIndex(0)
	defer f.Destroy()

	drv := gdal.OGRDriverByName("GeoJSON")
	s := gdal.CreateSpatialReference("")
	s.FromEPSG(4326)

	results := make(map[int64]string)

	for i := 0; i < vectors_feature_count(in); i++ {
		f := src.Feature(int64(i))
		x := f.FieldIndex(id)
		y := f.FieldAsInteger64(x)

		if x < 0 {
			w("YEAH, NAH...")
			return nil, errors.New("-1")
		}

		out := _filename()
		ds, _ := drv.Create(out, []string{})

		layer := ds.CreateLayer("Layer0", s, src.Type(), []string{})
		layer.Create(f)

		ds.Destroy()

		results[y] = out

		w("%d: %s", y, out)
	}

	return results, nil
}

func vectors_features_extra_attrs(in filename, w reporter) (map[int64]string, error) {
	w("VECTORS FEATURES")

	f := gdal.OpenDataSource(in, 0)
	defer f.Destroy()
	src := f.LayerByIndex(0)

	results := make(map[int64]string)

	t := gdal.CreateSpatialReference("")
	t.FromEPSG(3857)

	for i := 0; i < vectors_feature_count(in); i++ {
		f := src.Feature(int64(i))

		g := f.Geometry()
		g.TransformTo(t)

		switch f.Geometry().Type() {

		case gdal.GT_Polygon:
			println("AREA: ", int(g.Area()))

		case gdal.GT_LineString:
			println("LENGTH: ", int(g.Length()))

		case gdal.GT_MultiLineString:
			println("LENGTH: ", int(g.Length()))
		}
	}

	return results, nil
}

func vectors_feature_count(in filename) int {
	f := gdal.OpenDataSource(in, 0)
	defer f.Destroy()

	src := f.LayerByIndex(0)
	cs, _ := src.FeatureCount(true)

	return cs
}

func vectors_info(in filename) dataset_info {
	b := info_bounds(in)
	if b.Type() == gdal.GT_None {
		return dataset_info{}
	}

	e := b.Envelope()

	return dataset_info{
		vectors_fields(in),
		vectors_feature_count(in),
		bounds{e.MinX(), e.MinY(), e.MaxX(), e.MaxY()},
	}
}

func vectors_fields(in filename) []string {
	f := gdal.OpenDataSource(in, 0)
	defer f.Destroy()

	fdef := f.
		LayerByIndex(0).
		Definition()

	c := fdef.FieldCount()
	a := make([]string, c)

	for i := 0; i < c; i++ {
		a[i] = fdef.FieldDefinition(i).Name()
	}

	return a
}

func maybe_zip(in filename) string {
	r, err := zip.OpenReader(in)
	if err != nil {
		return in
	}
	defer r.Close()

	out := _filename()

	if !strings.HasSuffix(in, ".zip") {
		os.Rename(in, in+".zip")
		in = in + ".zip"
	}

	cmd := exec.Command("ogr2ogr", "-f", "GeoJSON", out, "/vsizip/"+in)
	cmd.Output()

	return out
}

func maybe_shp(in filename) string {
	f, _ := os.Open(in)
	defer f.Close()

	h := []byte{0, 0, 39, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	b := make([]byte, 20)

	f.Read(b)

	if !bytes.Equal(b, h) {
		return in
	}

	out := _filename()

	if !strings.HasSuffix(in, ".shp") {
		os.Rename(in, in+".shp")
		in = in + ".shp"
	}

	os.Setenv("SHAPE_RESTORE_SHX", "YES")
	cmd := exec.Command("ogr2ogr", "-f", "GeoJSON", out, in)
	cmd.Output()

	return out
}
