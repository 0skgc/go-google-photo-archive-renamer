package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/adrium/goheif"
	"github.com/rwcarlsen/goexif/exif"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const timeStampFormat = "20060102-150405"

var metaJsonPathSplit = regexp.MustCompile(`([^.]+)(\.\w+)(\(\d+\))?(\.\w+)`)
var pathSplit = regexp.MustCompile(`([^()]*)(\(\d+\))?(\.\w+)`)
var jsonExtRegex = regexp.MustCompile(`^(?i)\.json$`)

var timestampMap = map[string]int{}
var renamedExtMap = map[string]int{}
var copyMetaJsonCount int
var unRenamedExtMap = map[string]int{}

// GooglePhotoMeta json struct
type GooglePhotoMeta struct {
	PhotoTakenTime PhotoTakenTime `json:"photoTakenTime"`
}

// PhotoTakenTime json struct
type PhotoTakenTime struct {
	TimeStamp json.Number `json:"timestamp"`
}

type meta struct {
	jsonPath       string
	photoTakenTime time.Time
}

func main() {
	var isDryrun bool
	var targetDir string
	flag.StringVar(&targetDir, "target", "", "target dir path")
	flag.BoolVar(&isDryrun, "dryrun", false, "true is logging only")
	flag.Parse()
	targetDir = strings.TrimSpace(targetDir)

	if targetDir == "" {
		log.Fatalf("target is empty")
		return
	}
	if isDryrun {
		log.Print("dryrun mode")
	} else {
		log.Print("rename mode")
	}

	if err := normalizeMetaJsonFiles(targetDir, isDryrun); err != nil {
		log.Fatalf("error %+v\n", err)
	}

	if err := renameForDirFiles(targetDir, isDryrun); err != nil {
		log.Fatalf("error %+v\n", err)
	}

	log.Printf("# Copy meta json: %v", copyMetaJsonCount)
	log.Print("# Renamed Ext")
	for k, v := range renamedExtMap {
		log.Printf("  - %v: %v", k, v)
	}
	log.Print("# UnRenamed Ext")
	for k, v := range unRenamedExtMap {
		log.Printf("  - %v: %v", k, v)
	}
}

func normalizeMetaJsonFiles(targetDir string, isDryrun bool) error {
	files, err := ioutil.ReadDir(targetDir)
	if err != nil {
		return err
	}

	const HEICExt = ".HEIC"
	const MP4Ext = ".MP4"
	for _, file := range files {
		path := filepath.Join(targetDir, file.Name())
		if !isJSON(path) {
			continue
		}
		submatch := metaJsonPathSplit.FindStringSubmatch(path)
		if submatch == nil {
			continue
		}
		baseFilePath := submatch[1]
		baseFileExt := submatch[2]
		baseFileNumber := submatch[3]
		jsonFileExt := submatch[4]

		// iPhoneのLivePhotosの場合、画像ファイルと対になるMP4ファイルのJSONがない場合があるので、jsonを複製する
		// 画像ファイルはHEIC限定とする
		if !strings.EqualFold(baseFileExt, HEICExt) || baseFileExt == "" {
			continue
		}
		mp4Path := baseFilePath + baseFileNumber + MP4Ext
		if _, err := os.Stat(mp4Path); os.IsNotExist(err) {
			continue
		}
		mp4JsonPath := baseFilePath + MP4Ext + baseFileNumber + jsonFileExt
		if _, err := os.Stat(mp4JsonPath); err == nil {
			continue
		}
		if isDryrun {
			log.Printf("Copy(dryrun) to json meta file : %v", mp4JsonPath)
		} else {
			if err := copyMetaJsonFile(path, mp4JsonPath); err != nil {
				return err
			}
			log.Printf("Copy to json meta file : %v", mp4JsonPath)
		}
		copyMetaJsonCount = copyMetaJsonCount + 1
	}
	return nil
}

