package thumbnailer

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/urandom/readeef"
	"github.com/urandom/readeef/content"
	"github.com/urandom/readeef/content/data"
)

type Description struct {
	log readeef.Logger
}

func NewDescription(l readeef.Logger) content.Thumbnailer {
	return Description{log: l}
}

func (t Description) Generate(a content.Article) error {
	ad := a.Data()

	thumbnail := a.Repo().ArticleThumbnail()
	td := data.ArticleThumbnail{
		ArticleId: ad.Id,
		Processed: true,
	}

	t.log.Debugf("Generating thumbnail for article %s\n", a)

	td.Thumbnail, td.Link =
		generateThumbnailFromDescription(strings.NewReader(ad.Description))

	thumbnail.Data(td)
	if thumbnail.Update(); thumbnail.HasErr() {
		return errors.Wrapf(thumbnail.Err(), "Error saving thumbnail of %s to database", a)
	}

	return nil
}
