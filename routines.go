package main

import (
	"encoding/json"
	"fmt"
)

func routine_admin_boundaries(w reporter, s3 s3config, in filename, idfield string, resolution int) (string, error) {
	in = maybe_zip(in)
	in = maybe_shp(in)

	rprj, err := vectors_reproject(in, 3857, w)
	if err != nil {
		return "", err
	}
	w("%s <- reprojected", rprj)

	ids, err := raster_ids(rprj, idfield, resolution, w)
	if err != nil {
		return "", err
	}
	w("%s <- *raster ids", ids)

	stripped, err := vectors_strip(rprj, []string{idfield}, w)
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

	w("CLEAN UP")

	trash(rprj, stripped)

	if run_server {
		keeps := []filename{ids, rprjstripped}

		for _, f := range keeps {
			w("%s -> S3", f)
			s3put(f, s3)
			trash(f)
		}
	}

	w("DONE")

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

func routine_simplify(w reporter, s3 s3config, in filename, factor float32, idfield string, resolution int) (string, error) {
	in = maybe_zip(in)
	in = maybe_shp(in)

	simpl, err := vectors_simplify(in, factor, w)
	if err != nil {
		return "", err
	}
	w("%s <- *simplified", simpl)

	prj, err := vectors_reproject(simpl, 3857, w)
	if err != nil {
		return "", err
	}
	w("%s <- reprojected", prj)

	ids, err := raster_ids(prj, idfield, resolution, w)
	if err != nil {
		return "", err
	}
	w("%s <- *raster ids", ids)

	if run_server {
		keeps := []filename{simpl, ids}

		for _, f := range keeps {
			w("%s -> S3", f)
			s3put(f, s3)
			trash(f)
		}
	}

	w("DONE")

	jsonstr := fmt.Sprintf(`{ "vectors": "%s", "raster": "%s" }`, _uuid(simpl), _uuid(ids))

	return jsonstr, nil
}

func routine_clip_proximity(w reporter, s3 s3config, in filename, ref filename, fields []string, resolution int, simplify float32) (string, error) {
	in = maybe_zip(in)
	in = maybe_shp(in)

	stripped, err := vectors_strip(in, fields, w)
	if err != nil {
		return "", err
	}
	w("%s <- stripped", stripped)

	refprj, err := vectors_reproject(ref, 3857, w)
	if err != nil {
		return "", err
	}
	w("%s <- reprojected reference", refprj)

	zeros, err := raster_zeros(refprj, resolution, w)
	if err != nil {
		return "", err
	}
	w("%s <- zeros", zeros)

	simpl, err := vectors_simplify(ref, simplify, w)
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

	w("CLEAN UP")
	trash(in, ref, stripped, rstr, refprj, simpl)

	if run_server {
		keeps := []filename{clipped, prox}

		for _, f := range keeps {
			w("%s -> S3", f)
			s3put(f, s3)
			trash(f)
		}
	}

	w("DONE")

	jsonstr := fmt.Sprintf(`{ "vectors": "%s", "raster": "%s" }`, _uuid(clipped), _uuid(prox))

	return jsonstr, nil
}

func routine_csv_points(w reporter, s3 s3config, in filename, ref filename, lnglat [2]string, fields []string, resolution int) (string, error) {
	points, err := csv_points(in, lnglat, fields)
	if err != nil {
		return "", err
	}
	w("%s <- csv points", points)

	refprj, err := vectors_reproject(ref, 3857, w)
	if err != nil {
		return "", err
	}
	w("%s <- reprojected reference", refprj)

	zeros, err := raster_zeros(refprj, resolution, w)
	if err != nil {
		return "", err
	}
	w("%s <- zeros", zeros)

	clipped, err := vectors_clip(points, ref, w)
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

	w("CLEAN UP")
	trash(in, ref, points, rstr, refprj)

	if run_server {
		keeps := []filename{clipped, prox}

		for _, f := range keeps {
			w("%s -> S3", f)
			s3put(f, s3)
			trash(f)
		}
	}

	w("DONE")

	jsonstr := fmt.Sprintf(`{ "vectors": "%s", "raster": "%s" }`, _uuid(clipped), _uuid(prox))

	return jsonstr, nil
}

func routine_crop_raster(w reporter, s3 s3config, in filename, base filename, ref filename, conf string, resolution int) (string, error) {
	in = maybe_zip(in)
	in = maybe_shp(in)

	var c raster_config
	err := json.Unmarshal([]byte(conf), &c)
	if err != nil {
		return "", err
	}

	cropped, err := raster_crop(in, base, ref, c, resolution, w)
	if err != nil {
		return "", err
	}
	w("%s <- cropped", cropped)

	w("CLEAN UP")

	if run_server {
		keeps := []filename{cropped}

		for _, f := range keeps {
			w("%s -> S3", f)
			s3put(f, s3)
			trash(f)
		}
	}

	w("DONE")

	jsonstr := fmt.Sprintf(`{ "raster": "%s" }`, _uuid(cropped))

	return jsonstr, nil
}

func routine_subgeographies(w reporter, s3 s3config, in filename, id string) (string, error) {
	in = maybe_zip(in)
	in = maybe_shp(in)

	r, _ := vectors_features_split(in, id, w)

	w("CLEAN UP")

	if run_server {
		for i, f := range r {
			r[i] = _uuid(r[i])

			w("%s -> S3", f)
			s3put(f, s3)
			trash(f)
		}
	}

	w("DONE")

	jsonstr, _ := json.Marshal(r)

	return string(jsonstr), nil
}