func copyMetaJsonFile(srcJsonPath string, dstJsonPath string) error {
	src, err := os.Open(srcJsonPath)
	if err != nil {
		panic(err)
	}
	defer src.Close()
	dst, err := os.Create(dstJsonPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}

func renameForDirFiles(targetDir string, isDryrun bool) error {
	files, err := ioutil.ReadDir(targetDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		path := filepath.Join(targetDir, file.Name())
		if file.IsDir() || isJSON(path) {
			continue
		}

		fileExt := strings.ToUpper(filepath.Ext(path))
		meta, err := getMetaFromJSON(path)
		if err != nil {
			meta, err = getMetaFromJpeg(path)
		}
		if err != nil {
			meta, err = getMetaFromHeic(path)
		}
		if err != nil {
			extCount := unRenamedExtMap[fileExt]
			unRenamedExtMap[fileExt] = extCount + 1
			log.Printf("err: %v", err)
			continue
		}
		_formatTimestamp := meta.photoTakenTime.Format(timeStampFormat)

		tsCount := timestampMap[_formatTimestamp]
		timestampMap[_formatTimestamp] = tsCount + 1

		formatTimestamp := fmt.Sprintf("%v-%v", _formatTimestamp, tsCount)
		baseName := getFileNameWithoutExt(path)

		newNameFromPath := strings.Replace(path, baseName, formatTimestamp, 1)

		if err := rename(isDryrun, path, newNameFromPath, meta.photoTakenTime); err != nil {
			return err
		}

		if meta.jsonPath != "" {
			jsonBaseName := getFileNameWithoutExt(meta.jsonPath)
			newNameFromJSON := strings.Replace(meta.jsonPath, jsonBaseName, formatTimestamp+filepath.Ext(newNameFromPath), 1)

			if err := rename(isDryrun, meta.jsonPath, newNameFromJSON, meta.photoTakenTime); err != nil {
				return err
			}
		}

		extCount := renamedExtMap[fileExt]
		renamedExtMap[fileExt] = extCount + 1
	}
	return nil
}

func getMetaFromJSON(path string) (*meta, error) {
	jsonPath, err := getJSONPath(path)
	if err != nil {
		return nil, err
	}

	raw, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		return nil, err
	}
	var dto GooglePhotoMeta
	if err := json.Unmarshal(raw, &dto); err != nil {
		return nil, err
	}
	unixTime, err := dto.PhotoTakenTime.TimeStamp.Int64()
	if err != nil {
		return nil, err
	}
	photoTakenTime := time.Unix(unixTime, 0)
	return &meta{
		jsonPath,
		photoTakenTime,
	}, nil
}

func getMetaFromJpeg(path string) (*meta, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decode, err := exif.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("%v parse: %v", path, err)
	}
	dateTime, err := decode.DateTime()
	if err != nil {
		return nil, err
	}
	return &meta{
		"",
		dateTime,
	}, nil
}

func getMetaFromHeic(path string) (*meta, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bin, err := goheif.ExtractExif(file)
	if err != nil {
		return nil, fmt.Errorf("%v open: %v", path, err)
	}
	decode, err := exif.Decode(bytes.NewReader(bin))
	if err != nil {
		return nil, fmt.Errorf("%v parse: %v", path, err)
	}
	dateTime, err := decode.DateTime()
	if err != nil {
		return nil, err
	}
	return &meta{
		"",
		dateTime,
	}, nil
}

func rename(isDryrun bool, oldPath string, newPath string, time time.Time) error {
	if oldPath == newPath {
		return nil
	}
	if isDryrun {
		log.Print("# Rename(dryrun)")
	} else {
		log.Print("# Rename")
	}
	log.Printf("  - From: %v ", oldPath)
	log.Printf("  -   To: %v ", newPath)
	if isDryrun {
		return nil
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}

	if err := os.Chtimes(newPath, time, time); err != nil {
		return err
	}
	return nil
}

func isJSON(path string) bool {
	return jsonExtRegex.MatchString(filepath.Ext(path))
}

func getJSONPath(path string) (string, error) {
	if isJSON(path) {
		return path, nil
	}

	baseName := filepath.Base(path)
	submatch := pathSplit.FindStringSubmatch(baseName)
	var err error
	for _, maxLen := range [2]int{46, 255} {
		jsonName := (submatch[1] + submatch[3])
		if len(jsonName) > maxLen {
			jsonName = jsonName[:maxLen]
		}
		jsonName = jsonName + submatch[2] + ".json"

		jsonPath := strings.Replace(path, baseName, jsonName, 1)
		if _, err = os.Stat(jsonPath); err == nil {
			return jsonPath, nil
		}
	}

	return "", err
}

func getFileNameWithoutExt(path string) string {
	return filepath.Base(getPathWithoutExt(path))
}

func getPathWithoutExt(path string) string {
	return path[:len(path)-len(filepath.Ext(path))]
}
