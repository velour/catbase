package meme

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/config"
)

type MemePlugin struct {
	bot bot.Bot
	c   *config.Config

	images cachedImages
}

type cachedImage struct {
	created time.Time
	repr    []byte
}

var horizon = 24 * 7

type cachedImages map[string]*cachedImage

func (ci cachedImages) cleanup() {
	for key, img := range ci {
		if time.Now().After(img.created.Add(time.Hour * time.Duration(horizon))) {
			delete(ci, key)
		}
	}
}

func New(b bot.Bot) *MemePlugin {
	mp := &MemePlugin{
		bot:    b,
		c:      b.Config(),
		images: make(cachedImages),
	}

	horizon = mp.c.GetInt("meme.horizon", horizon)

	b.Register(mp, bot.Message, mp.message)
	b.Register(mp, bot.Help, mp.help)
	mp.registerWeb(b.DefaultConnector())

	return mp
}

func (p *MemePlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	return false
}

func (p *MemePlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	webRoot := p.c.Get("BaseURL", "https://catbase.velour.ninja")
	formats := p.c.GetMap("meme.memes", defaultFormats)
	msg := "Use `/meme [format] [text]` to create a meme.\nI know the following formats:"
	msg += "\n`[format]` can be a URL"
	msg += fmt.Sprintf("\nor a format from the list of %d pre-made memes listed on the website", len(formats))
	msg += fmt.Sprintf("\nHead over to %s/meme to view and add new meme formats", webRoot)
	msg += "\nYou can use `_` as a placeholder for empty text and a newline to separate top vs bottom."
	p.bot.Send(c, bot.Message, message.Channel, msg)
	return true
}

func (p *MemePlugin) registerWeb(c bot.Connector) {
	http.HandleFunc("/slash/meme", p.slashMeme(c))
	http.HandleFunc("/meme/img/", p.img)
	http.HandleFunc("/meme/all", p.all)
	http.HandleFunc("/meme/add", p.addMeme)
	http.HandleFunc("/meme/rm", p.rmMeme)
	http.HandleFunc("/meme", p.webRoot)
	p.bot.RegisterWeb("/meme", "Memes")
}

type webResp struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type webResps []webResp

func (w webResps) Len() int      { return len(w) }
func (w webResps) Swap(i, j int) { w[i], w[j] = w[j], w[i] }

type ByName struct{ webResps }

func (s ByName) Less(i, j int) bool { return s.webResps[i].Name < s.webResps[j].Name }

func (p *MemePlugin) all(w http.ResponseWriter, r *http.Request) {
	memes := p.c.GetMap("meme.memes", defaultFormats)
	values := webResps{}
	for n, u := range memes {

		realURL, err := url.Parse(u)
		if err != nil || realURL.Scheme == "" {
			realURL, _ = url.Parse("https://imgflip.com/s/meme/" + u)
		}
		values = append(values, webResp{n, realURL.String()})
	}
	sort.Sort(ByName{values})

	out, err := json.Marshal(values)
	if err != nil {
		w.WriteHeader(500)
		log.Error().Err(err).Msgf("could not serve all memes route")
		return
	}
	w.Write(out)
}

func mkCheckError(w http.ResponseWriter) func(error) bool {
	return func(err error) bool {
		if err != nil {
			log.Error().Err(err).Msgf("meme failed")
			w.WriteHeader(500)
			e, _ := json.Marshal(err)
			w.Write(e)
			return true
		}
		return false
	}
}

func (p *MemePlugin) rmMeme(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(405)
		fmt.Fprintf(w, "Incorrect HTTP method")
		return
	}
	checkError := mkCheckError(w)
	decoder := json.NewDecoder(r.Body)
	values := webResp{}
	err := decoder.Decode(&values)
	if checkError(err) {
		return
	}
	formats := p.c.GetMap("meme.memes", defaultFormats)
	delete(formats, values.Name)
	err = p.c.SetMap("meme.memes", formats)
	checkError(err)
}

func (p *MemePlugin) addMeme(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		fmt.Fprintf(w, "Incorrect HTTP method")
		return
	}
	checkError := mkCheckError(w)
	decoder := json.NewDecoder(r.Body)
	values := webResp{}
	err := decoder.Decode(&values)
	if checkError(err) {
		return
	}
	formats := p.c.GetMap("meme.memes", defaultFormats)
	formats[values.Name] = values.URL
	err = p.c.SetMap("meme.memes", formats)
	checkError(err)
}

func (p *MemePlugin) webRoot(w http.ResponseWriter, r *http.Request) {
	var tpl = template.Must(template.New("factoidIndex").Parse(string(memeIndex)))
	tpl.Execute(w, struct{ Nav []bot.EndPoint }{p.bot.GetWebNavigation()})
}

