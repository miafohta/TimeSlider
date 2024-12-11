//================================================
// This test requires mp4 src file in path below
// tmp/input/sitepath/{}/{id}/240p.mp4
//
//================================================
package timeslider

import (
	"mercury/core/jsonconf"
	"mercury/core/log"
	"fmt"
	"os"
	"testing"
)

var (
	l  *log.Logger
	ts *Timeslider
)

func newTimeslider() error {
	var c *Conf
	var err error

	if err = jsonconf.UnmarshalFile("/home/username/timeslider/testConf.json", &c); err != nil {
		fmt.Println("tsConf")
		return err
	}

	l = log.NewLogger()
	if ts, err = NewTimeslider(l, c); err != nil {
		return err
	}
	return nil
}

func TestGenerateTimeslider(t *testing.T) {
	var err error
	if err = newTimeslider(); err != nil {
		t.Log("NewTimeslider")
		t.Log(err)
		t.Fail()
		return
	}

	src := "/pathtotestfile/240p.mp4"
	finalDst := "temp/outputpath/"

	if err := ts.GenerateTimeslider(src, finalDst); err != nil {
		t.Log(err)
		t.Fail()
		return
	}

	//create fake mp4 file
	fi := "360p.mp4"
	path := "tmp/input/111/"
	src = path + fi
	dst := "tmp/output/111/"

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err = os.MkdirAll(path, 0777); err != nil {
			t.Log(err)
			t.Fail()
			return
		}
	}

	if _, err := os.Create(src); err != nil {
		t.Log(err)
		t.Fail()
		return
	}
	//test to detect error
	if err := ts.GenerateTimeslider(src, dst); err == nil {
		t.Log("error not detected")
		t.Fail()
		return
	}
}

func TestFindSrcFile(t *testing.T) {
	var err error
	var file string
	srcPath := "tmp/input/sourcefile.mp4"

	if err = newTimeslider(); err != nil {
		t.Log("NewTimeslider")
		t.Log(err)
		t.Fail()
		return
	}

	if file, err = ts.FindSrcFile(srcPath); err != nil {
		t.Log(err)
		t.Fail()
		return
	}
	if file == "" {
		t.Log("src file missing for 102414-720")
		t.Fail()
		return
	}

	srcPath = "tpm/input/srcpath"

	if file, err = ts.FindSrcFile(srcPath); err != nil {
		t.Log(err)
		t.Fail()
		return
	}
	if file != "" {
		t.Log("src file is not supported")
		t.Fail()
	}

}

func TestGenerateTileImage(t *testing.T) {
	var err error
	if err = newTimeslider(); err != nil {
		t.Log("NewTimeslider")
		t.Log(err)
		t.Fail()
		return
	}
	//using fake mp4 file
	input := "tmp/input/360p.mp4"
	dst := "tmp/output/"

	if err := ts.GenerateTileImage(input, dst); err == nil {
		t.Log(err)
		t.Fail()
		return
	}

	input = "tmp/input/240p.mp4"
	dst = "tmp/output/"

	if err := ts.GenerateTileImage(input, dst); err == nil {
		t.Log("err not detected genarate thumb")
		t.Fail()
		return
	}

}

func TestGenerateVttFile(t *testing.T) {
	var err error
	var duration float64 = 3460

	if err = newTimeslider(); err != nil {
		return
	}

	dst := "tmp/output"

	if err := ts.GenerateVttFile(dst, duration); err == nil {
		t.Log("Vtt file not created")
		t.Fail()
		return
	}

	dst = "tmp/output"

	if err := ts.GenerateVttFile(dst, duration); err == nil {
		t.Log("err not detected VTT file")
		t.Fail()
		return
	}
}

func TestMoveTmpFileToDest(t *testing.T) {
	var err error
	if err = newTimeslider(); err != nil {
		return
	}

	tmpDst := "tmp/tmpfolder/"
	dst := "tmp/output/"

	if err := ts.MoveTmpFilesToDest(tmpDst, dst); err == nil {
		t.Log("err not detected TestMoveTmpFileToDest")
		t.Fail()
		return
	}

	tmpDst = "tmp/output/111111_111/"
	dst = "tmp/output/222222_222/"

	if _, err := os.Stat(tmpDst); os.IsNotExist(err) {
		if err := os.MkdirAll(tmpDst, 0777); err != nil {
			t.Log(err)
			t.Fail()
		}
	}

	f := "thumbnail.vtt"
	file := tmpDst + f
	if _, err := os.Create(file); err != nil {
		t.Log("err creating test file")
		t.Fail()
	}

	if err := ts.MoveTmpFilesToDest(tmpDst, dst); err != nil {
		t.Log("err not detected TestMoveTmpFileToDest")
		t.Fail()
		return
	}

}

func TestCanOverwriteFiles(t *testing.T) {
	var err error
	if err = newTimeslider(); err != nil {
		return
	}
	//dst doesn't exist
	srcFile := "tmp/input/360p.mp4"
	dst := "tmp/output/"

	if ts.CanOverwriteFiles(srcFile, dst) {
		t.Log("return true for overwriting file")
		t.Fail()
		return
	}

	srcFile = "tmp/input/240p.mp4"
	dst = "tmp/output/"

	if ts.CanOverwriteFiles(srcFile, dst) {
		t.Log("return false for overwriting file")
		t.Fail()
		return
	}
}
