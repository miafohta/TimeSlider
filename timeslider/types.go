package timeslider

import (
	"mercury/core/log" // "dtibase/log"
	//"os"
)

type Timeslider struct {
	l  log.LibLog
	c  *Conf
	mi *media.Media
}

type Conf struct {
	Dir         Directories
	Mp4         []string
	Filename    string
	ImgFilename string
	Mc          media.Conf
	MdConf      string
	SC          []SiteConf
	V           Vtt
	Logs        log.NameLevels
	Retry       int

        //command options
	Scale       string
	Tile        string
}

type Directories struct {
	FinalDst, TmpDir, News, Src string
}

type Vtt struct {
	NumberOfTiles int
	W             string
	H             string
	X             []string
	Y             []string
}

type SiteConf struct {
	SiteName, SitePath string
	SiteId		   uint32 
	MovieIds           []string
}

var (
	//ffmpeg command
	cmdGetInfo     = "ffmpeg -i %input"
	cmdOutputThumb = "/usr/local/bin/ffmpeg -loglevel error -i %input -vf scale=%scale,thumbnail=%fps,tile=%tile -q 1 -an -vsync 0 %dstoutput%03d.jpg"
)
