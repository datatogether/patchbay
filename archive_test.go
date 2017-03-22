package main

import (
	"github.com/qri-io/archive"
	"testing"
	"time"
)

func TestArchive(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode, skipping.")
		return
	}
	// defer resetTestData(appDB, "urls", "links", "snapshots")
	var (
		links []*archive.Link
		res   *Url
		err   error
	)
	close := make(chan bool)

	done := func(err error) {
		defer func() {
			f, _ := res.File()
			if err := f.Delete(); err != nil {
				t.Error(err.Error())
				return
			}

			for _, l := range links {
				f, _ := l.Dst.File()
				if err := f.Delete(); err != nil {
					t.Error(err.Error())
				}
			}
			close <- true
		}()
		time.Sleep(time.Second)

		if err != nil {
			t.Error(err.Error())
			return
		}

		for _, l := range links {
			dst := l.Dst
			f, err := dst.File()
			if err != nil {
				t.Error(err.Error())
				return
			}

			if err := f.GetS3(); err != nil {
				t.Error(err.Error())
				return
			}
		}

		f, err := res.File()
		if err != nil {
			t.Error(err.Error())
			return
		}

		if err := f.GetS3(); err != nil {
			t.Error(err.Error())
			return
		}
	}

	res, links, err = ArchiveUrl(appDB, "http://docs.qri.io", done)
	if err != nil {
		t.Error(err.Error())
		return
	}
	<-close
}
