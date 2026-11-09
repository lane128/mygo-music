package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/music"
)

// A player heart with plugin text still likes the current song without changing its text or playback selection.
func TestPlayerLikeCaptionClick(t *testing.T) {
	a := newTestApp()
	without(a, "isLiked")
	a.account = &music.Account{ID: "listener"}
	track := song("playing", "Playing")
	track.LikeCountText = "1w+"
	a.pl.Cue([]music.Track{track})
	calls := make(chan bool, 2)
	fake(a).like = func(id string, on bool) error {
		if id != track.ID {
			t.Errorf("liked %q, want %q", id, track.ID)
		}
		calls <- on
		return nil
	}
	a.pl.SetLyrics(nil)
	tt := ui.NewTester(a.view, 880, 780)
	settled(tt)
	if err := tt.Click("Like"); err != nil {
		t.Fatal(err)
	}
	if !wait(t, calls, "the like") || !a.liked[track.ID] {
		t.Fatal("the caption's heart did not like the song")
	}
	tt.Frame()
	if err := tt.Click("Unlike"); err != nil {
		t.Fatal(err)
	}
	if wait(t, calls, "the unlike") || a.liked[track.ID] {
		t.Fatal("the caption's heart did not unlike the song")
	}
	if got := a.pl.Current(); got.ID != track.ID || got.LikeCountText != "1w+" || a.nowPlaying {
		t.Fatalf("heart changed playback or plugin text: %+v, now playing %v", got, a.nowPlaying)
	}
}

// Unavailable songs and episodes refuse row playback but keep their hearts usable in either theme.
func TestUnavailableRowsKeepLikes(t *testing.T) {
	for _, kind := range []string{"track", "episode", "queue"} {
		for _, dark := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/dark=%v", kind, dark), func(t *testing.T) {
				a := newTestApp()
				without(a, "isLiked")
				a.account = &music.Account{ID: "listener"}
				playing := song("playing", "Playing")
				unavailable := song("unavailable", "Unavailable song")
				unavailable.Unavailable = true
				if kind == "episode" {
					unavailable.ShowID = "show"
				}
				tracks := []music.Track{playing, unavailable}
				a.pl.Cue(tracks)
				a.liked = map[string]bool{unavailable.ID: true}
				calls := make(chan string, 2)
				like := func(id string, on bool) error {
					if on {
						t.Error("asked to like an already liked song")
					}
					calls <- id
					return nil
				}
				fake(a).like, fake(a).likeEpisode = like, like
				tt := ui.NewTester(func(c *ui.Context) {
					a.pal = paletteFor(c.Theme())
					ui.Column(c).Grow(1).Children(func() {
						switch kind {
						case "track":
							a.trackRow(c, tracks, 1, true, "")
						case "episode":
							d := &showData{show: &music.Show{ID: "show"}, episodes: []music.Track{unavailable}}
							a.episodeRow(c, d, 0, a.pl.Current())
						case "queue":
							a.queuePane(c)
						}
					})
				}, 1000, 500)
				tt.SetDark(dark)
				settled(tt)
				// Gray styling alone used to leave screen readers unaware that the row cannot play.
				if info, ok := tt.AccessByRole(ui.RoleButton, unavailable.Name); !ok || info.Description == "" {
					t.Fatalf("the unavailable row has no accessible playback indication: %+v; tree: %+v", info, tt.AccessAll())
				}
				// A disabled parent used to disable an unavailable episode's heart too.
				if err := tt.Click(unavailable.Name); err != nil {
					t.Fatal(err)
				}
				tt.Frame()
				if a.pl.Current().ID != playing.ID {
					t.Fatal("the unavailable row changed playback")
				}
				if err := tt.Click("Unlike"); err != nil {
					t.Fatal(err)
				}
				if id := wait(t, calls, "the unlike"); id != unavailable.ID || a.liked[id] {
					t.Fatalf("unliked %q, liked state %v", id, a.liked)
				}
				if a.pl.Current().ID != playing.ID {
					t.Fatal("liking the unavailable song changed playback")
				}
			})
		}
	}
}

// With MM_FRAMES set, saves unavailable lists and player captions in both themes for visual inspection.
func TestTrackStatusFrames(t *testing.T) {
	dir := framesDir(t)
	for _, dark := range []bool{false, true} {
		for _, width := range []int{880, 1180} {
			a := newTestApp()
			a.account = &music.Account{ID: "listener", Name: "Alex"}
			tracks := framesTracks(a)
			tracks[0].LikeCountText = "1w+"
			tracks[1].Unavailable = true
			tracks[2].Unavailable, tracks[2].Album.ImageURL = true, ""
			a.liked = map[string]bool{tracks[0].ID: true, tracks[1].ID: true, tracks[2].ID: true}
			a.pl.Cue(tracks)
			a.pl.SetLyrics(nil)
			a.playlists["list"] = &music.Playlist{ID: "list", Name: "Evening Walks", Tracks: tracks}
			a.route = route{page: pagePlaylist, src: "fake", id: "list"}
			tt := ui.NewTester(a.view, width, 780)
			tt.SetDark(dark)
			settled(tt)
			saveFrame(t, dir, fmt.Sprintf("track-status-%v-%d.png", dark, width), tt.Image())
			if width == 1180 {
				a.nowPlaying, a.npPane = true, paneQueue
				tt.Advance(time.Second)
				settled(tt)
				saveFrame(t, dir, fmt.Sprintf("track-status-now-%v.png", dark), tt.Image())
				for _, pane := range []struct {
					id   int
					name string
				}{{paneStage, "stage"}, {paneWall, "wall"}} {
					a.npPane = pane.id
					tt.Move(600, 400)
					tt.Advance(time.Second)
					settled(tt)
					saveFrame(t, dir, fmt.Sprintf("track-status-%s-%v.png", pane.name, dark), tt.Image())
				}
				a.nowPlaying = false
				a.stations, a.stationsOnce = []music.Station{{ID: "st", Name: "Night Drive"}}, true
				a.route = route{page: pageStations, src: "fake"}
				a.pl.AttachRadio(radioName("fake", &a.stations[0]), func(func([]music.Track)) {})
				tt.Advance(time.Second)
				settled(tt)
				saveFrame(t, dir, fmt.Sprintf("track-status-radio-%v.png", dark), tt.Image())
				a.nowPlaying, a.npPane = true, paneQueue
				// A source may supply totals without a personal-like operation.
				without(a, "like")
				settled(tt)
				saveFrame(t, dir, fmt.Sprintf("track-status-readonly-%v.png", dark), tt.Image())
			}
		}
	}
}