func (p *MemePlugin) img(w http.ResponseWriter, r *http.Request) {
	_, file := path.Split(r.URL.Path)
	id := file
	if img, ok := p.images[id]; ok {
		w.Write(img.repr)
	} else {
		w.WriteHeader(404)
		w.Write([]byte("not found"))
	}
	p.images.cleanup()
}

func (p *MemePlugin) slashMeme(c bot.Connector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		log.Debug().Msgf("Meme:\n%+v", r.PostForm.Get("text"))
		channel := r.PostForm.Get("channel_id")
		channelName := r.PostForm.Get("channel_name")
		from := r.PostForm.Get("user_name")
		text := r.PostForm.Get("text")
		log.Debug().Msgf("channel: %s", channel)

		parts := strings.SplitN(text, " ", 2)
		if len(parts) != 2 {
			log.Debug().Msgf("Bad meme request: %s, %s", from, text)
			p.bot.Send(c, bot.Message, channel, fmt.Sprintf("%s tried to send me a bad meme request.", from))
			return
		}
		isCmd, message := bot.IsCmd(p.c, parts[1])
		format := parts[0]

		bullyIcon := ""

		for _, bully := range p.c.GetArray("meme.bully", []string{}) {
			if format == bully {
				if u, err := c.Profile(bully); err == nil {
					bullyIcon = u.Icon
				} else {
					log.Debug().Err(err).Msgf("could not get profile for %s", format)
				}
				formats := p.c.GetMap("meme.memes", defaultFormats)
				format = randEntry(formats)
				break
			}
		}

		if u, err := c.Profile(from); bullyIcon == "" && err == nil {
			bullyIcon = u.Icon
		}

		log.Debug().Strs("parts", parts).Msgf("Meme:\n%+v", text)
		w.WriteHeader(200)
		w.Write(nil)

		go func() {
			top, bottom := "", message
			parts = strings.Split(message, "\n")
			if len(parts) > 1 {
				top, bottom = parts[0], parts[1]
			}

			if top == "_" {
				message = bottom
			} else if bottom == "_" {
				message = top
			}

			id, err := p.genMeme(format, top, bottom, bullyIcon)
			if err != nil {
				msg := fmt.Sprintf("Hey %s, I couldn't download that image you asked for.", from)
				p.bot.Send(c, bot.Message, channel, msg)
				return
			}
			baseURL := p.c.Get("BaseURL", `https://catbase.velour.ninja`)
			u, _ := url.Parse(baseURL)
			u.Path = path.Join(u.Path, "meme", "img", id)

			log.Debug().Msgf("image is at %s", u.String())
			p.bot.Send(c, bot.Message, channel, "", bot.ImageAttachment{
				URL:    u.String(),
				AltTxt: fmt.Sprintf("%s: %s", from, message),
			})
			m := msg.Message{
				User: &user.User{
					ID:    from,
					Name:  from,
					Admin: false,
				},
				Channel:     channel,
				ChannelName: channelName,
				Body:        message,
				Command:     isCmd,
				Time:        time.Now(),
			}

			p.bot.Receive(c, bot.Message, m)
		}()
	}
}

func randEntry(m map[string]string) string {
	for _, e := range m {
		return e
	}
	return ""
}

func DownloadTemplate(u *url.URL) (image.Image, error) {
	res, err := http.Get(u.String())
	if err != nil {
		log.Error().Msgf("template from %s failed because of %v", u.String(), err)
		return nil, err
	}
	defer res.Body.Close()
	image, _, err := image.Decode(res.Body)
	if err != nil {
		log.Error().Msgf("Could not decode %v because of %v", u, err)
		return nil, err
	}
	return image, nil
}

var defaultFormats = map[string]string{
	"fry":    "Futurama-Fry.jpg",
	"aliens": "Ancient-Aliens.jpg",
	"doge":   "Doge.jpg",
	"simply": "One-Does-Not-Simply.jpg",
	"wonka":  "Creepy-Condescending-Wonka.jpg",
	"grumpy": "Grumpy-Cat.jpg",
	"raptor": "Philosoraptor.jpg",
}

