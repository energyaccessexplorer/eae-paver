package main

import (
	"errors"
	"fmt"
	"github.com/energyaccessexplorer/gdal"
	"strconv"
)

type raster_config struct {
	Numbertype string `json:"numbertype"`
	Nodata     int    `json:"nodata"`
	Resample   string `json:"resample"`
}

func raster_ids(in filename, gid string, resolution int, w reporter) (filename, error) {
	src, err := gdal.OpenEx(in, gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		return "", err
	}
	defer src.Close()

	out := _filename()

	res := strconv.Itoa(resolution)

	opts := []string{
		"-a", gid,
		"-a_srs", "EPSG:3857",
		"-a_nodata", "-1",
		"-tr", res, res,
		"-of", "GTiff",
		"-ot", "Int16",
		"-co", "COMPRESS=DEFLATE",
		"-co", "PREDICTOR=1",
		"-co", "ZLEVEL=9",
	}

	release := capture()
	dest, err := gdal.Rasterize(out, src, opts)

	result := release()
	if err != nil {
		return "", errors.New(result)
	}
	dest.Close()

	return out, err
}

func raster_geometry(in filename, zs filename, w reporter) (filename, error) {
	src, err := gdal.OpenEx(in, gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		return "", err
	}

	zeros, err := gdal.OpenEx(zs, gdal.OFUpdate, nil, nil, nil)
	if err != nil {
		return "", err
	}

	f := gdal.OpenDataSource(in, 0)
	defer f.Destroy()

	layer := f.LayerByIndex(0)

	opts := []string{
		"-l", layer.Name(),
		"-burn", "1",
	}

	_, ok := layer.FeatureCount(false)
	if !ok {
		return "", errors.New("Could not get feature count")
	}

	release := capture()
	err = gdal.RasterizeOverwrite(zeros, src, opts)

	result := release()
	if err != nil {
		return "", errors.New(result)
	}

	// rasterized.Close()
	zeros.Close()
	src.Close()

	return zs, err
}

func raster_proximity(in filename, w reporter) (filename, error) {
	src, err := gdal.OpenEx(in, gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		return "", err
	}
	defer src.Close()

	drv, err := gdal.GetDriverByName("GTiff")
	if err != nil {
		return "", err
	}

	out := _filename()

	opts := []string{
		"-ot", "Int16",
		"-nodata", "-1",
		"DISTUNITS=PIXEL",
		"VALUES=1",
		"USE_INPUT_NODATA=YES",
		fmt.Sprintf("MAXDIST=%d", 1024),
		"-co", "COMPRESS=DEFLATE",
		"-co", "PREDICTOR=1",
		"-co", "ZLEVEL=9",
	}

	ds := drv.CreateCopy(out, src, 0, []string{}, gdal.DummyProgress, nil)

	release := capture()
	err = src.
		RasterBand(1).
		ComputeProximity(ds.RasterBand(1), opts, gdal.DummyProgress, nil)

	result := release()
	if err != nil {
		return "", errors.New(result)
	}
	ds.Close()

	return out, err
}

func raster_zeros(in filename, resolution int, w reporter) (filename, error) {
	src, err := gdal.OpenEx(in, gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		return "", err
	}
	defer src.Close()

	out := _filename()

	res := strconv.Itoa(resolution)

	opts := []string{
		"-burn", "0",
		"-a_nodata", "-1",
		"-a_srs", "EPSG:3857",
		"-tr", res, res,
		"-of", "GTiff",
		"-ot", "Int16",
	}

	release := capture()
	dest, err := gdal.Rasterize(out, src, opts)

	result := release()
	if err != nil {
		return "", errors.New(result)
	}
	defer dest.Close()

	return out, err
}

func raster_crop(in filename, base filename, ref filename, c raster_config, res int, w reporter) (filename, error) {
	w("RASTER CROP")

	r, err := gdal.OpenEx(base, gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		return "", err
	}
	defer r.Close()

	src, err := gdal.OpenEx(in, gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		return "", err
	}
	defer src.Close()

	out := _filename()

	f := gdal.OpenDataSource(ref, 0)
	defer f.Destroy()

	layer := f.LayerByIndex(0).Name()
	w(" cropping to first layer: %s", layer)

	x := r.RasterXSize()
	y := r.RasterYSize()

	w(" raster size: (%d,%d)", x, y)
	w(" numbertype: %s", c.Numbertype)
	w(" nodata: %d", c.Nodata)
	w(" resampling method: %s", c.Resample)
	w(" resolution: %d", res)

	r_out := _filename()

	r_opts := []string{
		"-of", "GTiff",
		"-t_srs", "EPSG:3857",
		"-tr", strconv.Itoa(res), strconv.Itoa(res),
		"-r", c.Resample,
	}

	release := capture()
	r_src, err := gdal.Warp(r_out, nil, []gdal.Dataset{src}, r_opts)

	result := release()
	if err != nil {
		err = errors.New(result)
		w(err.Error())
		return "", err
	}
	defer r_src.Close()

	c_opts := []string{
		"-cutline", ref,
		"-crop_to_cutline",
		"-cl", layer,
		"-of", "GTiff",
		"-ts", strconv.Itoa(x), strconv.Itoa(y),
		"-t_srs", "EPSG:3857",
		"-ot", c.Numbertype,
		"-dstnodata", strconv.Itoa(c.Nodata),
		"-co", "COMPRESS=DEFLATE",
		"-co", "PREDICTOR=1",
		"-co", "ZLEVEL=9",
	}

	release1 := capture()
	dest, err := gdal.Warp(out, nil, []gdal.Dataset{r_src}, c_opts)

	result1 := release1()
	if err != nil {
		err = errors.New(result1)
		w(err.Error())
		return "", err
	}
	defer dest.Close()

	return out, nil
}
