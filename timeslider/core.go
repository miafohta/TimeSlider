package timeslider

import (
	"mercury/auto/media"
	"mercury/core/jsonconf"
	"mercury/core/log"
	"mercury/core/util"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// func NewTimeslider {{{
func NewTimeslider(l log.LibLog, c *Conf) (*Timeslider, error) {
	var err error
	t := &Timeslider{l: l, c: c}

	//Config Sanity check
	if t.c == nil {
		return nil, fmt.Errorf("Missing conf file")
	}

	//Check values in config file
	if t.c.Dir.News == "" || t.c.Dir.Src == "" || t.c.Filename == "" || t.c.Dir.TmpDir == "" || t.c.Mp4[0] == "" {
		return nil, fmt.Errorf("Directories are not defined in config")
	}

	//Check physical paths in config
	for _, d := range c.SC {
		dir := strings.Replace(t.c.Dir.Src, "%site", d.SitePath, 1)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return nil, fmt.Errorf("input in conf directory does not exist")
		}
	}

	if _, err := os.Stat(t.c.Dir.TmpDir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("tmp in conf directory does not exist")
		} else {
			return nil, fmt.Errorf("tmp in conf direcory can't not read")
		}
	}

	//Load Media conf
	if err = jsonconf.UnmarshalFile(t.c.MdConf, &c.Mc); err != nil {
		return nil, fmt.Errorf("Media conf couldn't be parsed -%s", err)
	}
	//Load Media package
	if t.mi, err = media.NewMedia(l.NewMiniName(log.LOG_NORMAL, "Media"), &c.Mc); err != nil {
		return nil, fmt.Errorf("Media package err - %s", err)
	}

	return t, nil
} // }}}

//func GenerateTimeslider {{{
//Generate timeline thumbnail, vtt file in tmp dir, then move them to prod.
func (t *Timeslider) GenerateTimeslider(src, finalDst string) error {
	path, _ := filepath.Split(src)
	movie_id := filepath.Base(path)
	tmpDst := t.c.Dir.TmpDir + movie_id + "/"
	t.l.Debug(tmpDst)
	if err := t.GenerateTileImage(src, tmpDst); err != nil {
		//remove tmpDir if failed
		if err := os.RemoveAll(tmpDst); err != nil {
			return err
		}
		return err
	}
	if err := t.MoveTmpFilesToDest(tmpDst, finalDst); err != nil {
		//remove tmpDir if failed
		if err := os.RemoveAll(tmpDst); err != nil {
			return err
		}
		return err
	}
	if err := os.RemoveAll(tmpDst); err != nil {
		return err
	}
	return nil
} //}}}

