package readeef

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/urandom/readeef/config"
	"github.com/urandom/readeef/content"
	"github.com/urandom/readeef/content/data"
	"github.com/urandom/webfw/util"
)

type Hubbub struct {
	config        config.Config
	repo          content.Repo
	endpoint      string
	removeFeed    chan<- content.Feed
	subscribe     chan content.Subscription
	unsubscribe   chan content.Subscription
	client        *http.Client
	log           Logger
	subscriptions []content.Subscription
	feedMonitors  []content.FeedMonitor
}

type SubscriptionError struct {
	error
	Subscription content.Subscription
}

var (
	ErrNotConfigured = errors.New("Hubbub callback URL is not set")
	ErrNoFeedHubLink = errors.New("Feed does not contain a hub link")
	ErrSubscribed    = errors.New("Feed already subscribed")
	ErrNotSubscribed = errors.New("Feed is not subscribed")
)

func NewHubbub(repo content.Repo, c config.Config, l Logger, endpoint string,
	removeFeed chan<- content.Feed) *Hubbub {

	return &Hubbub{
		repo: repo, config: c, log: l, endpoint: endpoint,
		removeFeed: removeFeed,
		subscribe:  make(chan content.Subscription), unsubscribe: make(chan content.Subscription),
		client: NewTimeoutClient(c.Timeout.Converted.Connect, c.Timeout.Converted.ReadWrite)}
}

func (h *Hubbub) Client(c ...*http.Client) *http.Client {
	if len(c) > 0 {
		h.client = c[0]
	}

	return h.client
}

func (h *Hubbub) FeedMonitors(m ...[]content.FeedMonitor) []content.FeedMonitor {
	if len(m) > 0 {
		h.feedMonitors = m[0]
	}

	return h.feedMonitors
}

func (h Hubbub) Subscribe(f content.Feed) error {
	if u, err := url.Parse(h.config.Hubbub.CallbackURL); err != nil {
		return ErrNotConfigured
	} else {
		if !u.IsAbs() {
			return ErrNotConfigured
		}
	}

	fdata := f.Data()
	if u, err := url.Parse(fdata.HubLink); err != nil {
		return ErrNoFeedHubLink
	} else {
		if !u.IsAbs() {
			return ErrNoFeedHubLink
		}
	}

	s := f.Subscription()
	if s.HasErr() {
		return s.Err()
	}

	data := s.Data()
	if data.FeedId == fdata.Id {
		h.log.Infoln("Already subscribed to " + fdata.HubLink)
		return ErrSubscribed
	}

	data.Link = fdata.HubLink
	data.FeedId = fdata.Id
	data.SubscriptionFailure = true

	s.Data(data)
	s.Update()

	if s.HasErr() {
		return s.Err()
	}

	go func() {
		h.subscription(s, f, true)
	}()

	return nil
}

func (h Hubbub) Unsubscribe(f content.Feed) error {
	if u, err := url.Parse(h.config.Hubbub.CallbackURL); err != nil {
		return ErrNotConfigured
	} else {
		if !u.IsAbs() {
			return ErrNotConfigured
		}
	}

	fdata := f.Data()
	if u, err := url.Parse(fdata.HubLink); err != nil {
		return ErrNoFeedHubLink
	} else {
		if !u.IsAbs() {
			return ErrNoFeedHubLink
		}
	}

	s := f.Subscription()
	if s.HasErr() {
		return s.Err()
	}

	if s.Data().FeedId != fdata.Id {
		h.log.Infoln("Not subscribed to " + fdata.HubLink)
		return ErrNotSubscribed
	}

	go func() {
		h.subscription(s, f, false)
	}()
	return nil
}

func (h Hubbub) InitSubscriptions() error {
	h.repo.FailSubscriptions()
	subscriptions := h.repo.AllSubscriptions()

	h.log.Infof("Initializing %d hubbub subscriptions", len(subscriptions))

	go func() {
		for _, s := range subscriptions {
			f := h.repo.FeedById(s.Data().FeedId)
			if f.Err() != nil {
				continue
			}

			h.subscription(s, f, true)
		}
	}()

	go func() {
		after := time.After(h.config.FeedManager.Converted.UpdateInterval)
		subscriptions := []content.Subscription{}

		for {
			select {
			case s := <-h.subscribe:
				subscriptions = append(subscriptions, s)
			case s := <-h.unsubscribe:
				filtered := []content.Subscription{}
				for i := range subscriptions {
					if subscriptions[i] != s {
						filtered = append(filtered, subscriptions[i])
					}
				}

				subscriptions = filtered
			case <-after:
				for _, s := range subscriptions {
					if s.Data().VerificationTime.Add(time.Duration(s.Data().LeaseDuration)).Before(time.Now().Add(-30 * time.Minute)) {
						f := h.repo.FeedById(s.Data().FeedId)
						if f.Err() != nil {
							continue
						}

						h.log.Infof("Renewing subscription to %s\n", s)
						h.subscription(s, f, true)
					}
				}
			}
		}
	}()

	if h.repo.HasErr() {
		return errors.Wrap(h.repo.Err(), "initializing subscriptions")
	}

	return nil
}

func (h Hubbub) subscription(s content.Subscription, f content.Feed, subscribe bool) {
	var err error

	fdata := f.Data()
	u := callbackURL(h.config, h.endpoint, fdata.Id)

	body := url.Values{}
	body.Set("hub.callback", u)
	if subscribe {
		h.log.Infoln("Subscribing to hubbub for " + f.String() + " with url " + u)
		body.Set("hub.mode", "subscribe")
	} else {
		h.log.Infoln("Unsubscribing to hubbub for " + f.String() + " with url " + u)
		body.Set("hub.mode", "unsubscribe")
	}
	body.Set("hub.topic", fdata.Link)

	buf := util.BufferPool.GetBuffer()
	defer util.BufferPool.Put(buf)

	buf.WriteString(body.Encode())
	req, _ := http.NewRequest("POST", s.Data().Link, buf)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("From", h.config.Hubbub.From)

	resp, err := h.client.Do(req)

	if err != nil {
		err = SubscriptionError{error: err, Subscription: s}
	} else if resp.StatusCode != 202 {
		err = SubscriptionError{error: errors.New("Expected response status 202, got " + resp.Status), Subscription: s}
	}

	if err == nil {
		if subscribe {
			h.subscribe <- s
		} else {
			h.unsubscribe <- s
		}
	} else {
		fdata.SubscribeError = err.Error()
		h.log.Printf("Error subscribing to hub feed '%s': %s\n", f, err)

		f.Data(fdata)
		f.Update()
		if f.HasErr() {
			h.log.Printf("Error updating feed database record for '%s': %s\n", f, f.Err())
		}

		h.removeFeed <- f
	}
}

func callbackURL(c config.Config, endpoint string, feedId data.FeedId) string {
	return fmt.Sprintf("%s%s/%d", c.Hubbub.CallbackURL, endpoint, feedId)
}
