package main

import (
	"mercury/auto/timeslider"
	"mercury/core/log"
	"mercury/core/start"
	"flag"
	"io/ioutil"
	"os"
	"strings"
)

var (
	target string
)

func init() {
	flag.StringVar(&target, "target", "", "Set Target movies")
}

func usage() {
	flag.PrintDefaults()
	os.Exit(2)
}

type Conf struct {
	TS   timeslider.Conf
	Logs log.NameLevels
}

func main() {
	var c Conf
	var err error
	var retryMovieIds [][]string
	var checkDir string

	l := start.Start(&c, true)

	ts, err := timeslider.NewTimeslider(l.NewName("Timeslider"), &c.TS)
	if err != nil {
		l.Emerg("Failed to start Timeslider error -", err)
		return
	}

	flag.Parse()
	if target == "" {
		usage()
	}

	l.Notice("Timeslider started")

	//If target flag is "news", get movieIds from finished Encoder folder,
	//If target flag is "full", get movieIds from src folder in production,
	//If target flag is "force", get movieIds from config file.
	//loop each site in config to check src directory and see if there are new inputs
	for _, s := range c.TS.SC {
		var movieLists []os.FileInfo

		//if target is "full" check src directory and get movie_id
		if target == "full" {
			checkDir = c.TS.Dir.Src
			Dir := strings.Replace(checkDir, "%site", s.SitePath, 1)
			movieLists, err = ioutil.ReadDir(Dir)
			if err != nil {
				l.WarnF("Src directory error - %s - dir - %s", err, Dir)
				continue
			}

		//if target is "news" check finished directory to get movie_id and src
		} else if target == "news" {
			checkDir := c.TS.Dir.News //data/auto/encoder/finished/{site}/
			Dir := strings.Replace(checkDir, "%site", s.SitePath, 1)
			movieLists, err = ioutil.ReadDir(Dir)
			if err != nil {
				l.WarnF("News(Encode Finished) directory error - %s - %s", err, Dir)
				continue
			}

		//if target is "force" check movie_id of forceMovieIds in conf
		} else if target == "force" {
			if len(s.MovieIds) > 0 {
				for _, mi := range s.MovieIds {
					var movie_id os.FileInfo
					checkDir := c.TS.Dir.Src //data/www/{site}/member/
					Dir := strings.Replace(checkDir, "%site", s.SitePath, 1) + mi
					if movie_id, err = os.Stat(Dir); err != nil { 
						if os.IsNotExist(err) {
							l.WarnF("Src can't find for force movie IDs. error: %s dir: %s", err, Dir)
						} else {
							l.WarnF("Src Dir err - %s", err, Dir)
						}
						continue
					}
					movieLists = append(movieLists, movie_id)
				}
			}
		} else {
			l.WarnF("target movie has not been set")
			return
		}

		if len(movieLists) > 0 {
			for _, m := range movieLists {
                                //if target is news, check valid movie_id (to skip {movie_id.timestamp})
                                if target == "news" {
					movie := m.Name()
                                        if !ts.IsThisValidMovie(s.SiteId, movie) {
						l.NoticeF("movie id is not valid : %s", s.SiteId)
						continue
					}
				}
				//Find a src file to generate thumbnails
				var srcFile string
				///data/www/{sitename}/member/movie_id
				p := strings.Replace(c.TS.Dir.Src, "%site", s.SitePath, 1)
				srcPath := p + m.Name() + "/"
				if srcFile, err = ts.FindSrcFile(srcPath); err != nil {
					l.WarnF("find srcfile error - %s src - %s", err, srcPath)
				}
				//Skip if srcFile is not found
				if srcFile == "" {
					continue
				}

				//Check if the folder already exist in dst, if so then check timestamp
				//Production path
				///www/{sitedomain}/html/member/ts/{movieId}/
				dst := strings.Replace(c.TS.Dir.FinalDst, "%site", s.SiteName, 1) + m.Name() + "/"

				//if target is news or full, check dst and see it can orverwrite existing timeslider.
				if target == "news" || target == "full" {
					if dir, _ := ioutil.ReadDir(dst); dir != nil {
						if !ts.CanOverwriteFiles(srcFile, dst) {
							l.NoticeF("Timeslider already exists - %s, - %s", s.SiteName, m.Name())
							continue
						}
					}
				}
				//Generate timeslider
				if err := ts.GenerateTimeslider(srcFile, dst); err != nil {
					//if fail, keep movie_id in array to retry later
					sets := []string{srcFile, dst}
					retryMovieIds = append(retryMovieIds, sets)
					l.WarnF("Generating timeslider failed!! src - %s - %s", sets, err)
				}
			}
		} else {
			l.NoticeF("No new movie to generate timeslider. Site: %s", s.SitePath)
		}
	}

	//Retry failed movie ids
	var numberOfRetry = c.TS.Retry
	if len(retryMovieIds) > 0 {
		for i := range retryMovieIds {
			for k := 0; k < numberOfRetry; k++ {
				err := ts.GenerateTimeslider(retryMovieIds[i][0], retryMovieIds[i][1])
				if err != nil {
					l.WarnF("Retry generating timeslider failed!! src - %s - %s", retryMovieIds[i], err)
				} else {
					l.NoticeF("Retry success -%s", retryMovieIds[i])
					break
				}
			}
		}
	} else {
		l.Notice("No retryMovieIds to generate timeslider")
	}
	l.Notice("Timeslider done")
}
