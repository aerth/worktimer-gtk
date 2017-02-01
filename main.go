// command worktimer-gtk is a notification bar icon that works
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/aerth/filer"
	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
)

var titlestr = workmode

const breakmode = "Break Mode"
const workmode = "Work Mode"

var filename string
var (
	prefix       = flag.String("prefix", "", "Prefix for filename")
	justclockin  = flag.Bool("on", false, "Just punch in, don't launch status icon.")
	justclockout = flag.Bool("off", false, "Just punch out, don't launch status icon.")
	icon         = flag.Bool("icon", false, "Punch in and launch icon.")
	decode       = flag.Bool("decode", false, "Just print total hours and exit. Need -in flag.")
)
var working bool

func init() {
	flag.StringVar(&filename, "o", "", "Save to file")
	flag.StringVar(&filename, "in", "", "Decode file. Need -decode flag")
}

var start time.Time

// PunchIn is a duration of work, or a start time. Or a finish time.
// To save bytes in the Marshal
type PunchIn struct {
	Started time.Time `json:",omitempty"`
}

// PunchOut is a duration of work, or a start time. Or a finish time.
type PunchOut struct {
	PunchIn
	Started  time.Time     `json:",omitempty"`
	Finished time.Time     `json:",omitempty"`
	Duration time.Duration `json:",omitempty, string"`
}

var punchcards []PunchOut

func clockin() {
	var punch PunchIn
	punch.Started = time.Now()
	start = punch.Started

	filer.Touch(filename)
	// save to punchcard
	b, err := json.Marshal(&punch)
	if err != nil {
		panic(err)
	}
	filer.Append(filename, append(b, []byte("\n")...)) // !!!
	fmt.Printf("Working since %s.\nSaving to %q.\n", start, filename)
	working = true
}

func clockout() {
	var punch PunchOut
	now := time.Now()

	if start.IsZero() {
		start = getlastpunchin()
	}
	punch.Started = start
	punch.Finished = now
	punch.Duration = punch.Finished.Sub(punch.Started)
	// reset timer for breaks
	start = now
	working = false

	// save to punchcard
	b, err := json.Marshal(&punch)
	if err != nil {
		panic(err)
	}
	err = filer.Append(filename, append(b, []byte("\n")...)) // !!!
	if err != nil {
		panic(err)
	}

	fmt.Printf("Worked from %s to %s, total of %s.\n", punch.Started, punch.Finished, punch.Duration)
}
func main() {
	flag.Parse()
	if *decode && filename != "" {
		fmt.Println(gettotal())
		os.Exit(0)
	}
	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(1)
	}
	if flag.Args() != nil {
		for _, v := range flag.Args() {
			if strings.HasPrefix(v, "-") {
				flag.Usage()
				os.Exit(1)
			}
		}
	}

	if filename == "" {
		filename = "./" + *prefix + time.Now().Format("Jan2006") + ".json"
	}
	if *justclockin && !*justclockout {
		clockin()
		os.Exit(0)
	}

	if *justclockout && !*justclockin {
		clockout()
		os.Exit(0)
	}

	if *icon {
		iconlaunch()
	}

}

