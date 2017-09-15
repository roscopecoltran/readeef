package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"github.com/urandom/handler/auth"
	"github.com/urandom/handler/method"
	"github.com/urandom/readeef"
	"github.com/urandom/readeef/api/fever"
	"github.com/urandom/readeef/api/token"
	"github.com/urandom/readeef/api/ttrss"
	"github.com/urandom/readeef/config"
	"github.com/urandom/readeef/content/extract"
	"github.com/urandom/readeef/content/processor"
	"github.com/urandom/readeef/content/repo"
	"github.com/urandom/readeef/content/search"
	"github.com/urandom/readeef/log"
)

func Mux(
	ctx context.Context,
	service repo.Service,
	feedManager *readeef.FeedManager,
	hubbub *readeef.Hubbub,
	searchProvider search.Provider,
	extractor extract.Generator,
	fs http.FileSystem,
	processors []processor.Article,
	config config.Config,
	log log.Log,
) (http.Handler, error) {

	languageSupport := false
	if languages, err := readeef.GetLanguages(fs); err == nil {
		languageSupport = len(languages) > 0
	}

	features := features{
		I18N:       languageSupport,
		Popularity: len(config.Popularity.Providers) > 0,
		ProxyHTTP:  hasProxy(config),
		Search:     searchProvider != nil,
		Extractor:  extractor != nil,
	}

	storage, err := initTokenStorage(config.Auth)
	if err != nil {
		return nil, errors.Wrap(err, "initializing token storage")
	}

	routes := []routes{tokenRoutes(service.UserRepo(), storage, []byte(config.Auth.Secret), log)}

	if hubbub != nil {
		routes = append(routes, hubbubRoutes(hubbub, service, log))
	}

	emulatorRoutes := emulatorRoutes(ctx, service, searchProvider, feedManager, processors, config, log)
	routes = append(routes, emulatorRoutes...)

	routes = append(routes, mainRoutes(
		userMiddleware(service.UserRepo(), storage, []byte(config.Auth.Secret), log),
		featureRoutes(features),
		feedsRoutes(service, feedManager, log),
		tagRoutes(service.TagRepo(), log),
		articlesRoutes(service, extractor, searchProvider, processors, config, log),
		opmlRoutes(service, feedManager, log),
		eventsRoutes(ctx, service, storage, feedManager, log),
		userRoutes(service, []byte(config.Auth.Secret), log),
	))

	r := chi.NewRouter()

	r.Route("/v2", func(r chi.Router) {
		for _, sub := range routes {
			r.Route(sub.path, sub.route)
		}
	})

	return r, nil
}

func hasProxy(config config.Config) bool {
	for _, p := range config.Content.Article.Processors {
		if p == "proxy-http" {
			return true
		}
	}

	for _, p := range config.FeedParser.Processors {
		if p == "proxy-http" {
			return true
		}
	}

	return false
}

func initTokenStorage(config config.Auth) (token.Storage, error) {
	if err := os.MkdirAll(filepath.Dir(config.TokenStoragePath), 0777); err != nil {
		return nil, errors.Wrapf(err, "creating token storage path %s", config.TokenStoragePath)
	}

	return token.NewBoltStorage(config.TokenStoragePath)
}

type routes struct {
	path  string
	route func(r chi.Router)
}

func timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, d, "")
	}
}

func tokenRoutes(repo repo.User, storage token.Storage, secret []byte, log log.Log) routes {
	return routes{path: "/token", route: func(r chi.Router) {
		r.Use(timeout(time.Second))
		r.Method(method.POST, "/", tokenCreate(repo, secret, log))
		r.Method(method.DELETE, "/", tokenDelete(storage, secret, log))
	}}
}

func hubbubRoutes(hubbub *readeef.Hubbub, service repo.Service, log log.Log) routes {
	handler := hubbubRegistration(hubbub, service, log)

	return routes{path: "/hubbub", route: func(r chi.Router) {
		r.Use(timeout(5 * time.Second))
		r.Get("/", handler)
		r.Post("/", handler)
	}}
}

