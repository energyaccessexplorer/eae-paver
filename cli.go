package main

import (
	"fmt"
	"strings"
)

var (
	idfield string

	selectfields arrayFlag

	command       string
	inputfile     string
	basefile      string
	referencefile string
	targetfile    string
)

type arrayFlag []string

func (i *arrayFlag) String() string {
	return strings.Join(*i, ",")
}

func (i *arrayFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func cli() {
	var s3 s3config

	p := func(s string, x ...any) {
		println(fmt.Sprintf(s, x...))
	}

	if inputfile == "" {
		panic("No -i (input file) given")
	}

	var err error
	var out string

	switch command {
	case "bounds":
		{
			fmt.Println("bounds:", info_bounds(inputfile).ToJSON())
		}

	case "vectors-info":
		{
			fmt.Println("info:", vectors_info(inputfile))
		}

	case "subgeographies":
		{
			out, err = routine_subgeographies(p, s3, inputfile, idfield)
		}

	case "shp":
		{
			out = maybe_shp(inputfile)
		}

	case "zeros":
		{
			out, err = raster_zeros(inputfile, 1000, p)
		}

	case "strip":
		{
			if len(selectfields) == 0 {
				panic("No -s (select fields) given.")
			}

			out, err = vectors_strip(inputfile, selectfields, p)
		}

	case "rasterise":
		{
			if targetfile == "" {
				panic("No -t (targetfile) given:")
			}

			out, err = raster_geometry(inputfile, targetfile, p)
		}

	case "proximity":
		{
			if targetfile == "" {
				panic("No -t (targetfile) given:")
			}

			r, _ := raster_geometry(inputfile, targetfile, p)

			out, err = raster_proximity(r, p)
		}

	case "ids-raster":
		{
			out, err = raster_ids(inputfile, idfield, 1000, p)
		}

	case "clip":
		{
			if targetfile == "" {
				panic("No -t (targetfile) given:")
			}

			out, err = vectors_clip(inputfile, targetfile, p)
		}

	case "csv":
		{
			if len(selectfields) == 0 {
				panic("No -s (select fields) given.")
			}

			out, err = csv(inputfile, selectfields)
		}

	case "simplify":
		{
			if targetfile == "" {
				panic("No -t (targetfile) given:")
			}

			out, err = vectors_simplify(inputfile, 0.01, p)
		}

	case "csv-points":
		{
			if len(selectfields) == 0 {
				println("No -s (select fields) given.")
			}

			out, err = csv_points(inputfile, [2]string{"Longitude", "Latitude"}, selectfields)
		}

	case "admin-boundaries":
		{
			out, err = routine_admin_boundaries(nil, s3, inputfile, idfield, 1000)
		}

	case "routine-clip-proximity":
		{
			if referencefile == "" {
				panic("No -r (referencefile) given:")
			}

			out, err = routine_clip_proximity(p, s3, inputfile, referencefile, []string{idfield}, 1000, 0.001)
		}

	case "routine-crop-raster":
		{
			if referencefile == "" {
				panic("No -r (referencefile) given:")
			}

			if basefile == "" {
				panic("No -r (referencefile) given:")
			}

			out, err = routine_crop_raster(nil, s3, inputfile, basefile, referencefile, "{\"nodata\": -1, \"numbertype\": \"Int16\", \"resample\": \"average\"}", 1000)
		}

	case "s3put":
		{
			s3put(inputfile, s3)
		}

	default:
		{
			println("No (valid) -c command given:", command)
		}
	}

	if err != nil {
		println(err.Error())
	} else {
		println(out)
	}
}