func iconlaunch() {
	working = true
	glib.ThreadInit(nil)
	gdk.ThreadsInit()
	gdk.ThreadsEnter()
	gtk.Init(nil)

	glib.SetApplicationName(titlestr)
	go func() {
		for {

			finish := time.Now()
			total := finish.Sub(start)

			if !working {
				fmt.Printf("Not working. Stopped at %s, counting %s.\n", start, total)
			} else {

				fmt.Printf("Worked from %s to %s, counting %s.\n", start, finish, total)
			}
			//filer.Append(filename, []byte(fmt.Sprintf("Worked from %s to %s, counting %s.\n", start, finish, total)))
			time.Sleep(time.Second * 1)
		}
	}()
	clockin()
	defer clockout()

	icon := gtk.NewStatusIconFromStock(gtk.STOCK_YES)
	remenu := gtk.NewMenu()
	clockinBtn := gtk.NewMenuItemWithLabel("Clock In")
	quitBtn := gtk.NewMenuItemWithLabel("Exit")
	clockoutBtn := gtk.NewMenuItemWithLabel("Clock Out")
	startedlabel := gtk.NewMenuItemWithLabel("Started: " + start.Format(time.Kitchen))
	breaklabel := gtk.NewMenuItemWithLabel("Stopped.")
	g := gdk.NewColor("green")
	r := gdk.NewColor("lightgrey")
	clockinBtn.Connect("activate", func() {
		icon.SetFromStock(gtk.STOCK_YES)
		clockinBtn.SetVisible(false)
		clockoutBtn.SetVisible(true)
		quitBtn.SetVisible(false)
		breaklabel.SetVisible(false)
		startedlabel.SetVisible(true)
		titlestr = workmode
		icon.SetTooltipMarkup(fmt.Sprintf("<span color='green'>%s</span>", titlestr))
		glib.SetApplicationName(titlestr)

		remenu.ModifyBG(gtk.STATE_NORMAL, g)

		clockin()

	})
	clockoutBtn.Connect("activate", func() {
		icon.SetFromStock(gtk.STOCK_NO)
		clockinBtn.SetVisible(true)
		clockoutBtn.SetVisible(false)
		quitBtn.SetVisible(true)
		breaklabel.SetVisible(true)
		startedlabel.SetVisible(false)
		titlestr = breakmode
		icon.SetTooltipMarkup(fmt.Sprintf("<span color='red'>%s</span>", titlestr))
		glib.SetApplicationName(titlestr)
		remenu.ModifyBG(gtk.STATE_NORMAL, r)

		clockout()

	})
	quitBtn.Connect("activate", func() {
		gtk.MainQuit()
	})
	remenu.Append(clockoutBtn)
	remenu.Append(clockinBtn)
	remenu.Append(quitBtn)
	remenu.Append(startedlabel)

	remenu.Append(breaklabel)
	icon.SetTitle(titlestr)
	icon.SetTooltipMarkup(fmt.Sprintf("<span color='green'>%s</span>", titlestr))
	go func() {
		for {
			time.Sleep(1 * time.Second)
			gdk.ThreadsEnter()
			remenu.SetTooltipText(time.Now().Sub(start).String())
			if !working {
				icon.SetTooltipMarkup(fmt.Sprintf("<span color='red'>%s</span> %s", titlestr, time.Now().Sub(start).String()))
			} else {
				icon.SetTooltipMarkup(fmt.Sprintf("<span color='green'>%s</span> %s", titlestr, time.Now().Sub(start).String()))
			}
			gdk.ThreadsLeave()
		}
	}()
	icon.Connect("popup-menu", func(cbx *glib.CallbackContext) {
		remenu.Popup(nil, nil, gtk.StatusIconPositionMenu, icon, uint(cbx.Args(0)), uint32(cbx.Args(1)))
	})
	remenu.ShowAll()
	remenu.ModifyBG(gtk.STATE_NORMAL, g)
	clockinBtn.SetVisible(false)
	quitBtn.SetVisible(false)
	breaklabel.SetVisible(false)
	gtk.Main()
}
func getlastpunchin() time.Time {

	file, er := os.Open(filename)
	if er != nil {
		fmt.Println(er)
		os.Exit(1)
	}
	var latest time.Time
	var punchins []PunchIn
	scanner := bufio.NewScanner(file)
	var scan int
	for scanner.Scan() {
		scan++
		var p PunchIn
		er = json.Unmarshal(scanner.Bytes(), &p)
		if er != nil {
			// log it
			continue
		}

		if p.Started.Sub(latest) > 0 {
			latest = p.Started
		}
		punchins = append(punchins, p)
	}

	return latest
}
func gettotal() string {

	file, er := os.Open(filename)
	if er != nil {
		fmt.Println(er)
		os.Exit(1)
	}
	var punchouts []PunchOut
	scanner := bufio.NewScanner(file)
	var scan int
	for scanner.Scan() {
		scan++
		var p PunchOut
		er = json.Unmarshal(scanner.Bytes(), &p)
		if er != nil {
			// log it panic(er)
			continue
		}

		punchouts = append(punchouts, p)
	}
	var x, y, z big.Int
	z.SetInt64(0)
	for _, v := range punchouts {
		if int64(v.Duration) > 0 {
			//fmt.Println(v.Duration)
			x.SetInt64(int64(v.Duration))
			z.Add(&x, &z)
		}
	}
	y.SetInt64(int64(time.Hour))
	//	fmt.Println(z.String() + " nanoseconds / " + y.String() + " nanoseconds = ")
	var f1, f2, f3 big.Float

	f1.SetInt(&z)
	f2.SetInt(&y)
	f3.Quo(&f1, &f2)

	return f3.String() + " Hours"
}
