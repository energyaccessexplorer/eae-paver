package main

import (
	"encoding/json"
	"fmt"
)

func routine_admin_boundaries(w reporter, p routine_params) (string, error) {
	in := p.Dataset
	in = maybe_zip(in)
	in = maybe_shp(in)

	rprj, err := vectors_reproject(in, 3857, w)
	if err != nil {
		return "", err
	}
	w("%s <- reprojected", rprj)

	ids, err := raster_ids(rprj, p.Attr, p.Resolution, w)
	if err != nil {
		return "", err
	}
	w("%s <- *raster ids", ids)

	stripped, err := vectors_strip(rprj, []string{p.Attr}, w)
	if err != nil {
		return "", err
	}
	w("%s <- stripped", stripped)

	rprjstripped, err := vectors_reproject(in, 4326, w)
	if err != nil {
		return "", err
	}
	w("%s <- *stripped reprojected", rprjstripped)

	info := vectors_info(rprjstripped)

	cleanup(
		[]filename{ids, rprjstripped},
		[]filename{rprj, stripped},
		w, p.S3,
	)

	jinfo, err := json.Marshal(info)
	if err != nil {
		return "", err
	}

	jsonstr := fmt.Sprintf(
		`{ "vectors": "%s", "raster": "%s", "info": %s }`,
		_uuid(rprjstripped),
		_uuid(ids),
		jinfo,
	)

	return jsonstr, nil
}

func routine_simplify(w reporter, p routine_params) (string, error) {
	in := p.Dataset
	in = maybe_zip(in)
	in = maybe_shp(in)

	simpl, err := vectors_simplify(in, p.Simplify, w)
	if err != nil {
		return "", err
	}
	w("%s <- *simplified", simpl)

	prj, err := vectors_reproject(simpl, 3857, w)
	if err != nil {
		return "", err
	}
	w("%s <- reprojected", prj)

	ids, err := raster_ids(prj, p.Attr, p.Resolution, w)
	if err != nil {
		return "", err
	}
	w("%s <- *raster ids", ids)

	cleanup(
		[]filename{simpl, ids},
		[]filename{},
		w, p.S3,
	)

	jsonstr := fmt.Sprintf(`{ "vectors": "%s", "raster": "%s" }`, _uuid(simpl), _uuid(ids))

	return jsonstr, nil
}

func routine_clip_proximity(w reporter, p routine_params) (string, error) {
	in := p.Dataset
	in = maybe_zip(in)
	in = maybe_shp(in)

	stripped, err := vectors_strip(in, p.Fields, w)
	if err != nil {
		return "", err
	}
	w("%s <- stripped", stripped)

	refprj, err := vectors_reproject(p.Reference, 3857, w)
	if err != nil {
		return "", err
	}
	w("%s <- reprojected reference", refprj)

	zeros, err := raster_zeros(refprj, p.Resolution, w)
	if err != nil {
		return "", err
	}
	w("%s <- zeros", zeros)

	simpl, err := vectors_simplify(p.Reference, p.Simplify, w)
	if err != nil {
		return "", err
	}
	w("%s <- simplified reference", simpl)

	clipped, err := vectors_clip(in, simpl, w)
	if err != nil {
		return "", err
	}
	w("%s <- *clipped", clipped)

	rstr, err := raster_geometry_ones(clipped, zeros, w)
	if err != nil {
		return "", err
	}
	w("%s <- rasterised <- zeros", rstr) // overwrites zeros

	prox, err := raster_proximity(rstr, w)
	if err != nil {
		return "", err
	}
	w("%s <- *proximity", prox)

	cleanup(
		[]filename{clipped, prox},
		[]filename{in, p.Reference, stripped, rstr, refprj, simpl},
		w, p.S3,
	)

	jsonstr := fmt.Sprintf(`{ "vectors": "%s", "raster": "%s" }`, _uuid(clipped), _uuid(prox))

	return jsonstr, nil
}