//func GenerateTileImage {{{
func (t *Timeslider) GenerateTileImage(input, dst string) error {
	//create tmp folder for output
	_, err := ioutil.ReadDir(dst)
	if err != nil {
		if os.IsNotExist(err); err != nil {
			if err = os.MkdirAll(dst, 0755); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	//call media package to get movie info
	mInfo, err := t.mi.GetMediaInfo(input)
	if err != nil {
		return err
	}

	//calculate thumbnail frame option for ffmpeg command based on fps
	var fps int
	fps = int(mInfo.FrameRate * 10)
	rate := strconv.Itoa(fps)

	t.l.Debug(rate)

	//Parse thumbnail options to ffmpeg command
	cmd := strings.Replace(cmdOutputThumb, "%scale", t.c.Scale, 1)
	cmd = strings.Replace(cmd, "%input", input, 1)
	cmd = strings.Replace(cmd, "%fps", rate, 1)
	cmd = strings.Replace(cmd, "%tile", t.c.Tile, 1)
	cmd = strings.Replace(cmd, "%dst", dst, 1)

	var cmnd []string
	cmnd = strings.Split(cmd, " ")

	//run commands
	if err := util.RunCommand(t.l, cmnd); err != nil {
		return err
	}

	var duration float64
	duration = float64(mInfo.Duration)

	if err := t.GenerateVttFile(dst, duration); err != nil {
		return err
	}
	t.l.Debug(dst, duration)
	return nil
} //}}}

//func GenerateVTTFile {{{
func (t *Timeslider) GenerateVttFile(dst string, duration float64) error {
	img, err := ioutil.ReadDir(dst)
	if err != nil {
		return err
	}

	//Count the number of images so you can set the number of loop
	//to output lines of VTT file
	if len(img) < 0 {
		return fmt.Errorf("Thumbnail not found for - %s", dst)
	}

	fileNum := len(img) + 1
	loopCount := fileNum * t.c.V.NumberOfTiles
	imgPosition := make([]string, loopCount)
	t.l.Debug(loopCount)
	t.l.Debug(imgPosition)
	//Set VTT format of image and position
	prf := "output00"
	psf := ".jpg#xywh="
	count := 1
	n := 0

	//Insert each image position to an array to output later
	for i := 0; i < fileNum; i++ {
		if i > 8 {
			prf = "output0"
		}
		for j := 0; j < len(t.c.V.Y); j++ {
			for k := 0; k < len(t.c.V.X); k++ {
				imgPosition[n] = prf + strconv.Itoa(count) + psf + t.c.V.X[k] + "," + t.c.V.Y[j] + "," + t.c.V.W + "," + t.c.V.H + "\n" + "\n"
				n += 1
			}
		}
		count += 1
	}

	//Output timelines for VTT file.
	//
	//Example of a line
	//00:00:00.000  -->  00:00:10.000
	//output001.jpg#xywh=0,0,162,90
	//
	//Divide duration by 10 seconds to get the number of lines.

	loopCount = int(math.Floor(duration / 10))
	t.l.Debug(loopCount)
	t.l.Debug(n)
	var fromTime string
	var toTime string
	fromToTime := make([]string, loopCount)

	//loop for duration
	interval := float64(0)

	for i := 0; i < loopCount; i++ {
		if i == 0 {
			fromToTime[i] = "WEBVTT\n\n"
		} else {
			fromTime = t.formatDuration(interval)
			toTime = t.formatDuration(interval + 10)
			if i > n {
				break
			}
			fromToTime[i] = fromTime + "  -->  " + toTime + "\n" + imgPosition[i] + "\n"
			interval += 10
		}
	}

	filename := dst + t.c.Filename
	fout, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer fout.Close()

	for i := 0; i < len(fromToTime); i++ {
		if _, err = fout.WriteString(fromToTime[i]); err != nil {
			return err
		}
	}
	return nil
} //}}}

//func formatDuration {{{
func (t *Timeslider) formatDuration(interval float64) string {
	var hh string
	var mm string
	var ss string

	h := int(math.Floor(interval / 60 / 60))
	v := h * 60
	m := int(math.Floor(interval/60)) - v
	hm := (v * 60) + (m * 60)
	s := int(interval) - hm

	hh = strconv.Itoa(h)
	if m < 10 {
		mm = "0" + strconv.Itoa(m)
	} else {
		mm = strconv.Itoa(m)
	}
	if s < 10 {
		ss = "0" + strconv.Itoa(s)
	} else {
		ss = strconv.Itoa(s)
	}

	time := "0" + hh + ":" + mm + ":" + ss + ".000"

	return time
} //}}}

//func MoveTmpFilesToDest {{{
func (t *Timeslider) MoveTmpFilesToDest(tmpDst, dst string) error {
	//Read tmp folder
	var files []os.FileInfo
	var err error
	if files, err = ioutil.ReadDir(tmpDst); err != nil {
		return err
	}
	if len(files) > 0 {
		if _, err := os.Stat(dst); err != nil {
			if os.IsNotExist(err) {
				if err = os.MkdirAll(dst, 0755); err != nil {
					return err
				}
			} else {
				return err
			}
		}
		for _, f := range files {
			srcFile := tmpDst + f.Name()
			dstFile := dst + f.Name()
			if err := t.copyFileContents(srcFile, dstFile); err != nil {
				return err
			}
		}
	}
	return nil
} //}}}

//func copyFileContents {{{
func (t *Timeslider) copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

//}}}

//func FindSrcFile {{{
//This function looks for the src mp4 file. 360p is set as default in config.
//Also check the src file is modified at least one hour ago so the src file is complete
func (t *Timeslider) FindSrcFile(srcPath string) (string, error) {
	var srcFile string
	var err error
	for _, s := range t.c.Mp4 {
		srcFile = srcPath + s
		t.l.InfoF("src -%s", srcFile)
		var f os.FileInfo
		if f, err = os.Stat(srcFile); err != nil {
			if os.IsNotExist(err) {
				srcFile = ""
				continue
			} else {
				t.l.WarnF("error reading src file")
				continue
			}
		}
		//Make sure the src file has been created at least one hour ago
		now := time.Now()
		oneHourAfter := f.ModTime().Add(time.Duration(10) * time.Minute)
		t.l.Info(oneHourAfter)
		if now.Before(oneHourAfter) {
			t.l.InfoF("src is not complete - %s", srcFile)
			srcFile = ""
		} else {
			t.l.InfoF("src ok - %s", srcFile)
			break
		}
	}
	return srcFile, nil
} //}}}

//func CanOverwriteFiles {{{
//This function checks a timestamp of existing timeslider files
// and returns whether it needs overwritten
func (t *Timeslider) CanOverwriteFiles(srcFile, dst string) bool {
	var isVtt int = 0
	var isImg int = 0
	var fileList []os.FileInfo
	var dst_ftime time.Time
	var src_ftime time.Time

	//read dst folder
	fileList, err := ioutil.ReadDir(dst)
	if err != nil {
		t.l.WarnF("Dst dir couldn't read - %s - %s", dst, err)
		return false
	}

	//check if thumbnail & vtt already exist in the folder
	if len(fileList) > 0 {
		for _, f := range fileList {
			_, fname := filepath.Split(f.Name())
			if fname == t.c.Filename {
				dst_ftime = f.ModTime()
				isVtt = 1
				continue
			}
			if fname == t.c.ImgFilename {
				isImg = 1
				continue
			}
		}

		//if thumbnail and vtt files exist, compare modified time to src
		if isVtt == 1 && isImg == 1 {
			fi, err := os.Stat(srcFile)
			if err != nil {
				t.l.WarnF("srcFile couldn't read - %s - %s", srcFile, err)
				return false
			}
			src_ftime = fi.ModTime()
			//if if dst file is newer than src, return false to recreate thumbnails
			if dst_ftime.After(src_ftime) {
				return false
			}
		}
	}
	return true
} //}}}

// func IsThisValidMovie {{{
func (t *Timeslider) IsThisValidMovie(siteId uint32, movie string) bool {
	if err := t.mi.IsKnownMovie(siteId, movie); err != nil {
		t.l.Info("Movie Id is not valid : %s - %s", movie, err)
		return false
	}
	return true
} //}}}
