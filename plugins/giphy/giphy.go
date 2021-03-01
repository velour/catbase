package giphy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
)

type GiphyPlugin struct {
	b bot.Bot
	c *config.Config

	handlers bot.HandlerTable
}

func New(b bot.Bot) *GiphyPlugin {
	g := &GiphyPlugin{
		b: b,
		c: b.Config(),
	}
	g.register()
	return g
}

func (p *GiphyPlugin) register() {
	p.handlers = bot.HandlerTable{
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?i)^giph (?P<search>\S+)$`),
			HelpText: "search for a giph",
			Handler: func(r bot.Request) bool {
				u, w, h := p.query(r.Values["search"])
				p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "", bot.ImageAttachment{
					URL:    u,
					AltTxt: fmt.Sprintf("%s: %s\nPowered by Giphy", r.Msg.User.Name, r.Values["search"]),
					Width:  w,
					Height: h,
				})
				return true
			}},
	}
	p.b.RegisterTable(p, p.handlers)
	log.Debug().Msg("registering giph")
}

func (p *GiphyPlugin) query(search string) (string, int, int) {
	key := p.c.Get("GIPHYKEY", "NONE")
	if key == "NONE" {
		return "", 0, 0
	}
	u, _ := url.Parse("https://api.giphy.com/v1/gifs/search")
	values := u.Query()
	values.Set("api_key", key)
	values.Set("q", search)
	u.RawQuery = values.Encode()
	resp, err := http.Get(u.String())
	if err != nil {
		log.Error().Err(err).Msg("could not get giph")
		return "", 0, 0
	}
	searchData := searchResponse{}
	data, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(data, &searchData)
	if err != nil {
		log.Error().Err(err).Msg("could not decode giph")
		return "", 0, 0
	}
	value := searchData.Data[rand.Intn(len(searchData.Data))]
	downsized := value.Images.DownsizedMedium
	w, _ := strconv.Atoi(downsized.Width)
	h, _ := strconv.Atoi(downsized.Height)
	return downsized.URL, w, h
}

type searchResponse struct {
	Data []struct {
		Type             string `json:"type"`
		ID               string `json:"id"`
		URL              string `json:"url"`
		Slug             string `json:"slug"`
		BitlyGifURL      string `json:"bitly_gif_url"`
		BitlyURL         string `json:"bitly_url"`
		EmbedURL         string `json:"embed_url"`
		Username         string `json:"username"`
		Source           string `json:"source"`
		Title            string `json:"title"`
		Rating           string `json:"rating"`
		ContentURL       string `json:"content_url"`
		SourceTld        string `json:"source_tld"`
		SourcePostURL    string `json:"source_post_url"`
		IsSticker        int    `json:"is_sticker"`
		ImportDatetime   string `json:"import_datetime"`
		TrendingDatetime string `json:"trending_datetime"`
		Images           struct {
			Original struct {
				Height   string `json:"height"`
				Width    string `json:"width"`
				Size     string `json:"size"`
				URL      string `json:"url"`
				Mp4Size  string `json:"mp4_size"`
				Mp4      string `json:"mp4"`
				WebpSize string `json:"webp_size"`
				Webp     string `json:"webp"`
				Frames   string `json:"frames"`
				Hash     string `json:"hash"`
			} `json:"original"`
			Downsized struct {
				Height string `json:"height"`
				Width  string `json:"width"`
				Size   string `json:"size"`
				URL    string `json:"url"`
			} `json:"downsized"`
			DownsizedLarge struct {
				Height string `json:"height"`
				Width  string `json:"width"`
				Size   string `json:"size"`
				URL    string `json:"url"`
			} `json:"downsized_large"`
			DownsizedMedium struct {
				Height string `json:"height"`
				Width  string `json:"width"`
				Size   string `json:"size"`
				URL    string `json:"url"`
			} `json:"downsized_medium"`
			DownsizedSmall struct {
				Height  string `json:"height"`
				Width   string `json:"width"`
				Mp4Size string `json:"mp4_size"`
				Mp4     string `json:"mp4"`
			} `json:"downsized_small"`
			DownsizedStill struct {
				Height string `json:"height"`
				Width  string `json:"width"`
				Size   string `json:"size"`
				URL    string `json:"url"`
			} `json:"downsized_still"`
			FixedHeight struct {
				Height   string `json:"height"`
				Width    string `json:"width"`
				Size     string `json:"size"`
				URL      string `json:"url"`
				Mp4Size  string `json:"mp4_size"`
				Mp4      string `json:"mp4"`
				WebpSize string `json:"webp_size"`
				Webp     string `json:"webp"`
			} `json:"fixed_height"`
			FixedHeightDownsampled struct {
				Height   string `json:"height"`
				Width    string `json:"width"`
				Size     string `json:"size"`
				URL      string `json:"url"`
				WebpSize string `json:"webp_size"`
				Webp     string `json:"webp"`
			} `json:"fixed_height_downsampled"`
			FixedHeightSmall struct {
				Height   string `json:"height"`
				Width    string `json:"width"`
				Size     string `json:"size"`
				URL      string `json:"url"`
				Mp4Size  string `json:"mp4_size"`
				Mp4      string `json:"mp4"`
				WebpSize string `json:"webp_size"`
				Webp     string `json:"webp"`
			} `json:"fixed_height_small"`
			FixedHeightSmallStill struct {
				Height string `json:"height"`
				Width  string `json:"width"`
				Size   string `json:"size"`
				URL    string `json:"url"`
			} `json:"fixed_height_small_still"`
			FixedHeightStill struct {
				Height string `json:"height"`
				Width  string `json:"width"`
				Size   string `json:"size"`
				URL    string `json:"url"`
			} `json:"fixed_height_still"`
			FixedWidth struct {
				Height   string `json:"height"`
				Width    string `json:"width"`
				Size     string `json:"size"`
				URL      string `json:"url"`
				Mp4Size  string `json:"mp4_size"`
				Mp4      string `json:"mp4"`
				WebpSize string `json:"webp_size"`
				Webp     string `json:"webp"`
			} `json:"fixed_width"`
			FixedWidthDownsampled struct {
				Height   string `json:"height"`
				Width    string `json:"width"`
				Size     string `json:"size"`
				URL      string `json:"url"`
				WebpSize string `json:"webp_size"`
				Webp     string `json:"webp"`
			} `json:"fixed_width_downsampled"`
			FixedWidthSmall struct {
				Height   string `json:"height"`
				Width    string `json:"width"`
				Size     string `json:"size"`
				URL      string `json:"url"`
				Mp4Size  string `json:"mp4_size"`
				Mp4      string `json:"mp4"`
				WebpSize string `json:"webp_size"`
				Webp     string `json:"webp"`
			} `json:"fixed_width_small"`
			FixedWidthSmallStill struct {
				Height string `json:"height"`
				Width  string `json:"width"`
				Size   string `json:"size"`
				URL    string `json:"url"`
			} `json:"fixed_width_small_still"`
			FixedWidthStill struct {
				Height string `json:"height"`
				Width  string `json:"width"`
				Size   string `json:"size"`
				URL    string `json:"url"`
			} `json:"fixed_width_still"`
			Looping struct {
				Mp4Size string `json:"mp4_size"`
				Mp4     string `json:"mp4"`
			} `json:"looping"`
			OriginalStill struct {
				Height string `json:"height"`
				Width  string `json:"width"`
				Size   string `json:"size"`
				URL    string `json:"url"`
			} `json:"original_still"`
			OriginalMp4 struct {
				Height  string `json:"height"`
				Width   string `json:"width"`
				Mp4Size string `json:"mp4_size"`
				Mp4     string `json:"mp4"`
			} `json:"original_mp4"`
			Preview struct {
				Height  string `json:"height"`
				Width   string `json:"width"`
				Mp4Size string `json:"mp4_size"`
				Mp4     string `json:"mp4"`
			} `json:"preview"`
			PreviewGif struct {
				Height string `json:"height"`
				Width  string `json:"width"`
				Size   string `json:"size"`
				URL    string `json:"url"`
			} `json:"preview_gif"`
			PreviewWebp struct {
				Height string `json:"height"`
				Width  string `json:"width"`
				Size   string `json:"size"`
				URL    string `json:"url"`
			} `json:"preview_webp"`
			Four80WStill struct {
				Height string `json:"height"`
				Width  string `json:"width"`
				Size   string `json:"size"`
				URL    string `json:"url"`
			} `json:"480w_still"`
		} `json:"images"`
		AnalyticsResponsePayload string `json:"analytics_response_payload"`
		Analytics                struct {
			Onload struct {
				URL string `json:"url"`
			} `json:"onload"`
			Onclick struct {
				URL string `json:"url"`
			} `json:"onclick"`
			Onsent struct {
				URL string `json:"url"`
			} `json:"onsent"`
		} `json:"analytics"`
		User struct {
			AvatarURL    string `json:"avatar_url"`
			BannerImage  string `json:"banner_image"`
			BannerURL    string `json:"banner_url"`
			ProfileURL   string `json:"profile_url"`
			Username     string `json:"username"`
			DisplayName  string `json:"display_name"`
			Description  string `json:"description"`
			InstagramURL string `json:"instagram_url"`
			WebsiteURL   string `json:"website_url"`
			IsVerified   bool   `json:"is_verified"`
		} `json:"user,omitempty"`
	} `json:"data"`
	Pagination struct {
		TotalCount int `json:"total_count"`
		Count      int `json:"count"`
		Offset     int `json:"offset"`
	} `json:"pagination"`
	Meta struct {
		Status     int    `json:"status"`
		Msg        string `json:"msg"`
		ResponseID string `json:"response_id"`
	} `json:"meta"`
}
