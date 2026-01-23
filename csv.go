package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/energyaccessexplorer/gdal"
	"io"
	"os"
	"path"
	"strings"
	"time"
)

func csv(in filename, fields []string) (filename, error) {
	src := gdal.OpenDataSource(in, 0)
	layer := src.LayerByIndex(0).Name()

	cols := strings.Join(fields, ",")
	sql := fmt.Sprintf("SELECT %s FROM \"%s\"", cols, layer)

	fis := vectors_fields(in)

	for _, a := range fields {
		for _, fi := range fis {
			if fi == a {
				goto found
			}
		}

		return "", errors.New(fmt.Sprintf("Could not find field %s in %s\n", a, fis))

	found:
		continue
	}

	var g gdal.Geometry
	l := src.ExecuteSQL(sql, g, "")

	var f0 gdal.Feature
	f0 = l.Feature(0)

	indexes := make([]int, len(fields))
	for j, str := range fields {
		indexes[j] = f0.FieldIndex(str)
	}

	var payload []string

	payload = append(payload, strings.Join(fields, ","))

	var f *gdal.Feature
	for {
		f = l.NextFeature()
		if f == nil {
			break
		}

		var line []string
		var s string

		for i := range indexes {
			if i > -1 {
				switch f.FieldDefinition(i).Type() {
				case 0:
					s = fmt.Sprintf("%d", f.FieldAsInteger(i))

				case 2:
					s = fmt.Sprintf("%f", f.FieldAsFloat64(i))

				case 4:
					s = f.FieldAsString(i)

				// case 6, 7: are deprecated by GDAL

				case 8:
					s = base64.StdEncoding.EncodeToString(f.FieldAsBinary(i))

				case 10:
					{
						t, ok := f.FieldAsDateTime(i)
						if !ok {
							return "", errors.New("csv: Failed to parse Date")
						}
						s = fmt.Sprintf("%d-%d-%d", t.Year(), t.Month(), t.Day())
					}

				case 11:
					{
						t, ok := f.FieldAsDateTime(i)
						if !ok {
							return "", errors.New("csv: Failed to parse DateTime")
						}

						s = t.Format(time.RFC3339)
					}

				case 12:
					s = fmt.Sprintf("%d", f.FieldAsInteger64(i))

				default:
					return "", errors.New("I don't know this type!")
				}
			}

			line = append(line, s)
		}

		payload = append(payload, strings.Join(line, ","))
	}
	defer src.ReleaseResultSet(l)

	fname := _filename()

	file, _ := os.Create(fname)
	defer file.Close()

	_, err := io.WriteString(file, strings.Join(payload, "\n"))
	if err != nil {
		return "", err
	}
	file.Sync()

	return fname, nil
}

func csv_points(in filename, lnglat [2]string, sel []string) (filename, error) {
	out := _filename()

	if len(sel) == 0 {
		sel = []string{"null"}
	}

	os.Rename(in, in+".csv")
	in = in + ".csv"

	openopts := []string{
		fmt.Sprintf("X_POSSIBLE_NAMES=%s", lnglat[0]),
		fmt.Sprintf("Y_POSSIBLE_NAMES=%s", lnglat[1]),
		"KEEP_GEOM_COLUMNS=YES",
	}

	src, err := gdal.OpenEx(in, gdal.OFReadOnly, []string{"CSV"}, openopts, nil)
	if err != nil {
		return "", err
	}
	defer src.Close()

	opts := []string{
		"-f", "GeoJSON",
		"-sql", fmt.Sprintf(
			"SELECT %s FROM \"%s\"",
			strings.Join(sel, ","),
			strings.Replace(path.Base(in), path.Ext(in), "", -1),
		),
	}

	release := capture()
	dest, err := gdal.VectorTranslate(out, []gdal.Dataset{src}, opts)

	result := release()
	if err != nil {
		return "", errors.New(result)
	}
	defer dest.Close()

	return out, nil
}

func csv_strip(in filename, sel []string) (filename, error) {
	out := _filename()

	if len(sel) == 0 {
		sel = []string{"null"}
	}

	src, err := gdal.Open(in, 0)
	if err != nil {
		println("Exiting...")
		return "", err
	}
	defer src.Close()

	opts := []string{
		"-f", "CSV",
		"-sql", fmt.Sprintf(
			"SELECT %s FROM \"%s\"",
			strings.Join(sel, ","),
			strings.Replace(path.Base(in), path.Ext(in), "", -1),
		),
	}

	release := capture()
	dest, err := gdal.Translate(out, src, opts)

	result := release()
	if err != nil {
		return "", errors.New(result)
	}
	defer dest.Close()

	return out, nil
}
