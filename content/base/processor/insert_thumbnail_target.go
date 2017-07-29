package processor

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/urandom/readeef"
	"github.com/urandom/readeef/content"
)

type InsertThumbnailTarget struct {
	log         readeef.Logger
	urlTemplate *template.Template
}

func NewInsertThumbnailTarget(l readeef.Logger) InsertThumbnailTarget {
	return InsertThumbnailTarget{log: l}
}

func (p InsertThumbnailTarget) ProcessArticles(ua []content.UserArticle) []content.UserArticle {
	if len(ua) == 0 {
		return ua
	}

	p.log.Infof("Proxying urls of feed '%d'\n", ua[0].Data().FeedId)

	for i := range ua {
		data := ua[i].Data()

		if data.ThumbnailLink == "" {
			continue
		}

		if d, err := goquery.NewDocumentFromReader(strings.NewReader(data.Description)); err == nil {
			if insertThumbnailTarget(d, data.ThumbnailLink, p.log) {
				if content, err := d.Html(); err == nil {
					// net/http tries to provide valid html, adding html, head and body tags
					content = content[strings.Index(content, "<body>")+6 : strings.LastIndex(content, "</body>")]

					data.Description = content
					ua[i].Data(data)
				}
			}
		}
	}

	return ua
}

func insertThumbnailTarget(d *goquery.Document, thumbnailLink string, log readeef.Logger) bool {
	changed := false

	if d.Find(".top-image").Length() > 0 {
		return changed
	}

	thumbDoc, err := goquery.NewDocumentFromReader(strings.NewReader(fmt.Sprintf(`<img src="%s">`, thumbnailLink)))
	if err != nil {
		log.Infof("Error generating thumbnail image node: %v\n", err)
		return changed
	}

	d.Find("body").PrependSelection(thumbDoc.Find("img"))
	changed = true

	return changed
}
