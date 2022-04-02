package input

import (
	"bufio"
	"context"
	"fmt"
	"github.com/bytepowered/flow/v3/extends/events"
	"github.com/bytepowered/flow/v3/extends/match"
	flow "github.com/bytepowered/flow/v3/pkg"
	"github.com/bytepowered/runv"
	"github.com/spf13/viper"
	"os"
	"time"
)

const filestreamTag = "filestream"

var (
	_ flow.Input    = new(FileStreamReader)
	_ runv.Initable = new(FileStreamReader)
)

type FileStreamReader struct {
	*BaseInput
	filepath string
	drops    []match.Matcher
}

func NewFileStreamReader() *FileStreamReader {
	return &FileStreamReader{
		BaseInput: NewBaseInput(filestreamTag),
	}
}

func (r *FileStreamReader) OnInit() error {
	with := func(pre string) {
		r.filepath = viper.GetString(pre + "path")
		for _, pattern := range viper.GetStringSlice(pre + "drops") {
			r.drops = append(r.drops, match.MustCompile(pattern))
		}
	}
	with(r.Tag() + ".")
	return nil
}

func (r *FileStreamReader) OnReceived(ctx context.Context, queue chan<- flow.Event) {
	f, err := os.OpenFile(r.filepath, os.O_RDONLY, os.FileMode(0))
	if err != nil {
		flow.Log().Panicf("FILESTREAM: open file error: %s", err.Error())
	}
	fi, err := f.Stat()
	if err != nil {
		flow.Log().Panicf("FILESTREAM: failed to stat file %s: %s", r.filepath, err)
	}
	err = checkRegularFile(fi)
	if err != nil {
		flow.Log().Panic(err)
	}
	flow.Log().Infof("FILESTREAM: scan file: %s", r.filepath)
	scan := bufio.NewScanner(f)
	records, failed := 0, 0
	for scan.Scan() {
		select {
		case <-ctx.Done():
			break
		case <-r.Done():
			break
		default:
			// next
		}
		records++
		err := scan.Err()
		if err != nil {
			failed++
			flow.Log().Errorf("FILESTREAM: scan file error: %s", err.Error())
			continue
		}
		line := scan.Text()
		// Drop lines
		if matchAny(r.drops, line) {
			flow.Log().Debugf("FILESTREAM: Drop line: %s", line)
			continue
		}
		queue <- events.BufferedEventOf(flow.Header{
			Time: time.Now().UnixNano(),
			Tag:  r.Tag(),
			Kind: 0,
		}, []byte(line))
	}
	flow.Log().Infof("FILESTREAM: scan file, done, lines: %d/%d", records, failed)
}

func matchAny(matchers []match.Matcher, text string) bool {
	for _, m := range matchers {
		if m.MatchString(text) {
			return true
		}
	}
	return false
}

func checkRegularFile(fi os.FileInfo) error {
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("tried to open non regular file: %q %s", fi.Mode(), fi.Name())
	}
	return nil
}