func emulatorRoutes(
	ctx context.Context,
	service repo.Service,
	searchProvider search.Provider,
	feedManager *readeef.FeedManager,
	processors []processor.Article,
	config config.Config,
	log log.Log,
) []routes {
	rr := make([]routes, 0, len(config.API.Emulators))

	for _, e := range config.API.Emulators {
		switch e {
		case "tt-rss":
			rr = append(rr, routes{
				path: fmt.Sprintf("/v%d/tt-rss/", ttrss.API_LEVEL),
				route: func(r chi.Router) {
					r.Use(timeout(5 * time.Second))
					r.Get("/", ttrss.FakeWebHandler)

					r.Post("/api/", ttrss.Handler(
						ctx, service, searchProvider, feedManager, processors,
						[]byte(config.Auth.Secret), config.FeedManager.Converted.UpdateInterval,
						log,
					))
				},
			})
		case "fever":
			rr = append(rr, routes{
				path: fmt.Sprintf("/v%d/fever/", fever.API_VERSION),
				route: func(r chi.Router) {
					r.Use(timeout(5 * time.Second))
					r.Post("/", fever.Handler(service, processors, log))
				},
			})
		}
	}

	return rr
}

type middleware func(next http.Handler) http.Handler

func mainRoutes(middleware []middleware, subroutes ...routes) routes {
	return routes{path: "/", route: func(r chi.Router) {
		for _, m := range middleware {
			r.Use(m)
		}

		for _, sub := range subroutes {
			r.Route(sub.path, sub.route)
		}
	}}
}

func userMiddleware(repo repo.User, storage token.Storage, secret []byte, log log.Log) []middleware {
	return []middleware{
		func(next http.Handler) http.Handler {
			return auth.RequireToken(next, tokenValidator(repo, storage, log), secret)
		},
		func(next http.Handler) http.Handler {
			return userContext(repo, next, log)
		},
		userValidator,
	}
}

func featureRoutes(features features) routes {
	return routes{path: "/features", route: func(r chi.Router) {
		r.Use(timeout(time.Second))
		r.Get("/", featuresHandler(features))
	}}
}

func feedsRoutes(service repo.Service, feedManager *readeef.FeedManager, log log.Log) routes {
	return routes{path: "/feed", route: func(r chi.Router) {
		feedRepo := service.FeedRepo()
		r.Use(timeout(5 * time.Second))
		r.Get("/", listFeeds(feedRepo, log))
		r.Post("/", addFeed(feedRepo, feedManager))

		r.Get("/discover", discoverFeeds(feedRepo, feedManager, log))

		r.Route("/{feedID:[0-9]+}", func(r chi.Router) {
			r.Use(feedContext(service.FeedRepo(), log))

			r.Delete("/", deleteFeed(feedRepo, feedManager, log))

			r.Get("/tags", getFeedTags(service.TagRepo(), log))
			r.Post("/tags", setFeedTags(feedRepo, log))

		})
	}}
}

func tagRoutes(repo repo.Tag, log log.Log) routes {
	return routes{path: "/tag", route: func(r chi.Router) {
		r.Use(timeout(5 * time.Second))
		r.Get("/", listTags(repo, log))
		r.Get("/feedIDs", getTagsFeedIDs(repo, log))
	}}
}