func routine_csv_points(w reporter, p routine_params) (string, error) {
	points, err := csv_points(p.Dataset, p.LngLat, p.Fields)
	if err != nil {
		return "", err
	}
	w("%s <- csv points", points)

	refprj, err := vectors_reproject(p.Reference, 3857, w)
	if err != nil {
		return "", err
	}
	w("%s <- reprojected reference", refprj)

	zeros, err := raster_zeros(refprj, p.Resolution, w)
	if err != nil {
		return "", err
	}
	w("%s <- zeros", zeros)

	clipped, err := vectors_clip(points, p.Reference, w)
	if err != nil {
		return "", err
	}
	w("%s <- *clipped", clipped)

	rstr, err := raster_geometry_ones(clipped, zeros, w)
	if err != nil {
		return "", err
	}
	w("%s <- rasterised <- zeros", rstr) // overwrites zeros

	prox, err := raster_proximity(rstr, w)
	if err != nil {
		return "", err
	}
	w("%s <- *proximity", prox)

	cleanup(
		[]filename{clipped, prox},
		[]filename{p.Dataset, p.Reference, points, rstr, refprj},
		w, p.S3,
	)

	jsonstr := fmt.Sprintf(`{ "vectors": "%s", "raster": "%s" }`, _uuid(clipped), _uuid(prox))

	return jsonstr, nil
}

func routine_crop_raster(w reporter, p routine_params) (string, error) {
	in := p.Dataset
	in = maybe_zip(in)
	in = maybe_shp(in)

	cropped, err := raster_crop(in, p.Base, p.Reference, p.Config, p.Resolution, w)
	if err != nil {
		return "", err
	}
	w("%s <- cropped", cropped)

	cleanup(
		[]filename{cropped},
		[]filename{},
		w, p.S3,
	)

	jsonstr := fmt.Sprintf(`{ "raster": "%s" }`, _uuid(cropped))

	return jsonstr, nil
}

func routine_subgeographies(w reporter, p routine_params) (string, error) {
	in := p.Dataset
	in = maybe_zip(in)
	in = maybe_shp(in)

	r, _ := vectors_features_split(in, p.Attr, w)

	for i, f := range r {
		r[i] = _uuid(r[i])
		cleanup([]filename{f}, []filename{}, w, p.S3)
	}

	jsonstr, _ := json.Marshal(r)

	return string(jsonstr), nil
}

func routine_vectors_extra_attributes(w reporter, p routine_params) (string, error) {
	in := p.Dataset
	in = maybe_zip(in)
	in = maybe_shp(in)

	r, _ := vectors_features_extra_attrs(in, w)

	for i, f := range r {
		r[i] = _uuid(r[i])
		cleanup([]filename{f}, []filename{}, w, p.S3)
	}

	jsonstr, _ := json.Marshal(r)

	return string(jsonstr), nil
}

func routine_csv_raster(w reporter, p routine_params) (string, error) {
	in := p.Dataset

	points, err := csv_points(in, p.LngLat, []string{p.Attr})
	if err != nil {
		return "", err
	}
	w("%s <- csv points", points)

	refprj, err := vectors_reproject(p.Reference, 3857, w)
	if err != nil {
		return "", err
	}
	w("%s <- reprojected reference", refprj)

	zeros, err := raster_zeros(refprj, p.Resolution, w)
	if err != nil {
		return "", err
	}
	w("%s <- zeros", zeros)

	clipped, err := vectors_clip(points, p.Reference, w)
	if err != nil {
		return "", err
	}
	w("%s <- clipped", clipped)

	rstr, err := raster_geometry_attr(clipped, zeros, p.Attr, w)
	if err != nil {
		return "", err
	}
	w("%s <- rasterised", rstr)

	cleanup(
		[]filename{rstr},
		[]filename{in, p.Reference, points, clipped, refprj},
		w, p.S3,
	)

	jsonstr := fmt.Sprintf(`{ "raster": "%s" }`, _uuid(rstr))

	return jsonstr, nil
}

func cleanup(keeps []filename, deletes []filename, w reporter, s3 s3config) {
	w("CLEAN UP")

	trash(deletes...)

	for _, f := range keeps {
		w("%s -> S3", f)
		s3put(f, s3)
		trash(f)
	}

	w("DONE")
}