func (p *MemePlugin) genMeme(meme, top, bottom, bully string) (string, error) {
	fontSizes := []float64{48, 36, 24, 16, 12}
	fontSize := fontSizes[0]

	formats := p.c.GetMap("meme.memes", defaultFormats)

	path := uuid.New().String()

	imgName, ok := formats[meme]
	if !ok {
		imgName = meme
	}

	u, err := url.Parse(imgName)
	if err != nil || u.Scheme == "" {
		log.Debug().Err(err).Str("imgName", imgName).Msgf("url not detected")
		u, _ = url.Parse("https://imgflip.com/s/meme/" + imgName)
	}

	log.Debug().Msgf("Attempting to download url: %s", u.String())

	img, err := DownloadTemplate(u)
	if err != nil {
		log.Debug().Msgf("failed to download image: %s", err)
		return "", err
	}

	r := img.Bounds()
	w := r.Dx()
	h := r.Dy()

	maxSz := 750.0

	if w > h {
		scale := maxSz / float64(w)
		w = int(float64(w) * scale)
		h = int(float64(h) * scale)
	} else {
		scale := maxSz / float64(h)
		w = int(float64(w) * scale)
		h = int(float64(h) * scale)
	}

	log.Debug().Msgf("trynig to resize to %v, %v", w, h)
	img = resize.Resize(uint(w), uint(h), img, resize.Lanczos3)
	r = img.Bounds()
	w = r.Dx()
	h = r.Dy()
	log.Debug().Msgf("resized to %v, %v", w, h)

	if bully != "" {
		img = p.applyBully(img, bully)
	}

	m := gg.NewContext(w, h)
	m.DrawImage(img, 0, 0)
	fontLocation := p.c.Get("meme.font", "impact.ttf")
	for _, sz := range fontSizes {
		err = m.LoadFontFace(fontLocation, sz) // problem
		if err != nil {
			log.Error().Err(err).Msg("could not load font")
		}
		topW, _ := m.MeasureString(top)
		botW, _ := m.MeasureString(bottom)
		if topW < float64(w) && botW < float64(w) {
			fontSize = sz
			break
		}
	}

	if top == "_" {
		top = ""
	}
	if bottom == "_" {
		bottom = ""
	}

	// Apply black stroke
	m.SetHexColor("#000")
	strokeSize := 6
	for dy := -strokeSize; dy <= strokeSize; dy++ {
		for dx := -strokeSize; dx <= strokeSize; dx++ {
			// give it rounded corners
			if dx*dx+dy*dy >= strokeSize*strokeSize {
				continue
			}
			x := float64(w/2 + dx)
			y := float64(h) - fontSize + float64(dy)
			y0 := fontSize + float64(dy)
			m.DrawStringAnchored(top, x, y0, 0.5, 0.5)
			m.DrawStringAnchored(bottom, x, y, 0.5, 0.5)
		}
	}

	// Apply white fill
	m.SetHexColor("#FFF")
	m.DrawStringAnchored(top, float64(w)/2, fontSize, 0.5, 0.5)
	m.DrawStringAnchored(bottom, float64(w)/2, float64(h)-fontSize, 0.5, 0.5)

	i := bytes.Buffer{}
	png.Encode(&i, m.Image())
	p.images[path] = &cachedImage{time.Now(), i.Bytes()}

	log.Debug().Msgf("Saved to %s\n", path)

	return path, nil
}

func (p *MemePlugin) applyBully(img image.Image, bully string) image.Image {
	log.Debug().Msgf("applying bully: %s", bully)
	dst := image.NewRGBA(img.Bounds())
	u, err := url.Parse(bully)
	if err != nil {
		return img
	}
	bullyImg, err := DownloadTemplate(u)
	if err != nil {
		return img
	}

	scaleFactor := p.c.GetFloat64("meme.bullyScale", 0.1)

	scaleFactor = float64(img.Bounds().Max.X) * scaleFactor / float64(bullyImg.Bounds().Max.X)

	newSzX := uint(float64(bullyImg.Bounds().Max.X) * scaleFactor)
	newSzY := uint(float64(bullyImg.Bounds().Max.Y) * scaleFactor)

	bullyImg = resize.Resize(newSzX, newSzY, bullyImg, resize.Lanczos3)

	draw.Draw(dst, img.Bounds(), img, image.Point{}, draw.Src)
	srcSz := img.Bounds().Size()

	w, h := bullyImg.Bounds().Max.X, bullyImg.Bounds().Max.Y

	pt := image.Point{srcSz.X - w, srcSz.Y - h}
	rect := image.Rect(pt.X, pt.Y, srcSz.X, srcSz.Y)

	draw.DrawMask(dst, rect, bullyImg, image.Point{}, &circle{image.Point{w / 2, h / 2}, w / 2}, image.Point{}, draw.Over)
	return dst
}

// the following is ripped off of https://blog.golang.org/image-draw
type circle struct {
	p image.Point
	r int
}

func (c *circle) ColorModel() color.Model {
	return color.AlphaModel
}

func (c *circle) Bounds() image.Rectangle {
	return image.Rect(c.p.X-c.r, c.p.Y-c.r, c.p.X+c.r, c.p.Y+c.r)
}

func (c *circle) At(x, y int) color.Color {
	xx, yy, rr := float64(x-c.p.X)+0.5, float64(y-c.p.Y)+0.5, float64(c.r)
	if xx*xx+yy*yy < rr*rr {
		return color.Alpha{255}
	}
	return color.Alpha{0}
}
