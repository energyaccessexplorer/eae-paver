package main

import (
	"fmt"
	"github.com/energyaccessexplorer/gdal"
	"strconv"
)

type bounds struct {
	Left   float64
	Bottom float64
	Right  float64
	Top    float64
}

type dataset_info struct {
	Fields       []string `json:"fields"`
	FeatureCount int      `json:"featurecount"`
	Bounds       bounds   `json:"bounds"`
}

func info_bounds(in filename) gdal.Geometry {
	none := gdal.Create(gdal.GT_None)

	t := gdal.CreateSpatialReference("")
	t.FromEPSG(4326)

	src := gdal.OpenDataSource(in, 0)
	defer src.Destroy()

	layer := src.LayerByIndex(0)

	env, err := layer.Extent(true)
	if err != nil {
		logger.Println("Failed to get layer extent")
		return none
	}

	// text := fmt.Sprintf(
	// 	"MULTIPOINT (%.32f %.32f, %.32f %.32f)",
	// 	env.MinX(), env.MaxX(), env.MinY(), env.MaxY(),
	// )

	text := fmt.Sprintf(
		`POLYGON ((%.32f %.32f), (%.32f %.32f), (%.32f %.32f), (%.32f %.32f), (%.32f %.32f))`,
		env.MaxX(), env.MinY(), env.MaxX(), env.MaxY(), env.MinX(), env.MaxY(), env.MinX(), env.MinY(),
		env.MaxX(), env.MinY())

	// TODO: this looks like we are going around uselessly...
	//
	layer.ResetReading()
	geom := layer.NextFeature().Geometry()

	v, ok := geom.SpatialReference().AttrValue("AUTHORITY", 1)
	if !ok {
		logger.Println("Failed setting AUTHORITY to spatial reference")
		return none
	}

	s := gdal.CreateSpatialReference("")
	i, err := strconv.Atoi(v)
	if err != nil {
		logger.Println(err.Error())
		return none
	}

	s.FromEPSG(i)

	g, err := gdal.CreateFromWKT(text, s)
	if err != nil {
		logger.Println(err.Error())
		return none
	}

	return g
}