func articlesRoutes(
	service repo.Service,
	extractor extract.Generator,
	searchProvider search.Provider,
	processors []processor.Article,
	config config.Config,
	log log.Log,
) routes {
	articleRepo := service.ArticleRepo()
	feedRepo := service.FeedRepo()
	tagRepo := service.TagRepo()

	return routes{path: "/article", route: func(r chi.Router) {
		r.Use(timeout(10 * time.Second))
		r.Get("/", getArticles(service, userRepoType, noRepoType, processors, config.API.Limits.ArticlesPerQuery, log))

		if searchProvider != nil {
			r.Route("/search", func(r chi.Router) {
				r.Get("/",
					articleSearch(tagRepo, searchProvider, userRepoType, processors, config.API.Limits.ArticlesPerQuery, log))
				r.With(feedContext(feedRepo, log)).Get("/feed/{feedID:[0-9]+}/",
					articleSearch(tagRepo, searchProvider, feedRepoType, processors, config.API.Limits.ArticlesPerQuery, log))
				r.With(tagContext(tagRepo, log)).Get("/tag/{tagID:[0-9]+}/",
					articleSearch(tagRepo, searchProvider, tagRepoType, processors, config.API.Limits.ArticlesPerQuery, log))
			})
		}

		r.Post("/read", articlesReadStateChange(service, userRepoType, config.API.Limits.ArticlesPerQuery, log))

		r.Route("/{articleID:[0-9]+}", func(r chi.Router) {
			r.Use(articleContext(articleRepo, processors, log))

			r.Get("/", getArticle)
			if extractor != nil {
				r.Get("/format", formatArticle(service.ExtractRepo(), extractor, processors, log))
			}
			r.Post("/read", articleStateChange(articleRepo, read, log))
			r.Delete("/read", articleStateChange(articleRepo, read, log))
			r.Post("/favorite", articleStateChange(articleRepo, favorite, log))
			r.Delete("/favorite", articleStateChange(articleRepo, favorite, log))
		})

		r.Route("/favorite", func(r chi.Router) {
			r.Get("/", getArticles(service, favoriteRepoType, noRepoType, processors, config.API.Limits.ArticlesPerQuery, log))

			r.Post("/read", articlesReadStateChange(service, favoriteRepoType, config.API.Limits.ArticlesPerQuery, log))
		})

		r.Route("/popular", func(r chi.Router) {
			r.With(feedContext(feedRepo, log)).Get("/feed/{feedID:[0-9]+}",
				getArticles(service, popularRepoType, feedRepoType, processors, config.API.Limits.ArticlesPerQuery, log))
			r.With(tagContext(tagRepo, log)).Get("/tag/{tagID:[0-9]+}",
				getArticles(service, popularRepoType, tagRepoType, processors, config.API.Limits.ArticlesPerQuery, log))
			r.Get("/", getArticles(service, popularRepoType, userRepoType, processors, config.API.Limits.ArticlesPerQuery, log))
		})

		r.Route("/feed/{feedID:[0-9]+}", func(r chi.Router) {
			r.Use(feedContext(feedRepo, log))

			r.Get("/", getArticles(service, feedRepoType, noRepoType, processors, config.API.Limits.ArticlesPerQuery, log))

			r.Post("/read", articlesReadStateChange(service, feedRepoType, config.API.Limits.ArticlesPerQuery, log))
		})

		r.Route("/tag/{tagID:[0-9]+}", func(r chi.Router) {
			r.Use(tagContext(tagRepo, log))

			r.Get("/", getArticles(service, tagRepoType, noRepoType, processors, config.API.Limits.ArticlesPerQuery, log))

			r.Post("/read", articlesReadStateChange(service, tagRepoType, config.API.Limits.ArticlesPerQuery, log))
		})

	}}
}

func opmlRoutes(service repo.Service, feedManager *readeef.FeedManager, log log.Log) routes {
	return routes{path: "/opml", route: func(r chi.Router) {
		r.Use(timeout(10 * time.Second))
		r.Get("/", exportOPML(service, feedManager, log))
		r.Post("/", importOPML(service.FeedRepo(), feedManager, log))
	}}
}

func eventsRoutes(
	ctx context.Context,
	service repo.Service,
	storage token.Storage,
	feedManager *readeef.FeedManager,
	log log.Log,
) routes {
	return routes{path: "/events", route: func(r chi.Router) {
		r.Get("/", eventSocket(ctx, service.FeedRepo(), storage, feedManager, log))
	}}
}

func userRoutes(service repo.Service, secret []byte, log log.Log) routes {
	repo := service.UserRepo()
	return routes{path: "/user", route: func(r chi.Router) {
		r.Use(timeout(5 * time.Second))

		r.Get("/current", getUserData)

		r.Route("/settings", func(r chi.Router) {
			r.Get("/", getSettingKeys)
			r.Get("/{key}", getSettingValue)
			r.Put("/{key}", setSettingValue(repo, secret, log))
		})

		r.Route("/", func(r chi.Router) {
			r.Use(adminValidator)

			r.Get("/", listUsers(repo, log))

			r.Post("/", addUser(repo, secret, log))
			r.Delete("/{name}", deleteUser(repo, log))

			r.Post("/{name}/settings/{key}", setSettingValue(repo, secret, log))
		})
	}}
}

func readJSON(w http.ResponseWriter, r io.Reader, data interface{}) (stop bool) {
	if b, err := ioutil.ReadAll(r); err == nil {
		if err = json.Unmarshal(b, data); err != nil {
			http.Error(w, "Error decoding JSON request: "+err.Error(), http.StatusBadRequest)
			return true
		}
	} else {
		http.Error(w, "Error reading request body: "+err.Error(), http.StatusInternalServerError)
		return true
	}

	return false
}

func fatal(w http.ResponseWriter, log log.Log, format string, err error) {
	log.Printf(format, err)
	http.Error(w, fmt.Sprintf(format, err.Error()), http.StatusInternalServerError)
}
